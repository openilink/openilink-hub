package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// TestHandleUpdateInstallation_ConfigUnwrap verifies that the write-path
// unwrap logic in handleUpdateInstallation stores the same canonical JSON
// regardless of whether the client sends a raw object or a double-encoded
// string, and that non-object JSON strings are NOT unwrapped.
func TestHandleUpdateInstallation_ConfigUnwrap(t *testing.T) {
	env := setupTestEnv(t)
	bot := createTestBot(t, env.store, env.user.ID, "unwrap-bot")
	app := createTestApp(t, env.store, env.user.ID, "unwrap-app", "unwrap-app", []string{"config:write"})
	inst := installTestApp(t, env.store, app.ID, bot.ID)

	basePath := fmt.Sprintf("/api/apps/%s/installations/%s", app.ID, inst.ID)

	// readConfig fetches the stored config via GET.
	readConfig := func(t *testing.T) json.RawMessage {
		t.Helper()
		resp := doJSON(t, env.ts, "GET", basePath, nil, withCookie(env.cookie))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET installation: expected 200, got %d", resp.StatusCode)
		}
		var result struct {
			Config json.RawMessage `json:"config"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode GET response: %v", err)
		}
		return result.Config
	}

	// putConfig sends a PUT with the raw body bytes as the "config" field.
	putConfig := func(t *testing.T, configValue any) {
		t.Helper()
		body := map[string]any{"config": configValue}
		resp := doJSON(t, env.ts, "PUT", basePath, body, withCookie(env.cookie))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("PUT installation: expected 200, got %d", resp.StatusCode)
		}
	}

	const wantForwardURL = "https://example.com/webhook"

	t.Run("normal object config is stored as-is", func(t *testing.T) {
		putConfig(t, map[string]string{"forward_url": wantForwardURL})

		got := readConfig(t)
		var parsed map[string]string
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatalf("stored config is not a JSON object: %v (raw: %s)", err, got)
		}
		if parsed["forward_url"] != wantForwardURL {
			t.Errorf("forward_url = %q, want %q", parsed["forward_url"], wantForwardURL)
		}
	})

	t.Run("double-encoded object config is unwrapped", func(t *testing.T) {
		// Simulate the legacy frontend bug: JSON.stringify called twice, so the
		// wire value is a JSON string whose contents are the real object.
		inner, _ := json.Marshal(map[string]string{"forward_url": wantForwardURL})
		doubleEncoded := string(inner) // will be marshalled again as a JSON string by doJSON

		putConfig(t, doubleEncoded)

		got := readConfig(t)
		var parsed map[string]string
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatalf("double-encoded config was not unwrapped to object: %v (raw: %s)", err, got)
		}
		if parsed["forward_url"] != wantForwardURL {
			t.Errorf("forward_url = %q, want %q", parsed["forward_url"], wantForwardURL)
		}
	})

	t.Run("JSON string that is not an object is NOT unwrapped", func(t *testing.T) {
		// Sending config: "[]" — a JSON string whose contents are an array.
		// The unwrap guard (unwrapped[0] == '{') must prevent clobbering.
		putConfig(t, "[]")

		got := readConfig(t)
		// The stored value must still be the literal string "[]", not the
		// parsed array []. That is: the raw bytes should be `"[]"` not `[]`.
		if string(got) != `"[]"` {
			t.Errorf("non-object JSON string was incorrectly unwrapped: got %s, want %q", got, `"[]"`)
		}
	})
}
