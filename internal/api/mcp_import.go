package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openilink/openilink-hub/internal/store"
)

const maxImportTools = 200

var blockedHeaderKeys = map[string]bool{
	"host":            true,
	"content-type":    true,
	"content-length":  true,
	"transfer-encoding": true,
	"connection":      true,
}

type mcpImportRequest struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

type mcpImportResult struct {
	ServerName    string          `json:"server_name,omitempty"`
	ServerVersion string          `json:"server_version,omitempty"`
	Tools         []store.AppTool `json:"tools"`
	Truncated     bool            `json:"truncated,omitempty"`
}

// handleImportMCP discovers tools from a remote MCP server.
func (s *Server) handleImportMCP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	body := http.MaxBytesReader(w, r.Body, 8*1024)
	var req mcpImportRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		jsonError(w, "url is required", http.StatusBadRequest)
		return
	}

	u, err := url.ParseRequestURI(req.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		jsonError(w, "url must be a valid http or https URL", http.StatusBadRequest)
		return
	}

	headers := filterHeaders(req.Headers)

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	result, err := discoverMCPTools(ctx, s.Version, req.URL, headers)
	if err != nil {
		slog.Warn("mcp import failed", "url", req.URL, "err", err)
		jsonError(w, "failed to connect to MCP server", http.StatusBadGateway)
		return
	}

	slog.Info("mcp import", "url", req.URL, "server", result.ServerName, "tools", len(result.Tools), "truncated", result.Truncated, "duration_ms", time.Since(start).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func filterHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	filtered := make(map[string]string, len(headers))
	for k, v := range headers {
		lower := stringToLower(k)
		if !blockedHeaderKeys[lower] {
			filtered[k] = v
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func stringToLower(s string) string {
	b := make([]byte, len(s))
	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func discoverMCPTools(ctx context.Context, hubVersion, serverURL string, headers map[string]string) (*mcpImportResult, error) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: ssrfSafeDialContext,
			ResponseHeaderTimeout: 10 * time.Second,
		},
		Timeout: 15 * time.Second,
	}

	opts := []transport.StreamableHTTPCOption{
		transport.WithHTTPBasicClient(httpClient),
	}
	if len(headers) > 0 {
		opts = append(opts, transport.WithHTTPHeaders(headers))
	}

	c, err := client.NewStreamableHttpClient(serverURL, opts...)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	if err := c.Start(ctx); err != nil {
		return nil, err
	}

	version := hubVersion
	if version == "" {
		version = "dev"
	}

	initResult, err := c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ClientInfo: mcp.Implementation{
				Name:    "OpeniLink Hub",
				Version: version,
			},
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		},
	})
	if err != nil {
		return nil, err
	}

	result := &mcpImportResult{
		Tools: []store.AppTool{},
	}
	if initResult != nil {
		result.ServerName = initResult.ServerInfo.Name
		result.ServerVersion = initResult.ServerInfo.Version
	}

	toolsResult, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}

	for i, t := range toolsResult.Tools {
		if i >= maxImportTools {
			result.Truncated = true
			break
		}

		appTool := store.AppTool{
			Name:        t.Name,
			Description: t.Description,
		}

		params, err := json.Marshal(t.InputSchema)
		if err == nil && len(params) > 0 && string(params) != `{"type":""}` && string(params) != `{"type":"object"}` {
			appTool.Parameters = params
		}

		result.Tools = append(result.Tools, appTool)
	}

	return result, nil
}

