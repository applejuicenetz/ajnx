package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/applejuicenetz/ajnx/internal/core"
	"github.com/applejuicenetz/ajnx/internal/gateway"
	"github.com/applejuicenetz/ajnx/internal/gateway/api"
	"github.com/applejuicenetz/ajnx/internal/gateway/auth"
	"github.com/applejuicenetz/ajnx/internal/gateway/poll"
	"github.com/applejuicenetz/ajnx/internal/plugin"
	"github.com/applejuicenetz/ajnx/internal/setup"
)

func main() {
	// Konfiguration aus Umgebungsvariablen (Fallback-Werte)
	port := getEnv("GATEWAY_PORT", "9000")
	coreHost := getEnv("CORE_HOST", "http://localhost")
	corePort := getEnv("CORE_XML_PORT", "9851")
	corePassword := getEnv("CORE_PASSWORD", "applejuice")

	// setup.lock einlesen
	lockPath := getEnv("SETUP_LOCK_PATH", "./data/setup.lock")
	sessionSecret := ""

	setupMgr, err := setup.NewManager(lockPath)
	if err != nil {
		log.Printf("WARNUNG: setup.lock nicht lesbar (%v) – Env-Vars werden verwendet", err)
	} else if setupMgr.IsComplete() {
		snap := setupMgr.Snapshot()
		coreHost = snap.CoreHost
		corePort = fmt.Sprintf("%d", snap.CorePort)
		corePassword = snap.CorePassword
		sessionSecret = snap.SessionSecret
		log.Printf("Konfiguration aus setup.lock geladen (Core: %s:%s)", coreHost, corePort)
	}

	// Core Client initialisieren
	coreURL := fmt.Sprintf("%s:%s", coreHost, corePort)
	var client *core.XMLCoreClient
	if setupMgr != nil && setupMgr.IsComplete() {
		client = core.NewXMLCoreClientPrehashed(coreURL, corePassword)
	} else {
		client = core.NewXMLCoreClient(coreURL, corePassword)
	}

	// Plugin System
	bus := plugin.NewBus()
	pm := plugin.NewManager(bus)
	if err := pm.StartAll(); err != nil {
		log.Fatalf("FEHLER: Plugins konnten nicht gestartet werden: %v", err)
	}
	defer pm.StopAll()

	client.SetEventBus(bus)

	// State und Auth
	state := gateway.NewState()
	sm := auth.NewSessionManager()

	// Poller starten (WICHTIG: setupMgr übergeben für dynamisches Reload)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	poller := poll.NewPoller(client, state, setupMgr)
	poller.Run(ctx)

	// Interner Service-Token
	internalToken := getEnv("INTERNAL_API_KEY", "")
	if internalToken == "" {
		internalToken = sessionSecret
	}
	if internalToken == "" {
		log.Fatal("FEHLER: Weder INTERNAL_API_KEY noch session_secret gefunden.")
	}

	// API Server
	server := api.NewServer(state, sm, client, internalToken, sessionSecret)
	
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: server.Router(),
	}

	go func() {
		log.Printf("AJNX Gateway listening on :%s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	httpServer.Shutdown(shutdownCtx)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func md5prefix(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])[:8]
}
