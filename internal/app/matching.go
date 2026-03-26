package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/openilink/openilink-hub/internal/store"
)

// ParseMention extracts handle, command, and text from @handle messages.
// Returns handle, command (with / prefix or empty), remaining text.
// Examples:
//   "@echo-work hello"          → "echo-work", "", "hello"
//   "@echo-work /echo hello"    → "echo-work", "/echo", "hello"
//   "@echo-work"                → "echo-work", "", ""
//   "hello"                     → "", "", ""
func ParseMention(content string) (handle, command, text string) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "@") {
		return "", "", ""
	}
	parts := strings.SplitN(content[1:], " ", 2)
	handle = strings.ToLower(parts[0])
	if handle == "" {
		return "", "", ""
	}
	remaining := ""
	if len(parts) > 1 {
		remaining = strings.TrimSpace(parts[1])
	}
	if strings.HasPrefix(remaining, "/") {
		cmdParts := strings.SplitN(remaining[1:], " ", 2)
		command = "/" + strings.ToLower(cmdParts[0])
		if len(cmdParts) > 1 {
			text = strings.TrimSpace(cmdParts[1])
		}
		return handle, command, text
	}
	return handle, "", remaining
}

// MatchHandle finds an enabled installation with the given handle on a bot.
func (d *Dispatcher) MatchHandle(botID, handle string) (*store.AppInstallation, error) {
	inst, err := d.store().GetInstallationByHandle(botID, handle)
	if err != nil {
		return nil, err
	}
	if !inst.Enabled {
		return nil, nil
	}
	return inst, nil
}

// MatchCommand parses a slash command from the message content and finds
// installations on the given bot whose app has registered that command.
// Content format: "/commandname args..." or "@bothandle /commandname args..."
// Returns matching installations, the parsed command name (without "/"), and
// the remaining args string.
func (d *Dispatcher) MatchCommand(botID string, content string) ([]store.AppInstallation, string, string, error) {
	command, args := parseCommand(content)
	if command == "" {
		return nil, "", "", nil
	}

	installations, err := d.store().ListInstallationsByBot(botID)
	if err != nil {
		return nil, "", "", fmt.Errorf("list installations: %w", err)
	}

	var matched []store.AppInstallation
	for _, inst := range installations {
		if !inst.Enabled {
			continue
		}

		app, err := d.store().GetApp(inst.AppID)
		if err != nil {
			slog.Error("failed to get app for matching",
				"app_id", inst.AppID, "err", err)
			continue
		}

		if appHasCommand(app, command) {
			matched = append(matched, inst)
		}
	}

	return matched, command, args, nil
}

// MatchEvent finds installations on the given bot whose app subscribes to
// the specified event type. The wildcard event type "message" matches any
// "message.*" event.
func (d *Dispatcher) MatchEvent(botID string, eventType string) ([]store.AppInstallation, error) {
	installations, err := d.store().ListInstallationsByBot(botID)
	if err != nil {
		return nil, fmt.Errorf("list installations: %w", err)
	}

	var matched []store.AppInstallation
	for _, inst := range installations {
		if !inst.Enabled {
			continue
		}

		app, err := d.store().GetApp(inst.AppID)
		if err != nil {
			slog.Error("failed to get app for event matching",
				"app_id", inst.AppID, "err", err)
			continue
		}

		// App must have message:read scope to receive message events
		if strings.HasPrefix(eventType, "message.") || eventType == "message" {
			if !appHasScope(app, "message:read") {
				continue
			}
		}

		if appSubscribesToEvent(app, eventType) {
			matched = append(matched, inst)
		}
	}

	return matched, nil
}

// parseCommand extracts a command name and args from message content.
// Recognizes formats like:
//   - "/command arg1 arg2"
//   - "@mention /command arg1 arg2"
//
// Returns empty command if no slash command is found.
func parseCommand(content string) (string, string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", ""
	}

	// Skip leading @mention if present.
	if strings.HasPrefix(content, "@") {
		idx := strings.IndexByte(content, ' ')
		if idx < 0 {
			// Just a mention, no command.
			return "", ""
		}
		content = strings.TrimSpace(content[idx+1:])
	}

	// Must start with "/" to be a command.
	if !strings.HasPrefix(content, "/") {
		return "", ""
	}

	// Split into command and args.
	content = content[1:] // strip leading "/"
	parts := strings.SplitN(content, " ", 2)
	command := strings.ToLower(parts[0])
	if command == "" {
		return "", ""
	}

	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	return command, args
}

// appHasCommand checks whether an app has a tool with a matching Command trigger.
func appHasCommand(app *store.App, commandName string) bool {
	if app == nil || len(app.Tools) == 0 {
		return false
	}

	var tools []store.AppTool
	if err := json.Unmarshal(app.Tools, &tools); err != nil {
		slog.Error("failed to unmarshal app tools",
			"app_id", app.ID, "err", err)
		return false
	}

	for _, tool := range tools {
		if tool.Command == "" {
			continue
		}
		if strings.ToLower(tool.Command) == strings.ToLower(commandName) {
			return true
		}
	}
	return false
}

// appHasScope checks whether an app declares the given scope.
func appHasScope(app *store.App, scope string) bool {
	if app == nil || len(app.Scopes) == 0 {
		return false
	}
	var scopes []string
	if err := json.Unmarshal(app.Scopes, &scopes); err != nil {
		return false
	}
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// appSubscribesToEvent checks whether an app subscribes to the given event type.
// Supports the wildcard "message" which matches any "message.*" event.
func appSubscribesToEvent(app *store.App, eventType string) bool {
	if app == nil || len(app.Events) == 0 {
		return false
	}

	var events []string
	if err := json.Unmarshal(app.Events, &events); err != nil {
		slog.Error("failed to unmarshal app events",
			"app_id", app.ID, "err", err)
		return false
	}

	for _, subscribed := range events {
		if subscribed == eventType {
			return true
		}
		// Wildcard: "message" matches any "message.*" event.
		if subscribed == "message" && strings.HasPrefix(eventType, "message.") {
			return true
		}
	}
	return false
}

