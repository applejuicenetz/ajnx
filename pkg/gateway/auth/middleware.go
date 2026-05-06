package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/applejuicenetz/ajnx/pkg/setup"
)

// contextKey ist ein privater Typ für Context-Keys in diesem Package.
type contextKey struct{ name string }

// sessionContextKey ist der Schlüssel, unter dem die Session im Request-Context gespeichert wird.
var sessionContextKey = &contextKey{"session"}

// SessionFromContext gibt die Session aus dem Request-Context zurück.
// Gibt nil zurück wenn keine User-Session vorhanden (z.B. bei internem Service-Token).
func SessionFromContext(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(sessionContextKey).(*Session)
	return s, ok && s != nil
}

// AuthMiddleware prüft den Authorization-Header auf einen gültigen Bearer-Token.
// Akzeptiert entweder eine aktive User-Session oder den aktuellen internalToken/SessionSecret
// aus dem setup.Manager (Service-to-Service Kommunikation).
func AuthMiddleware(sm *SessionManager, m *setup.Manager, staticInternalToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header missing", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			token := parts[1]

			if staticInternalToken != "" && token == staticInternalToken {
				next.ServeHTTP(w, r)
				return
			}

			if m != nil && m.IsComplete() {
				if token == m.Snapshot().SessionSecret {
					next.ServeHTTP(w, r)
					return
				}
			}

			session, ok := sm.ValidateToken(token)
			if !ok {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), sessionContextKey, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
