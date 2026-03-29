package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/store"
)

// setupMCP creates the MCP server and returns an http.Handler for the /mcp endpoint.
func (s *Server) setupMCP() http.Handler {
	mcpSrv := server.NewMCPServer(
		"OpeniLink Hub",
		s.Version,
		server.WithToolCapabilities(false),
		server.WithInstructions("OpeniLink Hub MCP Server. Use these tools to send messages and manage contacts through your Bot."),
	)

	// send_message tool
	mcpSrv.AddTool(
		mcp.NewTool("send_message",
			mcp.WithDescription("Send a message through the Bot"),
			mcp.WithString("to", mcp.Description("Recipient user ID"), mcp.Required()),
			mcp.WithString("content", mcp.Description("Message text content"), mcp.Required()),
			mcp.WithString("type", mcp.Description("Message type: text, image, video, file"), mcp.DefaultString("text")),
		),
		s.mcpSendMessage,
	)

	// list_contacts tool
	mcpSrv.AddTool(
		mcp.NewTool("list_contacts",
			mcp.WithDescription("List recent contacts of the Bot"),
		),
		s.mcpListContacts,
	)

	// get_bot_info tool
	mcpSrv.AddTool(
		mcp.NewTool("get_bot_info",
			mcp.WithDescription("Get information about the Bot"),
		),
		s.mcpGetBotInfo,
	)

	httpHandler := server.NewStreamableHTTPServer(mcpSrv,
		server.WithStateLess(true),
		server.WithEndpointPath("/mcp"),
	)

	return s.mcpAuth(httpHandler)
}

// mcpAuth wraps an http.Handler with MCP token authentication.
func (s *Server) mcpAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header or query param
		var token string
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		inst, err := s.Store.GetInstallationByToken(token)
		if err != nil || !inst.Enabled {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// Verify this is an MCP Server installation
		if inst.AppSlug != "mcp-server" {
			http.Error(w, "token is not for MCP Server app", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), installationKey, inst)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// mcpSendMessage handles the send_message MCP tool call.
func (s *Server) mcpSendMessage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	inst := installationFromContext(ctx)
	if inst == nil {
		return mcp.NewToolResultError("unauthorized"), nil
	}

	to := req.GetString("to", "")
	content := req.GetString("content", "")
	msgType := req.GetString("type", "text")

	if to == "" {
		return mcp.NewToolResultError("'to' is required"), nil
	}
	if msgType == "text" && content == "" {
		return mcp.NewToolResultError("'content' is required for text messages"), nil
	}

	botInst, ok := s.BotManager.GetInstance(inst.BotID)
	if !ok {
		bot, err := s.Store.GetBot(inst.BotID)
		if err != nil {
			return mcp.NewToolResultError("bot not found"), nil
		}
		if bot.Status == "session_expired" {
			return mcp.NewToolResultError("bot session expired"), nil
		}
		return mcp.NewToolResultError("bot not connected"), nil
	}

	if canSend, reason := s.checkSendability(inst.BotID, botInst.Status()); !canSend {
		return mcp.NewToolResultError(reason), nil
	}

	contextToken := s.Store.GetLatestContextToken(inst.BotID)
	outMsg := provider.OutboundMessage{
		Recipient:    to,
		Text:         content,
		ContextToken: contextToken,
	}

	clientID, err := botInst.Send(ctx, outMsg)
	if err != nil {
		slog.Error("mcp: send failed", "bot_id", inst.BotID, "err", err)
		return mcp.NewToolResultError("send failed: " + err.Error()), nil
	}

	// Save outbound message to DB
	item := map[string]any{"type": msgType, "text": content}
	itemList, _ := json.Marshal([]any{item})
	s.Store.SaveMessage(&store.Message{
		BotID:       inst.BotID,
		Direction:   "outbound",
		ToUserID:    to,
		MessageType: 2,
		ItemList:    itemList,
	})

	slog.Info("mcp: message sent", "bot_id", inst.BotID, "to", to, "client_id", clientID)
	return mcp.NewToolResultText(fmt.Sprintf("Message sent successfully (client_id: %s)", clientID)), nil
}

// mcpListContacts handles the list_contacts MCP tool call.
func (s *Server) mcpListContacts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	inst := installationFromContext(ctx)
	if inst == nil {
		return mcp.NewToolResultError("unauthorized"), nil
	}

	contacts, err := s.Store.ListRecentContacts(inst.BotID, 100)
	if err != nil {
		slog.Error("mcp: list contacts failed", "bot_id", inst.BotID, "err", err)
		return mcp.NewToolResultError("failed to list contacts"), nil
	}

	data, _ := json.Marshal(contacts)
	return mcp.NewToolResultText(string(data)), nil
}

// mcpGetBotInfo handles the get_bot_info MCP tool call.
func (s *Server) mcpGetBotInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	inst := installationFromContext(ctx)
	if inst == nil {
		return mcp.NewToolResultError("unauthorized"), nil
	}

	bot, err := s.Store.GetBot(inst.BotID)
	if err != nil {
		return mcp.NewToolResultError("bot not found"), nil
	}

	status := bot.Status
	if botInst, ok := s.BotManager.GetInstance(inst.BotID); ok {
		status = botInst.Status()
	}

	info := map[string]any{
		"id":         bot.ID,
		"name":       bot.Name,
		"provider":   bot.Provider,
		"status":     status,
		"msg_count":  bot.MsgCount,
		"created_at": bot.CreatedAt,
	}
	data, _ := json.Marshal(info)
	return mcp.NewToolResultText(string(data)), nil
}
