package config

import (
	"flag"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ListenAddr string
	DBPath     string
	RPOrigin   string // WebAuthn Relying Party origin, e.g. "http://localhost:9800"
	RPID       string // WebAuthn Relying Party ID, e.g. "localhost"
	RPName     string
	Secret     string // server secret for token encryption

	// Storage (MinIO / S3, or local filesystem)
	StorageEndpoint  string
	StorageAccessKey string
	StorageSecretKey string
	StorageBucket    string
	StorageSSL       bool
	StoragePublicURL string
	StoragePath      string // local filesystem path (used when S3 is not configured)

	// OAuth providers
	GitHubClientID      string
	GitHubClientSecret  string
	LinuxDoClientID     string
	LinuxDoClientSecret string

	AdminSyncSharedSecret  string
	AdminSyncAllowlist     string
	SupabaseURL            string
	SupabaseServiceRoleKey string
	SupabaseSchema         string
	OutboxBatchSize        int
	OutboxPollIntervalMS   int
	OutboxMaxRetries       int
	AIFullPromptMaxBytes   int
}

func Parse() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.ListenAddr, "listen", envOr("LISTEN", ":9800"), "listen address")
	flag.StringVar(&cfg.DBPath, "db", envOr("DATABASE_URL", DefaultDBPath()), "database path or PostgreSQL URL")
	flag.StringVar(&cfg.RPOrigin, "origin", envOr("RP_ORIGIN", "http://localhost:9800"), "WebAuthn RP origin")
	flag.StringVar(&cfg.RPID, "rpid", envOr("RP_ID", "localhost"), "WebAuthn RP ID")
	flag.StringVar(&cfg.RPName, "rpname", envOr("RP_NAME", "OpeniLink Hub"), "WebAuthn RP display name")
	flag.StringVar(&cfg.Secret, "secret", envOr("SECRET", "change-me-in-production"), "server secret")
	// Storage
	cfg.StorageEndpoint = envOr("STORAGE_ENDPOINT", "")
	cfg.StorageAccessKey = envOr("STORAGE_ACCESS_KEY", "")
	cfg.StorageSecretKey = envOr("STORAGE_SECRET_KEY", "")
	cfg.StorageBucket = envOr("STORAGE_BUCKET", "openilink")
	cfg.StorageSSL = envOr("STORAGE_SSL", "") == "true"
	cfg.StoragePublicURL = envOr("STORAGE_PUBLIC_URL", "")
	cfg.StoragePath = envOr("STORAGE_PATH", "")
	// OAuth
	cfg.GitHubClientID = envOr("GITHUB_CLIENT_ID", "")
	cfg.GitHubClientSecret = envOr("GITHUB_CLIENT_SECRET", "")
	cfg.LinuxDoClientID = envOr("LINUXDO_CLIENT_ID", "")
	cfg.LinuxDoClientSecret = envOr("LINUXDO_CLIENT_SECRET", "")
	cfg.AdminSyncSharedSecret = envOr("ADMIN_SYNC_SHARED_SECRET", "")
	cfg.AdminSyncAllowlist = envOr("ADMIN_SYNC_ALLOWLIST", "")
	cfg.SupabaseURL = envOr("SUPABASE_URL", "")
	cfg.SupabaseServiceRoleKey = envOr("SUPABASE_SERVICE_ROLE_KEY", "")
	cfg.SupabaseSchema = envOr("SUPABASE_SCHEMA", "public")
	cfg.OutboxBatchSize = envOrInt("OUTBOX_BATCH_SIZE", 100)
	cfg.OutboxPollIntervalMS = envOrInt("OUTBOX_POLL_INTERVAL_MS", 500)
	cfg.OutboxMaxRetries = envOrInt("OUTBOX_MAX_RETRIES", 10)
	cfg.AIFullPromptMaxBytes = envOrInt("AI_FULL_PROMPT_MAX_BYTES", 8192)
	flag.Parse()
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
