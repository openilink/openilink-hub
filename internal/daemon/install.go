package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"text/template"
)

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
ProtectHome=read-only
ReadWritePaths={{.DataDir}}

[Install]
WantedBy=multi-user.target
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
    <true/>
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
}

// Install installs the service for the current platform.
func Install(listen, dataDir string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
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
		dir := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user")
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

	// Create service user if needed
	if _, err := exec.LookPath("useradd"); err == nil {
		exec.Command("useradd", "--system", "--no-create-home", "--shell", "/usr/sbin/nologin", cfg.User).Run()
	}
	// Ensure data dir is owned by service user
	exec.Command("chown", "-R", cfg.User+":"+cfg.User, cfg.DataDir).Run()

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

func installLaunchd(cfg installConfig) error {
	var dir, path string
	if os.Getuid() == 0 {
		dir = "/Library/LaunchDaemons"
	} else {
		home, _ := os.UserHomeDir()
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
		path := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", "openilink-hub.service")
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
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, "Library", "LaunchAgents", "com.openilink.hub.plist")
	}
	exec.Command("launchctl", "unload", path).Run()
	os.Remove(path)
	fmt.Println("Service removed.")
	return nil
}

func writeTemplate(path, tmpl string, data any) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return err
	}
	return t.Execute(f, data)
}
