package config

import (
	"flag"
	"os"
)

type Config struct {
	ListenAddr string
	DBPath     string
	RPOrigin   string // WebAuthn Relying Party origin, e.g. "http://localhost:9800"
	RPID       string // WebAuthn Relying Party ID, e.g. "localhost"
	RPName     string
	Secret     string // server secret for token encryption

	// Storage (MinIO / S3)
	StorageEndpoint  string
	StorageAccessKey string
	StorageSecretKey string
	StorageBucket    string
	StorageSSL       bool
	StoragePublicURL string

	// OAuth providers
	GitHubClientID     string
	GitHubClientSecret string
	LinuxDoClientID     string
	LinuxDoClientSecret string
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
	// OAuth
	cfg.GitHubClientID = envOr("GITHUB_CLIENT_ID", "")
	cfg.GitHubClientSecret = envOr("GITHUB_CLIENT_SECRET", "")
	cfg.LinuxDoClientID = envOr("LINUXDO_CLIENT_ID", "")
	cfg.LinuxDoClientSecret = envOr("LINUXDO_CLIENT_SECRET", "")
	flag.Parse()
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
