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
