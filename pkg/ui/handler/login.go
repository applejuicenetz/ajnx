package handler

import (
	"fmt"
	"net/http"

	"github.com/applejuicenetz/ajnx/pkg/setup"
	"github.com/applejuicenetz/ajnx/pkg/ui/view"
)

// LoginHandler verwaltet die AJNX-Login-Seite.
type LoginHandler struct {
	setupMgr   *setup.Manager
	gatewayURL string
	token      string
}

// NewLoginHandler erstellt einen neuen LoginHandler.
// gatewayURL und token werden benötigt um den Core-Status vor dem Login zu prüfen.
func NewLoginHandler(mgr *setup.Manager, gatewayURL, token string) *LoginHandler {
	return &LoginHandler{setupMgr: mgr, gatewayURL: gatewayURL, token: token}
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	welcome := r.URL.Query().Get("welcome") == "1"

	if r.Method == http.MethodPost {
		h.handlePost(w, r)
		return
	}

	view.LoginPage("", welcome).Render(r.Context(), w)
}

func (h *LoginHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		view.LoginPage("Ungültige Anfrage.", false).Render(r.Context(), w)
		return
	}

	password := r.FormValue("password")
	snap := h.setupMgr.Snapshot()

	if setup.MD5Hex(password) != snap.CorePassword {
		view.LoginPage("Falsches Passwort. Bitte erneut versuchen.", false).Render(r.Context(), w)
		return
	}

	// Core-Verbindung prüfen bevor die Session erstellt wird.
	// Verhindert, dass der Nutzer in ein nicht-funktionsfähiges Dashboard gelangt.
	if err := h.checkCoreStatus(r); err != nil {
		view.LoginPage(err.Error(), false).Render(r.Context(), w)
		return
	}

	setSessionCookie(w, "authenticated", snap.SessionSecret)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// checkCoreStatus fragt den Gateway-Status ab und gibt einen Fehler zurück wenn der Core
// nicht erreichbar ist oder keine Daten liefert.
func (h *LoginHandler) checkCoreStatus(r *http.Request) error {
	url := h.gatewayURL + "/api/status"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("Interner Fehler beim Status-Check.")
	}
	req.Header.Set("Authorization", "Bearer "+h.token)

	resp, err := gatewayClient.Do(req)
	if err != nil {
		return fmt.Errorf("Gateway nicht erreichbar – bitte prüfe ob der Gateway-Dienst läuft.")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable {
		return fmt.Errorf("Core nicht verbunden – bitte prüfe Host, Port und Passwort in den Einstellungen.")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Core-Verbindung fehlgeschlagen (Status %d) – bitte Konfiguration prüfen.", resp.StatusCode)
	}
	return nil
}
