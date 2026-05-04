package api

import (
	"encoding/json"
	"net/http"
)

type LoginRequest struct {
	Host     string `json:"host"`
	Password string `json:"password"` // MD5
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ok, err := s.client.CheckPassword(r.Context(), req.Password)
	if err != nil {
		http.Error(w, "Core nicht erreichbar", http.StatusServiceUnavailable)
		return
	}
	if !ok {
		http.Error(w, "Ungültiges Passwort", http.StatusUnauthorized)
		return
	}

	token, err := s.sm.CreateSession(req.Host, req.Password)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{Token: token})
}
