package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/applejuicenetz/ajnx/pkg/core"
	"github.com/applejuicenetz/ajnx/pkg/gateway"
	"github.com/applejuicenetz/ajnx/pkg/gateway/api"
	"github.com/applejuicenetz/ajnx/pkg/gateway/auth"
	"github.com/applejuicenetz/ajnx/pkg/gateway/poll"
	"github.com/applejuicenetz/ajnx/pkg/plugin"
	"github.com/applejuicenetz/ajnx/pkg/setup"
)

func main() {
	port := getEnv("GATEWAY_PORT", "9000")
	coreHost := getEnv("CORE_HOST", "http://localhost")
	corePort := getEnv("CORE_XML_PORT", "9851")
	corePassword := getEnv("CORE_PASSWORD", "applejuice")

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
	}

	coreURL := fmt.Sprintf("%s:%s", coreHost, corePort)
	var client *core.XMLCoreClient
	if setupMgr != nil && setupMgr.IsComplete() {
		client = core.NewXMLCoreClientPrehashed(coreURL, corePassword)
	} else {
		client = core.NewXMLCoreClient(coreURL, corePassword)
	}

	var nativeClient core.CoreClient = nil

	bus := plugin.NewBus()
	pm := plugin.NewManager(bus)
	if err := pm.StartAll(); err != nil {
		log.Fatalf("FEHLER: Plugins konnten nicht gestartet werden: %v", err)
	}
	defer pm.StopAll()

	client.SetEventBus(bus)

	state := gateway.NewState()
	sm := auth.NewSessionManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	poller := poll.NewPoller(client, state, setupMgr)
	poller.Run(ctx)

	internalToken := getEnv("INTERNAL_API_KEY", "")
	if internalToken == "" {
		internalToken = sessionSecret
	}
	if internalToken == "" {
		log.Fatal("FEHLER: Weder INTERNAL_API_KEY noch session_secret gefunden.")
	}

	server := api.NewServer(state, sm, client, nativeClient, setupMgr, internalToken, sessionSecret)
	
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: server.Router(),
	}

	go func() {
		log.Printf("aJnX Gateway listening on :%s", port)
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
