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

func signToken(value, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))
	sig := hex.EncodeToString(mac.Sum(nil))
	return value + "." + sig
}

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

func LoginGuard(m *setup.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			secret := m.Snapshot().SessionSecret
			if secret == "" {
				htmxRedirect(w, r, "/login")
				return
			}

			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				htmxRedirect(w, r, "/login")
				return
			}
			if _, ok := verifyToken(cookie.Value, secret); !ok {
				htmxRedirect(w, r, "/login")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
