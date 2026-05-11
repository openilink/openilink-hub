package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type SupabaseClient interface {
	UpsertMessageEvent(ctx context.Context, payload json.RawMessage) error
	UpsertPromptProfileEvent(ctx context.Context, payload json.RawMessage) error
	UpsertBindingInvalidatedEvent(ctx context.Context, payload json.RawMessage) error
}

type HTTPSupabaseClient struct {
	baseURL string
	apiKey  string
	schema  string
	client  *http.Client
}

func NewHTTPSupabaseClient(url, serviceRoleKey, schema string) (*HTTPSupabaseClient, error) {
	url = strings.TrimRight(strings.TrimSpace(url), "/")
	serviceRoleKey = strings.TrimSpace(serviceRoleKey)
	if url == "" || serviceRoleKey == "" {
		return nil, fmt.Errorf("supabase url/key required")
	}
	if schema == "" {
		schema = "public"
	}
	return &HTTPSupabaseClient{
		baseURL: url,
		apiKey:  serviceRoleKey,
		schema:  schema,
		client:  &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (c *HTTPSupabaseClient) UpsertMessageEvent(ctx context.Context, payload json.RawMessage) error {
	return c.upsert(ctx, "openilink_mirror_messages", payload)
}

func (c *HTTPSupabaseClient) UpsertPromptProfileEvent(ctx context.Context, payload json.RawMessage) error {
	return c.upsert(ctx, "openilink_mirror_prompt_profiles", payload)
}

func (c *HTTPSupabaseClient) UpsertBindingInvalidatedEvent(ctx context.Context, payload json.RawMessage) error {
	return c.upsert(ctx, "openilink_mirror_binding_invalidations", payload)
}

func (c *HTTPSupabaseClient) upsert(ctx context.Context, table string, payload json.RawMessage) error {
	endpoint := fmt.Sprintf("%s/rest/v1/%s", c.baseURL, table)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader("["+string(payload)+"]"))
	if err != nil {
		return err
	}
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "resolution=merge-duplicates,return=minimal")
	req.Header.Set("Accept-Profile", c.schema)
	req.Header.Set("Content-Profile", c.schema)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("supabase upsert failed: status=%d", resp.StatusCode)
	}
	return nil
}
