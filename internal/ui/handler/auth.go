package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/applejuicenetz/ajnx/internal/setup"
)

const sessionCookieName = "ajnx_session"

// signToken erstellt ein HMAC-SHA256-signiertes Token aus dem gegebenen Wert.
func signToken(value, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))
	sig := hex.EncodeToString(mac.Sum(nil))
	return value + "." + sig
}

// verifyToken prüft ein signiertes Token und gibt den Originalwert zurück.
// Gibt ("", false) zurück wenn die Signatur ungültig ist.
func verifyToken(signed, secret string) (string, bool) {
	idx := strings.LastIndex(signed, ".")
	if idx < 0 {
		return "", false
	}
	value := signed[:idx]
	expected := signToken(value, secret)
	if !hmac.Equal([]byte(signed), []byte(expected)) {
		return "", false
	}
	return value, true
}

// setSessionCookie setzt ein HTTP-Only-Session-Cookie.
func setSessionCookie(w http.ResponseWriter, value, secret string) {
	signed := signToken(value, secret)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    signed,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// LoginGuard ist Middleware, die auf geschützten UI-Routen eine gültige Session erfordert.
// Bei fehlendem oder ungültigem Cookie wird auf /login umgeleitet.
func LoginGuard(m *setup.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			secret := m.Snapshot().SessionSecret
			if secret == "" {
				// Kein Secret vorhanden -> kein Login möglich
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			if _, ok := verifyToken(cookie.Value, secret); !ok {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
