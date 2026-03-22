package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/openilink/openilink-hub/internal/api"
	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/bot"
	"github.com/openilink/openilink-hub/internal/config"
	"github.com/openilink/openilink-hub/internal/database"
	"github.com/openilink/openilink-hub/internal/relay"
)

func main() {
	cfg := config.Parse()

	// Database
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		slog.Error("database open failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// WebAuthn
	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: cfg.RPName,
		RPID:          cfg.RPID,
		RPOrigins:     []string{cfg.RPOrigin},
	})
	if err != nil {
		slog.Error("webauthn init failed", "err", err)
		os.Exit(1)
	}

	// Server components
	srv := &api.Server{
		DB:           db,
		WebAuthn:     wa,
		SessionStore: auth.NewSessionStore(),
	}

	hub := relay.NewHub(srv.SetupUpstreamHandler())
	mgr := bot.NewManager(db, hub)
	srv.BotManager = mgr
	srv.Hub = hub

	// Start all saved bots
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	mgr.StartAll(ctx)

	// Periodic cleanup
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				auth.CleanExpiredSessions(db)
			}
		}
	}()

	// HTTP server
	httpSrv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: srv.Handler(),
	}

	go func() {
		<-ctx.Done()
		slog.Info("shutting down...")
		mgr.StopAll()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		httpSrv.Shutdown(shutCtx)
	}()

	fmt.Printf("OpenILink Hub running on http://localhost%s\n", cfg.ListenAddr)
	if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
