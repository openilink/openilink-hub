package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// DataDir returns the platform-standard data directory for openilink-hub.
//
//	Linux:       ~/.local/share/openilink-hub/
//	macOS:       ~/Library/Application Support/openilink-hub/
//	root/service: /var/lib/openilink-hub/
func DataDir() string {
	if os.Getuid() == 0 {
		return "/var/lib/openilink-hub"
	}
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "openilink-hub")
	default: // linux, freebsd, etc.
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, "openilink-hub")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "share", "openilink-hub")
	}
}

// DefaultDBPath returns the default SQLite database path.
func DefaultDBPath() string {
	return filepath.Join(DataDir(), "openilink.db")
}

// EnsureDataDir creates the data directory if it doesn't exist.
func EnsureDataDir() error {
	return os.MkdirAll(DataDir(), 0755)
}
