package auth

import (
	"context"
	"net/http"
	"strings"
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
// Akzeptiert entweder eine aktive User-Session oder den statischen internalToken
// (Service-to-Service Kommunikation, z.B. UI-Server → Gateway).
func AuthMiddleware(sm *SessionManager, internalToken string) func(http.Handler) http.Handler {
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

			// Interner Service-Token (UI-Server → Gateway)
			if internalToken != "" && token == internalToken {
				next.ServeHTTP(w, r)
				return
			}

			// User-Session prüfen
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
