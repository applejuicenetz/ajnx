package handler

import (
	"net/http"
	"strings"
	"github.com/applejuicenetz/ajnx/internal/setup"
)

// SetupGuard stellt sicher, dass der Wizard nur bei Bedarf erreichbar ist.
func SetupGuard(m *setup.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			isSetup := strings.HasPrefix(path, "/setup/")
			isAllowed := isSetup ||
				strings.HasPrefix(path, "/static/") ||
				path == "/healthz"

			if m.IsComplete() {
				// Setup ist fertig: /setup/* Anfragen werden umgeleitet
				if isSetup {
					http.Redirect(w, r, "/", http.StatusSeeOther)
					return
				}
			} else {
				// Setup fehlt: Alle nicht-erlaubten Pfade werden zum Wizard geschickt
				if !isAllowed {
					http.Redirect(w, r, "/setup/welcome", http.StatusSeeOther)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
