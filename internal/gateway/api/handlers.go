package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/applejuicenetz/ajnx/internal/core"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	info := s.state.GetInformation()
	if info == nil {
		errStr := s.state.GetLastError()
		if errStr == "" {
			errStr = "Status not yet available"
		}
		http.Error(w, errStr, http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (s *Server) handleDownloads(w http.ResponseWriter, r *http.Request) {
	downloads := s.state.GetDownloads()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(downloads)
}

func (s *Server) handleUploads(w http.ResponseWriter, r *http.Request) {
	uploads := s.state.GetUploads()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(uploads)
}

func (s *Server) handleServers(w http.ResponseWriter, r *http.Request) {
	servers := s.state.GetServers()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

func (s *Server) handleDownloadPause(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.client.PauseDownload(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleDownloadResume(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.client.ResumeDownload(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleDownloadCancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.client.CancelDownload(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleProcessLink(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Link string `json:"link"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		fmt.Printf("[Gateway] Fehler beim Decodieren des Bodies: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Regex zum Finden aller appleJuice Links
	// Wir erlauben Leerzeichen im Link (für Dateinamen), stoppen aber bei typischen Delimitern oder Zeilenumbruch.
	re := regexp.MustCompile(`(ajfsp://|web\+ajlink://|web\+ajfsp://)[^<>"\r\n]+`)
	matches := re.FindAllString(payload.Link, -1)

	if len(matches) == 0 {
		// Falls kein Protokoll gefunden wurde, versuchen wir es als einen einzelnen, 
		// evtl. unvollständigen Link zu behandeln (Legacy-Support)
		if payload.Link != "" {
			trimmed := strings.TrimSpace(payload.Link)
			if trimmed != "" {
				matches = []string{trimmed}
			}
		}
	}

	if len(matches) == 0 {
		http.Error(w, "Keine gültigen Links gefunden", http.StatusBadRequest)
		return
	}

	fmt.Printf("[Gateway] Verarbeite %d Links...\n", len(matches))

	var errs []error
	for _, link := range matches {
		fmt.Printf("[Gateway] Sende Link an Core: %s\n", link)
		if err := s.client.ProcessLink(r.Context(), link); err != nil {
			fmt.Printf("[Gateway] Fehler bei Link %s: %v\n", link, err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 && len(errs) == len(matches) {
		msg := fmt.Sprintf("Alle %d Links konnten nicht verarbeitet werden. Letzter Fehler: %v", len(matches), errs[0])
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	fmt.Printf("[Gateway] Verarbeitung abgeschlossen (%d erfolgreich)\n", len(matches)-len(errs))
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleSearchStart(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		fmt.Printf("[Gateway] Fehler beim Decodieren des Search-Bodies: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if payload.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	fmt.Printf("[Gateway] Starte Core-Suche für: %q\n", payload.Query)
	if err := s.client.StartSearch(r.Context(), payload.Query); err != nil {
		fmt.Printf("[Gateway] Core-Suche-Fehler: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("[Gateway] Suche erfolgreich an Core übergeben.")
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleSearchGet(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[Gateway] Abfrage der Suchergebnisse vom Core...")
	searches, err := s.client.Searches(r.Context())
	if err != nil {
		fmt.Printf("[Gateway] Fehler beim Abrufen der Suchen: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("[Gateway] %d Suchen vom Core empfangen. Sende an UI...\n", len(searches))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searches)
}

func (s *Server) handleSearchCancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.client.CancelSearch(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	settings, err := s.client.GetSettings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (s *Server) handleSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	var settings core.XMLSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.client.UpdateSettings(r.Context(), settings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleCleanDownloads(w http.ResponseWriter, r *http.Request) {
	if err := s.client.CleanDownloadList(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleServerConnect(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.client.ConnectServer(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleServerDisconnect(w http.ResponseWriter, r *http.Request) {
	if err := s.client.DisconnectServer(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleServerRemove(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.client.RemoveServer(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleCoreShutdown(w http.ResponseWriter, r *http.Request) {
	if err := s.client.ShutdownCore(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

