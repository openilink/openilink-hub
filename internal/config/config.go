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
}

func Parse() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.ListenAddr, "listen", envOr("LISTEN", ":9800"), "listen address")
	flag.StringVar(&cfg.DBPath, "db", envOr("DB_PATH", "openilink-hub.db"), "sqlite database path")
	flag.StringVar(&cfg.RPOrigin, "origin", envOr("RP_ORIGIN", "http://localhost:9800"), "WebAuthn RP origin")
	flag.StringVar(&cfg.RPID, "rpid", envOr("RP_ID", "localhost"), "WebAuthn RP ID")
	flag.StringVar(&cfg.RPName, "rpname", envOr("RP_NAME", "OpenILink Hub"), "WebAuthn RP display name")
	flag.StringVar(&cfg.Secret, "secret", envOr("SECRET", "change-me-in-production"), "server secret")
	flag.Parse()
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
