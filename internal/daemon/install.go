package daemon

import (
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
)

// listenRe validates listen address format: :port or host:port
var listenRe = regexp.MustCompile(`^[a-zA-Z0-9.\-]*:[0-9]+$`)

const systemdUnit = `[Unit]
Description=OpeniLink Hub
After=network.target

[Service]
Type=simple
ExecStart={{.ExecPath}} -listen {{.Listen}}
WorkingDirectory={{.DataDir}}
Restart=on-failure
RestartSec=5
{{- if ne .User ""}}
User={{.User}}
Group={{.User}}
{{- end}}

# Hardening
NoNewPrivileges=true
ProtectSystem=strict
{{- if ne .User ""}}
ProtectHome=read-only
{{- end}}
ReadWritePaths={{.DataDir}}

[Install]
WantedBy={{.WantedBy}}
`

const launchdPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.openilink.hub</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.ExecPath}}</string>
        <string>-listen</string>
        <string>{{.Listen}}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>WorkingDirectory</key>
    <string>{{.DataDir}}</string>
    <key>StandardOutPath</key>
    <string>{{.DataDir}}/hub.log</string>
    <key>StandardErrorPath</key>
    <string>{{.DataDir}}/hub.log</string>
</dict>
</plist>
`

type installConfig struct {
	ExecPath string
	DataDir  string
	Listen   string
	User     string
	WantedBy string
}

// Install installs the service for the current platform.
func Install(listen, dataDir string) error {
	if !listenRe.MatchString(listen) {
		return fmt.Errorf("invalid listen address: %q (expected host:port or :port)", listen)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return fmt.Errorf("absolute executable path: %w", err)
	}

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	cfg := installConfig{
		ExecPath: execPath,
		DataDir:  dataDir,
		Listen:   listen,
	}

	switch runtime.GOOS {
	case "linux":
		return installSystemd(cfg)
	case "darwin":
		return installLaunchd(cfg)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func installSystemd(cfg installConfig) error {
	if os.Getuid() != 0 {
		// User service
		cfg.User = ""
		cfg.WantedBy = "default.target"
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		dir := filepath.Join(home, ".config", "systemd", "user")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		path := filepath.Join(dir, "openilink-hub.service")
		if err := writeTemplate(path, systemdUnit, cfg); err != nil {
			return err
		}
		fmt.Printf("Service installed: %s\n", path)
		fmt.Println("\nTo start:")
		fmt.Println("  systemctl --user daemon-reload")
		fmt.Println("  systemctl --user enable --now openilink-hub")
		fmt.Println("\nTo view logs:")
		fmt.Println("  journalctl --user -u openilink-hub -f")
		return nil
	}

	// System service (root)
	path := "/etc/systemd/system/openilink-hub.service"
	cfg.User = "openilink"
	cfg.WantedBy = "multi-user.target"

	// Create service user if needed (try useradd first, then adduser for Alpine)
	if err := createSystemUser(cfg.User); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not create user %q: %v\n", cfg.User, err)
	}
	// Ensure data dir is owned by service user
	if err := exec.Command("chown", "-R", cfg.User+":"+cfg.User, cfg.DataDir).Run(); err != nil {
		return fmt.Errorf("chown data directory to %s: %w", cfg.User, err)
	}

	if err := writeTemplate(path, systemdUnit, cfg); err != nil {
		return err
	}
	fmt.Printf("Service installed: %s\n", path)
	fmt.Println("\nTo start:")
	fmt.Println("  systemctl daemon-reload")
	fmt.Println("  systemctl enable --now openilink-hub")
	fmt.Println("\nTo view logs:")
	fmt.Println("  journalctl -u openilink-hub -f")
	return nil
}

func createSystemUser(name string) error {
	// Try useradd (glibc distros)
	if path, err := exec.LookPath("useradd"); err == nil {
		return exec.Command(path, "--system", "--no-create-home", "--shell", "/usr/sbin/nologin", name).Run()
	}
	// Try adduser (Alpine/busybox)
	if path, err := exec.LookPath("adduser"); err == nil {
		return exec.Command(path, "-S", "-D", "-H", "-s", "/sbin/nologin", name).Run()
	}
	return fmt.Errorf("neither useradd nor adduser found")
}

func installLaunchd(cfg installConfig) error {
	var dir, path string
	if os.Getuid() == 0 {
		dir = "/Library/LaunchDaemons"
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		dir = filepath.Join(home, "Library", "LaunchAgents")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path = filepath.Join(dir, "com.openilink.hub.plist")

	if err := writeTemplate(path, launchdPlist, cfg); err != nil {
		return err
	}
	fmt.Printf("Service installed: %s\n", path)
	fmt.Println("\nTo start:")
	fmt.Printf("  launchctl load %s\n", path)
	fmt.Println("\nTo stop:")
	fmt.Printf("  launchctl unload %s\n", path)
	fmt.Println("\nLogs:")
	fmt.Printf("  tail -f %s/hub.log\n", cfg.DataDir)
	return nil
}

// Uninstall removes the service for the current platform.
func Uninstall() error {
	switch runtime.GOOS {
	case "linux":
		return uninstallSystemd()
	case "darwin":
		return uninstallLaunchd()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func uninstallSystemd() error {
	if os.Getuid() != 0 {
		exec.Command("systemctl", "--user", "disable", "--now", "openilink-hub").Run()
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		path := filepath.Join(home, ".config", "systemd", "user", "openilink-hub.service")
		os.Remove(path)
		exec.Command("systemctl", "--user", "daemon-reload").Run()
		fmt.Println("Service removed.")
		return nil
	}
	exec.Command("systemctl", "disable", "--now", "openilink-hub").Run()
	os.Remove("/etc/systemd/system/openilink-hub.service")
	exec.Command("systemctl", "daemon-reload").Run()
	fmt.Println("Service removed.")
	return nil
}

func uninstallLaunchd() error {
	var path string
	if os.Getuid() == 0 {
		path = "/Library/LaunchDaemons/com.openilink.hub.plist"
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		path = filepath.Join(home, "Library", "LaunchAgents", "com.openilink.hub.plist")
	}
	exec.Command("launchctl", "unload", path).Run()
	os.Remove(path)
	fmt.Println("Service removed.")
	return nil
}

func writeTemplate(path, tmpl string, data any) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	// html/template auto-escapes XML metacharacters for plist safety
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return err
	}
	return t.Execute(f, data)
}
