package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/openilink/openilink-hub/internal/api"
	appdelivery "github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/bot"
	"github.com/openilink/openilink-hub/internal/builtin"
	"github.com/openilink/openilink-hub/internal/config"
	"github.com/openilink/openilink-hub/internal/daemon"
	"github.com/openilink/openilink-hub/internal/push"
	"github.com/openilink/openilink-hub/internal/registry"
	"github.com/openilink/openilink-hub/internal/relay"
	"github.com/openilink/openilink-hub/internal/sink"
	"github.com/openilink/openilink-hub/internal/storage"
	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/store/postgres"
	"github.com/openilink/openilink-hub/internal/store/sqlite"
	"github.com/openilink/openilink-hub/internal/supamemory"
	syncworker "github.com/openilink/openilink-hub/internal/sync"

	// Register providers
	_ "github.com/openilink/openilink-hub/internal/provider/ilink"

	// Register builtin apps
	_ "github.com/openilink/openilink-hub/internal/builtin/bridge"
	_ "github.com/openilink/openilink-hub/internal/builtin/github"
	_ "github.com/openilink/openilink-hub/internal/builtin/hermes"
	_ "github.com/openilink/openilink-hub/internal/builtin/mcpserver"
	_ "github.com/openilink/openilink-hub/internal/builtin/openclaw"
	_ "github.com/openilink/openilink-hub/internal/builtin/runner"
)

// Set by goreleaser ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func openStore(dsn string) (store.Store, error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return postgres.Open(dsn)
	}
	return sqlite.Open(dsn)
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("oih (OpeniLink Hub) %s (%s, %s)\n", version, commit, date)
			return
		case "install":
			listen := ":9800"
			if len(os.Args) > 2 {
				listen = os.Args[2]
			}
			if err := daemon.Install(listen, config.DataDir()); err != nil {
				fmt.Fprintf(os.Stderr, "install failed: %v\n", err)
				os.Exit(1)
			}
			return
		case "uninstall":
			if err := daemon.Uninstall(); err != nil {
				fmt.Fprintf(os.Stderr, "uninstall failed: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	cfg := config.Parse()

	// Ensure data directory exists for SQLite
	if !strings.HasPrefix(cfg.DBPath, "postgres") {
		if err := config.EnsureDataDir(); err != nil {
			slog.Error("create data directory failed", "err", err)
			os.Exit(1)
		}
	}

	// Database
	s, err := openStore(cfg.DBPath)
	if err != nil {
		slog.Error("database open failed", "err", err)
		os.Exit(1)
	}
	defer s.Close()

	// Seed builtin apps
	if err := builtin.SeedApps(s); err != nil {
		slog.Error("seed builtin apps failed", "err", err)
	}

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

	// Registry client
	regClient := registry.NewClient(0)
	registries, err := s.ListRegistries()
	if err != nil {
		slog.Warn("failed to load registry sources", "err", err)
	} else {
		enabledCount := 0
		for _, reg := range registries {
			if reg.Enabled {
				regClient.AddSource(reg.Name, reg.URL)
				enabledCount++
			}
		}
		if enabledCount > 0 {
			slog.Info("registry sources loaded", "count", enabledCount)
		}
	}

	// Server components
	srv := &api.Server{
		Store:        s,
		WebAuthn:     wa,
		SessionStore: auth.NewSessionStore(),
		Config:       cfg,
		OAuthStates:  api.SetupOAuth(cfg),
		Registry:     regClient,
		Version:      version,
	}

	var runtimeSupa *supamemory.Client
	if cfg.SupabaseURL != "" && cfg.SupabaseServiceRoleKey != "" && cfg.SupabaseMemoryEnabled {
		runtimeSupa, err = supamemory.NewClient(supamemory.Config{
			BaseURL:        cfg.SupabaseURL,
			ServiceRoleKey: cfg.SupabaseServiceRoleKey,
			Schema:         cfg.SupabaseSchema,
			MemoryEnabled:  cfg.SupabaseMemoryEnabled,
			MemoryTopK:     cfg.SupabaseMemoryTopK,
			MemoryTable:    cfg.SupabaseMemoryTable,
			MemoryMatchRPC: cfg.SupabaseMemoryMatchRPC,
			BindingsTable:  cfg.SupabaseBindingsTable,
			RoutesTable:    cfg.SupabaseRoutesTable,
			BotsTable:      cfg.SupabaseBotsTable,
			ProfilesTable:  cfg.SupabaseProfilesTable,
			EmbeddingModel: cfg.SupabaseEmbeddingModel,
		})
		if err != nil {
			slog.Warn("runtime supabase memory disabled", "err", err)
		}
	}
	srv.SupaMemory = runtimeSupa

	// Storage (optional): S3 > local FS > proxy fallback
	var objStore storage.Store
	if cfg.StorageEndpoint != "" {
		publicURL := cfg.StoragePublicURL
		if publicURL == "" {
			publicURL = cfg.RPOrigin + "/api/v1/media"
		}
		var err error
		objStore, err = storage.NewS3(storage.S3Config{
			Endpoint:  cfg.StorageEndpoint,
			AccessKey: cfg.StorageAccessKey,
			SecretKey: cfg.StorageSecretKey,
			Bucket:    cfg.StorageBucket,
			UseSSL:    cfg.StorageSSL,
			PublicURL: publicURL,
		})
		if err != nil {
			slog.Error("storage init failed (s3)", "err", err)
			os.Exit(1)
		}
		slog.Info("storage connected (s3)", "endpoint", cfg.StorageEndpoint, "bucket", cfg.StorageBucket)
	} else if cfg.StoragePath != "" {
		var err error
		objStore, err = storage.NewFS(cfg.StoragePath, cfg.RPOrigin+"/api/v1/media")
		if err != nil {
			slog.Error("storage init failed (fs)", "err", err)
			os.Exit(1)
		}
		slog.Info("storage connected (fs)", "dir", cfg.StoragePath)
	} else {
		slog.Info("storage not configured, media will use CDN proxy")
	}
	if objStore != nil {
		srv.ObjectStore = objStore
	}

	hub := relay.NewHub(srv.SetupUpstreamHandler())
	appDisp := appdelivery.NewDispatcher(s)
	aiSink := &sink.AI{Store: s, AppDisp: appDisp, Storage: objStore, SupaMemory: runtimeSupa}
	mgr := bot.NewManager(s, hub, aiSink, objStore, cfg.RPOrigin)
	aiSink.BotManager = mgr
	srv.BotManager = mgr
	srv.Hub = hub
	srv.AppWSHub = api.NewAppWSHub()
	srv.PushHub = push.NewHub()
	mgr.SetAppWSHub(srv.AppWSHub)
	mgr.SetPushHub(srv.PushHub)

	var outboxWorker *syncworker.OutboxWorker
	if cfg.SupabaseURL != "" && cfg.SupabaseServiceRoleKey != "" {
		supabaseClient, err := syncworker.NewHTTPSupabaseClient(cfg.SupabaseURL, cfg.SupabaseServiceRoleKey, cfg.SupabaseSchema)
		if err != nil {
			slog.Error("supabase client init failed", "err", err)
		} else {
			wcfg := syncworker.ParseOutboxConfig(cfg.OutboxBatchSize, cfg.OutboxPollIntervalMS, cfg.OutboxMaxRetries)
			outboxWorker = syncworker.NewOutboxWorker(s, supabaseClient, wcfg)
			slog.Info("outbox worker configured", "batch_size", wcfg.BatchSize, "poll_interval", wcfg.PollInterval, "max_retries", wcfg.MaxRetries)
		}
	} else {
		slog.Info("outbox worker disabled: missing SUPABASE_URL or SUPABASE_SERVICE_ROLE_KEY")
	}

	// Start all saved bots
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	mgr.StartAll(ctx)
	if outboxWorker != nil {
		go outboxWorker.Run(ctx)
	}

	// Periodic cleanup
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				auth.CleanExpiredSessions(s)
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

	fmt.Printf("OpeniLink Hub %s (%s, %s) running on http://localhost%s\n", version, commit, date, cfg.ListenAddr)
	fmt.Printf("Data: %s\n", config.DataDir())
	if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
