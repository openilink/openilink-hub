package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openilink/openilink-hub/internal/store"
)

type mcpImportRequest struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

type mcpImportResult struct {
	ServerName    string          `json:"server_name,omitempty"`
	ServerVersion string          `json:"server_version,omitempty"`
	Tools         []store.AppTool `json:"tools"`
}

// POST /api/apps/import-mcp — discover tools from a remote MCP server
func (s *Server) handleImportMCP(w http.ResponseWriter, r *http.Request) {
	var req mcpImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	result, err := discoverMCPTools(ctx, req.URL, req.Headers)
	if err != nil {
		jsonError(w, "failed to connect to MCP server: "+err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func discoverMCPTools(ctx context.Context, serverURL string, headers map[string]string) (*mcpImportResult, error) {
	opts := []transport.StreamableHTTPCOption{
		transport.WithHTTPTimeout(10 * time.Second),
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

	initResult, err := c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ClientInfo: mcp.Implementation{
				Name:    "OpeniLink Hub",
				Version: "1.0.0",
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
		return result, nil
	}

	for _, t := range toolsResult.Tools {
		appTool := store.AppTool{
			Name:        t.Name,
			Description: t.Description,
		}

		params, _ := json.Marshal(t.InputSchema)
		if len(params) > 0 && string(params) != `{"type":""}` && string(params) != `{"type":"object"}` {
			appTool.Parameters = params
		}

		result.Tools = append(result.Tools, appTool)
	}

	return result, nil
}
