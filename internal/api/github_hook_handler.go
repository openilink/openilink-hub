package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/openilink/openilink-hub/internal/provider"
)

// handleGitHubWebhook handles POST /api/hooks/github?token={app_token}
// This is the public endpoint that GitHub calls when events occur.
func (s *Server) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	inst, err := s.Store.GetInstallationByToken(token)
	if err != nil || inst.AppSlug != "github" {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	if !inst.Enabled {
		http.Error(w, "app disabled", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1*1024*1024)) // 1MB limit
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	// Verify signature if secret is configured
	var cfg struct {
		Secret string `json:"secret"`
	}
	if len(inst.Config) > 0 {
		if err := json.Unmarshal(inst.Config, &cfg); err != nil {
			slog.Error("github: invalid installation config", "installation_id", inst.ID, "err", err)
			http.Error(w, "invalid installation config", http.StatusInternalServerError)
			return
		}
	}
	if cfg.Secret != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		if !verifyGitHubSignature(cfg.Secret, body, sig) {
			http.Error(w, "invalid signature", http.StatusForbidden)
			return
		}
	}

	// Parse event
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		http.Error(w, "missing X-GitHub-Event header", http.StatusBadRequest)
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Format message
	text := formatGitHubEvent(eventType, payload)
	if text == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Send via bot
	botInst, ok := s.BotManager.GetInstance(inst.BotID)
	if !ok {
		slog.Warn("github: bot not connected", "bot_id", inst.BotID)
		http.Error(w, "bot not connected", http.StatusServiceUnavailable)
		return
	}

	sendCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	contextToken := s.Store.GetLatestContextToken(inst.BotID)
	_, err = botInst.Send(sendCtx, provider.OutboundMessage{
		Text:         text,
		ContextToken: contextToken,
	})
	if err != nil {
		slog.Error("github: send failed", "bot_id", inst.BotID, "event", eventType, "err", err)
		http.Error(w, "send failed", http.StatusBadGateway)
		return
	}

	slog.Info("github: event delivered", "bot_id", inst.BotID, "event", eventType)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true}`))
}

func verifyGitHubSignature(secret string, body []byte, signature string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	expected := signature[7:]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	actual := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(actual))
}

// formatGitHubEvent converts a GitHub webhook event into a readable message.
func formatGitHubEvent(eventType string, p map[string]any) string {
	repo := jsonStr(p, "repository", "full_name")
	sender := jsonStr(p, "sender", "login")

	switch eventType {
	case "ping":
		return fmt.Sprintf("🔔 [%s] Webhook 连接成功\n%s", repo, jsonStr(p, "zen"))

	case "push":
		ref := getString(p, "ref")
		branch := strings.TrimPrefix(strings.TrimPrefix(ref, "refs/heads/"), "refs/tags/")
		commits := getArray(p, "commits")
		if len(commits) == 0 {
			return ""
		}
		lines := []string{fmt.Sprintf("📦 [%s] %s 推送了 %d 个提交到 %s", repo, sender, len(commits), branch)}
		for i, c := range commits {
			if i >= 5 {
				lines = append(lines, fmt.Sprintf("  ... 还有 %d 个提交", len(commits)-5))
				break
			}
			cm := toMap(c)
			msg := getString(cm, "message")
			if idx := strings.Index(msg, "\n"); idx > 0 {
				msg = msg[:idx]
			}
			sha := getString(cm, "id")
			if len(sha) > 7 {
				sha = sha[:7]
			}
			lines = append(lines, fmt.Sprintf("  %s %s", sha, msg))
		}
		return strings.Join(lines, "\n")

	case "pull_request":
		action := getString(p, "action")
		pr := getMap(p, "pull_request")
		title := getString(pr, "title")
		number := getNumber(pr, "number")
		url := getString(pr, "html_url")
		merged := getBool(pr, "merged")

		emoji := "🔀"
		verb := action
		switch action {
		case "opened":
			verb = "创建了"
		case "closed":
			if merged {
				emoji = "🟣"
				verb = "合并了"
			} else {
				verb = "关闭了"
			}
		case "reopened":
			verb = "重新打开了"
		case "ready_for_review":
			verb = "标记为可 review"
		default:
			return ""
		}
		return fmt.Sprintf("%s [%s] %s %s PR #%d: %s\n%s", emoji, repo, sender, verb, int(number), title, url)

	case "issues":
		action := getString(p, "action")
		issue := getMap(p, "issue")
		title := getString(issue, "title")
		number := getNumber(issue, "number")
		url := getString(issue, "html_url")

		verb := action
		switch action {
		case "opened":
			verb = "创建了"
		case "closed":
			verb = "关闭了"
		case "reopened":
			verb = "重新打开了"
		default:
			return ""
		}
		return fmt.Sprintf("📋 [%s] %s %s Issue #%d: %s\n%s", repo, sender, verb, int(number), title, url)

	case "issue_comment":
		action := getString(p, "action")
		if action != "created" {
			return ""
		}
		issue := getMap(p, "issue")
		comment := getMap(p, "comment")
		title := getString(issue, "title")
		number := getNumber(issue, "number")
		body := getString(comment, "body")
		url := getString(comment, "html_url")
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		kind := "Issue"
		if _, ok := issue["pull_request"]; ok {
			kind = "PR"
		}
		return fmt.Sprintf("💬 [%s] %s 评论了 %s #%d: %s\n%s\n%s", repo, sender, kind, int(number), title, body, url)

	case "release":
		action := getString(p, "action")
		if action != "published" {
			return ""
		}
		release := getMap(p, "release")
		tag := getString(release, "tag_name")
		name := getString(release, "name")
		url := getString(release, "html_url")
		label := tag
		if name != "" && name != tag {
			label = name + " (" + tag + ")"
		}
		return fmt.Sprintf("🚀 [%s] %s 发布了新版本 %s\n%s", repo, sender, label, url)

	case "workflow_run":
		action := getString(p, "action")
		if action != "completed" {
			return ""
		}
		wf := getMap(p, "workflow_run")
		name := getString(wf, "name")
		conclusion := getString(wf, "conclusion")
		branch := getString(wf, "head_branch")
		url := getString(wf, "html_url")

		emoji := "✅"
		if conclusion != "success" {
			emoji = "❌"
		}
		return fmt.Sprintf("%s [%s] CI %s %s (分支 %s)\n%s", emoji, repo, name, conclusion, branch, url)

	case "create":
		refType := getString(p, "ref_type")
		ref := getString(p, "ref")
		return fmt.Sprintf("🌿 [%s] %s 创建了%s %s", repo, sender, refType, ref)

	case "delete":
		refType := getString(p, "ref_type")
		ref := getString(p, "ref")
		return fmt.Sprintf("🗑️ [%s] %s 删除了%s %s", repo, sender, refType, ref)

	case "star":
		action := getString(p, "action")
		if action != "created" {
			return ""
		}
		return fmt.Sprintf("⭐ [%s] %s star 了仓库", repo, sender)

	case "fork":
		forkee := getMap(p, "forkee")
		forkName := getString(forkee, "full_name")
		return fmt.Sprintf("🍴 [%s] %s fork 了仓库 → %s", repo, sender, forkName)

	default:
		return ""
	}
}

// JSON helper functions

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getNumber(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		if n, ok := v.(float64); ok {
			return n
		}
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if sub, ok := v.(map[string]any); ok {
			return sub
		}
	}
	return map[string]any{}
}

func getArray(m map[string]any, key string) []any {
	if v, ok := m[key]; ok {
		if arr, ok := v.([]any); ok {
			return arr
		}
	}
	return nil
}

func toMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

// jsonStr extracts a nested string value like jsonStr(p, "repository", "full_name").
func jsonStr(m map[string]any, keys ...string) string {
	current := m
	for i, key := range keys {
		if i == len(keys)-1 {
			return getString(current, key)
		}
		current = getMap(current, key)
	}
	return ""
}
