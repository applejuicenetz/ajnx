package handler

import "net/http"

func isNativeMode(r *http.Request) bool {
	cookie, err := r.Cookie("ajnx_native")
	if err != nil {
		return false
	}
	return cookie.Value == "true"
}

func addNativeHeader(r *http.Request, req *http.Request) {
	if isNativeMode(r) {
		req.Header.Set("X-AJNX-Native", "true")
	}
}

func htmxRedirect(w http.ResponseWriter, r *http.Request, target string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", target)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
