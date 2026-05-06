package handler

import (
	"net/http"
	"strings"
	"github.com/applejuicenetz/ajnx/internal/setup"
)

func SetupGuard(m *setup.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			isSetup := strings.HasPrefix(path, "/setup/")
			isAllowed := isSetup ||
				strings.HasPrefix(path, "/static/") ||
				path == "/healthz"

			if m.IsComplete() {
				if isSetup {
					http.Redirect(w, r, "/", http.StatusSeeOther)
					return
				}
			} else {
				if !isAllowed {
					htmxRedirect(w, r, "/setup/welcome")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
