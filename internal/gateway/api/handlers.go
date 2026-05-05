package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/applejuicenetz/ajnx/internal/core"
	"github.com/applejuicenetz/ajnx/internal/domain"
	"github.com/go-chi/chi/v5"
)

var linkRegex = regexp.MustCompile(`(ajfsp://|web\+ajlink://|web\+ajfsp://)[^<>"\r\n]+`)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	var info *domain.Information
	if s.isNative(r) {
		info, _ = s.nativeClient.Information(r.Context())
	} else {
		info = s.state.GetInformation()
	}

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
	var downloads []domain.Download
	if s.isNative(r) {
		downloads, _ = s.nativeClient.Downloads(r.Context())
	} else {
		downloads = s.state.GetDownloads()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(downloads)
}

func (s *Server) handleUploads(w http.ResponseWriter, r *http.Request) {
	var uploads []domain.Upload
	if s.isNative(r) {
		uploads, _ = s.nativeClient.Uploads(r.Context())
	} else {
		uploads = s.state.GetUploads()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(uploads)
}

func (s *Server) handleServers(w http.ResponseWriter, r *http.Request) {
	var servers []domain.Server
	if s.isNative(r) {
		servers, _ = s.nativeClient.Servers(r.Context())
	} else {
		servers = s.state.GetServers()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

func (s *Server) handleDownloadPause(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.getClient(r).PauseDownload(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleDownloadResume(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.getClient(r).ResumeDownload(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleDownloadCancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.getClient(r).CancelDownload(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleDownloadPDL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var payload struct {
		Value float64 `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.getClient(r).SetPowerDownload(r.Context(), id, payload.Value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleProcessLink(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Link string `json:"link"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	matches := linkRegex.FindAllString(payload.Link, -1)

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

	var errs []error
	for _, link := range matches {
		if err := s.getClient(r).ProcessLink(r.Context(), link); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 && len(errs) == len(matches) {
		msg := fmt.Sprintf("Alle %d Links konnten nicht verarbeitet werden. Letzter Fehler: %v", len(matches), errs[0])
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleSearchStart(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if payload.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	if err := s.getClient(r).StartSearch(r.Context(), payload.Query); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleSearchGet(w http.ResponseWriter, r *http.Request) {
	searches, err := s.getClient(r).Searches(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searches)
}

func (s *Server) handleSearchCancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.getClient(r).CancelSearch(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	settings, err := s.getClient(r).GetSettings(r.Context())
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

	if err := s.getClient(r).UpdateSettings(r.Context(), settings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleCleanDownloads(w http.ResponseWriter, r *http.Request) {
	if err := s.getClient(r).CleanDownloadList(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleServerConnect(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.getClient(r).ConnectServer(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleServerDisconnect(w http.ResponseWriter, r *http.Request) {
	if err := s.getClient(r).DisconnectServer(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleServerRemove(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.getClient(r).RemoveServer(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleCoreShutdown(w http.ResponseWriter, r *http.Request) {
	if err := s.getClient(r).ShutdownCore(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
func (s *Server) handleShares(w http.ResponseWriter, r *http.Request) {
	shares, err := s.getClient(r).Shares(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shares)
}

func (s *Server) handleShareDirsGet(w http.ResponseWriter, r *http.Request) {
	dirs, err := s.getClient(r).GetShareDirs(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dirs)
}

func (s *Server) handleShareDirsSet(w http.ResponseWriter, r *http.Request) {
	var dirs []domain.ShareDir
	if err := json.NewDecoder(r.Body).Decode(&dirs); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := s.getClient(r).SetShares(r.Context(), dirs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

