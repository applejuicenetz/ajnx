package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/applejuicenetz/ajnx/internal/setup"
	"github.com/applejuicenetz/ajnx/internal/ui/handler"
)

func main() {
	port := getEnv("UI_PORT", "8080")
	gatewayURL := getEnv("UI_GATEWAY_URL", "http://localhost:9000")
	gatewayToken := getEnv("INTERNAL_API_KEY", "")

	lockPath := getEnv("SETUP_LOCK_PATH", "./data/setup.lock")
	fs := http.FileServer(http.Dir("web/static"))

	lockDir := filepath.Dir(lockPath)
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		log.Fatalf(
			"Daten-Verzeichnis %q nicht beschreibbar: %v\n"+
				"Lösung: sudo chown -R $(whoami) %s\n"+
				"        oder: SETUP_LOCK_PATH=/tmp/ajnx-setup.lock",
			lockDir, err, lockDir,
		)
	}

	setupMgr, err := setup.NewManager(lockPath)
	if err != nil {
		log.Fatal("Setup Manager Fehler:", err)
	}

	nickname := ""
	sessionSecret := ""
	if setupMgr.IsComplete() {
		snap := setupMgr.Snapshot()
		nickname = snap.Nickname
		sessionSecret = snap.SessionSecret
		if gatewayToken == "" {
			gatewayToken = sessionSecret
		}
	}

	setupStore := setup.NewSessionStore()
	setupHandler := handler.NewSetupHandler(setupMgr, setupStore)
	loginHandler := handler.NewLoginHandler(setupMgr, gatewayURL, gatewayToken)

	downloadsHandler := handler.NewDownloadsHandler(gatewayURL, gatewayToken, nickname)
	uploadsHandler := handler.NewUploadsHandler(gatewayURL, gatewayToken, nickname)
	linksHandler := handler.NewLinksHandler(gatewayURL, gatewayToken)
	searchHandler := handler.NewSearchHandler(gatewayURL, gatewayToken, nickname)
	settingsHandler := handler.NewSettingsHandler(setupMgr, gatewayURL, gatewayToken, nickname)
	serversHandler := handler.NewServersHandler(gatewayURL, gatewayToken, nickname)
	sharesHandler := handler.NewSharesHandler(gatewayURL, gatewayToken, nickname)
	aboutHandler := handler.NewAboutHandler(nickname)
	helpHandler := handler.NewHelpHandler(nickname)

	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	mux.Handle("/setup/", setupHandler)

	mux.Handle("/login", loginHandler)

	// Downloads-Routen
	mux.Handle("/downloads", downloadsHandler)
	mux.Handle("/downloads/", downloadsHandler)
	mux.Handle("/partials/downloads-list", downloadsHandler)
	mux.Handle("/partials/downloads-cards", downloadsHandler)

	// Uploads-Routen
	mux.Handle("/uploads", uploadsHandler)
	mux.Handle("/partials/uploads-list", uploadsHandler)
	mux.Handle("/partials/uploads-cards", uploadsHandler)

	// Links-Route (AJ-Links hinzufügen)
	mux.Handle("/links", linksHandler)

	// Suche-Routen
	mux.Handle("/search", searchHandler)
	mux.Handle("/search/", searchHandler)
	mux.Handle("/partials/search-results", http.HandlerFunc(searchHandler.ServeResults))

	// Einstellungen-Routen
	mux.Handle("/settings", settingsHandler)
	mux.Handle("/settings/plugins/toggle", http.HandlerFunc(settingsHandler.HandleTogglePlugin))
	mux.Handle("/settings/core", http.HandlerFunc(settingsHandler.HandleCoreSettingsUpdate))

	// Server-Routen
	mux.Handle("/server", serversHandler)
	mux.Handle("/shares", sharesHandler)
	mux.Handle("/shares/", sharesHandler)
	mux.Handle("/servers/", serversHandler)
	mux.Handle("/partials/servers-cards", serversHandler)
	mux.Handle("/about", aboutHandler)
	mux.Handle("/help", helpHandler)

	// Geheim-Trigger für den nativen Core
	mux.HandleFunc("/nativecore=aktiv", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "ajnx_native",
			Value:    "true",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   86400 * 365, // 1 Jahr
		})
		http.Redirect(w, r, "/", http.StatusFound)
	})

	mux.HandleFunc("/nativecore=aus", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "ajnx_native",
			Value:    "false",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		})
		http.Redirect(w, r, "/", http.StatusFound)
	})


	mainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wir holen den Nickname dynamisch, falls er sich durch das Setup geändert hat
		currentNick := nickname
		if setupMgr.IsComplete() {
			currentNick = setupMgr.Snapshot().Nickname
		}

		// Den StatusHandler müssen wir eventuell mit dem aktuellen Nicknamen versorgen
		// Da der StatusHandler den Nickname im Constructor bekommt, 
		// nutzen wir hier eine kleine Hilfskonstruktion oder übergeben den Manager.
		// Für den Moment nehmen wir den initialen oder den aus dem Lock.
		
		switch r.URL.Path {
		case "/":
			handler.NewStatusHandler(gatewayURL, gatewayToken, currentNick).ServeHTTP(w, r)
		case "/partials/status-stats":
			handler.NewStatusHandler(gatewayURL, gatewayToken, currentNick).ServeStatusStats(w, r)
		case "/core/shutdown":
			handler.NewStatusHandler(gatewayURL, gatewayToken, currentNick).ServeShutdown(w, r)
		default:
			http.NotFound(w, r)
		}
	})

	guardedMain := handler.LoginGuard(setupMgr)(mainHandler)
	finalHandler := handler.SetupGuard(setupMgr)(buildMux(mux, guardedMain))

	log.Printf("aJnX UI listening on :%s", port)
	if err := http.ListenAndServe(":"+port, finalHandler); err != nil {
		log.Fatal(err)
	}
}

func buildMux(mux *http.ServeMux, guardedMain http.Handler) *http.ServeMux {
	mux.Handle("/", guardedMain)
	return mux
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
