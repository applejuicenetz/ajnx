package handler

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/applejuicenetz/ajnx/pkg/setup"
	"github.com/applejuicenetz/ajnx/pkg/ui/view"
)

type xmlSettingsResponse struct {
	XMLName xml.Name `xml:"settings"`
	Nick    string   `xml:"nick"`
}

// validPort prüft ob p ein gültiger TCP-Port ist (1–65535).
func validPort(p string) bool {
	n, err := strconv.Atoi(p)
	return err == nil && n >= 1 && n <= 65535
}

// normalizeHost stellt sicher dass der Host mit http:// beginnt.
func normalizeHost(host string) string {
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return host
	}
	return "http://" + host
}

type SetupHandler struct {
	manager    *setup.Manager
	store      *setup.SessionStore
	httpClient *http.Client
}

func NewSetupHandler(m *setup.Manager, s *setup.SessionStore) *SetupHandler {
	return &SetupHandler{
		manager:    m,
		store:      s,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (h *SetupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	s := h.store.GetState()

	switch path {
	case "/setup/welcome":
		h.handleWelcome(w, r, &s)
	case "/setup/core":
		h.handleCore(w, r, &s)
	case "/setup/test-connection":
		h.handleTestConnection(w, r)
	default:
		http.Redirect(w, r, "/setup/welcome", http.StatusSeeOther)
	}
}

func (h *SetupHandler) handleWelcome(w http.ResponseWriter, r *http.Request, state *setup.WizardState) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		h.store.UpdateState(func(s *setup.WizardState) {
			s.Language = r.FormValue("language")
			s.CurrentStep = 2
		})
		http.Redirect(w, r, "/setup/core", http.StatusSeeOther)
		return
	}
	view.SetupStep1(state).Render(r.Context(), w)
}

func (h *SetupHandler) handleCore(w http.ResponseWriter, r *http.Request, state *setup.WizardState) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		host := normalizeHost(r.FormValue("core_host"))
		port := r.FormValue("core_port")
		password := r.FormValue("core_password")

		// Fallback auf gespeichertes Passwort aus dem SessionStore/State,
		// falls das Feld im Formular leer ist (z.B. nach erfolgreichem Test).
		if password == "" && state.CorePassword != "" {
			password = state.CorePassword
		}

		if !validPort(port) {
			view.SetupStep2(state, "Ungültiger Port (1–65535)").Render(r.Context(), w)
			return
		}

		nick, err := h.testCoreConnection(host, port, password)
		if err != nil {
			h.store.UpdateState(func(s *setup.WizardState) {
				s.CoreHost = host
				s.CorePort = port
			})
			state.CoreHost = host
			state.CorePort = port
			view.SetupStep2(state, err.Error()).Render(r.Context(), w)
			return
		}

		portInt, _ := strconv.Atoi(port)
		if err := h.manager.Complete(setup.Lock{
			CoreHost:     host,
			CorePort:     portInt,
			CorePassword: setup.MD5Hex(password),
			Nickname:     nick,
		}); err != nil {
			view.SetupStep2(state, fmt.Sprintf("Setup konnte nicht gespeichert werden: %v", err)).Render(r.Context(), w)
			return
		}

		h.store.Clear()
		http.Redirect(w, r, "/login?welcome=1", http.StatusSeeOther)
		return
	}
	view.SetupStep2(state, "").Render(r.Context(), w)
}

func (h *SetupHandler) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	host := normalizeHost(r.FormValue("core_host"))
	port := r.FormValue("core_port")
	password := r.FormValue("core_password")

	if !validPort(port) {
		view.SetupTestResult(false, "Ungültiger Port", "Port muss zwischen 1 und 65535 liegen").Render(r.Context(), w)
		return
	}

	h.store.UpdateState(func(s *setup.WizardState) {
		s.CoreHost = host
		s.CorePort = port
		s.CorePassword = password
	})

	nick, err := h.testCoreConnection(host, port, password)
	if err != nil {
		view.SetupTestResult(false, "Verbindung fehlgeschlagen", err.Error()).Render(r.Context(), w)
		return
	}
	view.SetupTestResult(true, "Verbindung erfolgreich", fmt.Sprintf("%s:%s – Nick: %s", host, port, nick)).Render(r.Context(), w)
}

// testCoreConnection ruft settings.xml ab und gibt den Core-Nick zurück.
func (h *SetupHandler) testCoreConnection(host, port, password string) (string, error) {
	url := fmt.Sprintf("%s:%s/xml/settings.xml?password=%s", host, port, setup.MD5Hex(password))

	resp, err := h.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("Core nicht erreichbar (%s:%s): %v", host, port, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Antwort konnte nicht gelesen werden: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		preview := string(body)
		if len(preview) > 120 {
			preview = preview[:120] + "…"
		}
		return "", fmt.Errorf("HTTP %d – falsches Passwort oder falsche Adresse? (%s)", resp.StatusCode, preview)
	}

	trimmed := strings.TrimSpace(string(body))
	if strings.HasPrefix(trimmed, "<html") || strings.HasPrefix(trimmed, "<!") {
		preview := trimmed
		if len(preview) > 80 {
			preview = preview[:80] + "…"
		}
		return "", fmt.Errorf("Core antwortet mit HTML statt XML – URL korrekt? (%s)", preview)
	}

	var settings xmlSettingsResponse
	if err := xml.Unmarshal(body, &settings); err != nil {
		return "", fmt.Errorf("XML konnte nicht geparst werden: %w", err)
	}

	return settings.Nick, nil
}
