package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/applejuicenetz/ajnx/internal/core"
	"github.com/applejuicenetz/ajnx/internal/gateway"
	"github.com/applejuicenetz/ajnx/internal/gateway/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server stellt den API-Server dar.
type Server struct {
	router        *chi.Mux
	state         *gateway.State
	sm            *auth.SessionManager
	client        core.CoreClient
	internalToken string
	sessionSecret string
}

// NewServer erstellt einen neuen API-Server.
// internalToken: Service-Token für UI-Server → Gateway Kommunikation.
// sessionSecret: JWT-Signier-Secret aus setup.lock (leer = Setup nicht abgeschlossen).
func NewServer(state *gateway.State, sm *auth.SessionManager, client core.CoreClient, internalToken, sessionSecret string) *Server {
	s := &Server{
		router:        chi.NewRouter(),
		state:         state,
		sm:            sm,
		client:        client,
		internalToken: internalToken,
		sessionSecret: sessionSecret,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// Router gibt den internen chi-Router zurück.
func (s *Server) Router() *chi.Mux {
	return s.router
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.RealIP)
}

func (s *Server) setupRoutes() {
	// Login: öffentlich, aber rate-limited (max 5 Versuche/Minute pro IP)
	s.router.With(loginRateLimiter()).Post("/api/auth/login", s.handleLogin)

	// Geschützte Endpunkte
	s.router.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware(s.sm, s.internalToken))

		r.Get("/api/status", s.handleStatus)
		r.Get("/api/downloads", s.handleDownloads)
		r.Get("/api/uploads", s.handleUploads)
		r.Post("/api/downloads/{id}/pause", s.handleDownloadPause)
		r.Post("/api/downloads/{id}/resume", s.handleDownloadResume)
		r.Post("/api/downloads/{id}/cancel", s.handleDownloadCancel)
		r.Post("/api/downloads/clean", s.handleCleanDownloads)
		r.Get("/api/servers", s.handleServers)
		r.Post("/api/servers/{id}/connect", s.handleServerConnect)
		r.Post("/api/servers/disconnect", s.handleServerDisconnect)
		r.Post("/api/servers/{id}/remove", s.handleServerRemove)
		r.Post("/api/core/shutdown", s.handleCoreShutdown)
		r.Post("/api/links", s.handleProcessLink)
		r.Get("/api/search", s.handleSearchGet)
		r.Post("/api/search", s.handleSearchStart)
		r.Post("/api/search/{id}/cancel", s.handleSearchCancel)
		r.Get("/api/settings", s.handleSettingsGet)
		r.Post("/api/settings", s.handleSettingsUpdate)
	})
}

// ipBucket zählt Versuche pro IP innerhalb eines Zeitfensters.
type ipBucket struct {
	count     int
	windowEnd time.Time
}

// loginRateLimiter gibt Middleware zurück, die Login-Versuche auf 5/Minute pro IP begrenzt.
func loginRateLimiter() func(http.Handler) http.Handler {
	const maxAttempts = 5
	const window = time.Minute

	var mu sync.Mutex
	buckets := make(map[string]*ipBucket)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			mu.Lock()
			b, ok := buckets[ip]
			now := time.Now()
			if !ok || now.After(b.windowEnd) {
				b = &ipBucket{count: 0, windowEnd: now.Add(window)}
				buckets[ip] = b
			}
			b.count++
			over := b.count > maxAttempts
			mu.Unlock()

			if over {
				http.Error(w, "Zu viele Anmeldeversuche – bitte in einer Minute erneut versuchen", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
