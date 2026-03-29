package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

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

	// send_message — send text or media through the Bot
	mcpSrv.AddTool(
		mcp.NewTool("send_message",
			mcp.WithDescription("Send a message through the Bot to a contact. Supports text and media (image/video/file/voice). For text: set type=text and provide content. For media: set type to image/video/file/voice and provide either url or base64."),
			mcp.WithString("to", mcp.Description("Recipient user ID (use list_contacts to find IDs)"), mcp.Required()),
			mcp.WithString("type", mcp.Description("Message type"), mcp.DefaultString("text"), mcp.Enum("text", "image", "video", "file", "voice")),
			mcp.WithString("content", mcp.Description("Text content, required when type=text")),
			mcp.WithString("url", mcp.Description("URL to download media from, for non-text types")),
			mcp.WithString("base64", mcp.Description("Base64-encoded media data (supports data:mime;base64,... URI), for non-text types")),
			mcp.WithString("filename", mcp.Description("File name for the media attachment")),
		),
		s.mcpSendMessage,
	)

	// list_contacts — list recent contacts with user IDs
	mcpSrv.AddTool(
		mcp.NewTool("list_contacts",
			mcp.WithDescription("List the Bot's recent contacts. Returns user_id, last_msg_at, and msg_count for each contact. Use the user_id as the 'to' parameter in send_message."),
		),
		s.mcpListContacts,
	)

	// get_bot_info — get Bot status and metadata
	mcpSrv.AddTool(
		mcp.NewTool("get_bot_info",
			mcp.WithDescription("Get Bot information including name, provider, connection status, and message count."),
		),
		s.mcpGetBotInfo,
	)

	httpHandler := server.NewStreamableHTTPServer(mcpSrv,
		server.WithStateLess(true),
		server.WithEndpointPath("/mcp"),
	)

	return s.mcpAuth(httpHandler)
}

// mcpAuth wraps an http.Handler with MCP token authentication and request logging.
func (s *Server) mcpAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Extract token from Authorization header only (no query param to avoid credential leaks)
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			http.Error(w, "empty token", http.StatusUnauthorized)
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

		slog.Info("mcp: request", "method", r.Method, "inst", inst.ID, "bot", inst.BotID, "duration_ms", time.Since(start).Milliseconds())
	})
}

// mcpSendMessage handles the send_message MCP tool call.
// Mirrors handleBotAPISend: supports text and media, ObjectStore, tracing.
func (s *Server) mcpSendMessage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	inst := installationFromContext(ctx)
	if inst == nil {
		return mcp.NewToolResultError("unauthorized"), nil
	}

	if !s.requireScope(inst, "message:write") {
		return mcp.NewToolResultError("missing scope: message:write"), nil
	}

	to := req.GetString("to", "")
	msgType := req.GetString("type", "text")
	content := req.GetString("content", "")
	mediaURL := req.GetString("url", "")
	mediaBase64 := req.GetString("base64", "")
	fileName := req.GetString("filename", "")

	if to == "" {
		return mcp.NewToolResultError("'to' is required"), nil
	}
	if msgType == "text" && content == "" {
		return mcp.NewToolResultError("'content' is required for text messages"), nil
	}
	if msgType != "text" && content == "" && mediaURL == "" && mediaBase64 == "" {
		return mcp.NewToolResultError("content, url, or base64 is required"), nil
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

	traceID := generateTraceID()
	contextToken := s.Store.GetLatestContextToken(inst.BotID)

	// Build outbound message
	outMsg := provider.OutboundMessage{
		Recipient:    to,
		ContextToken: contextToken,
	}

	itemType := msgType
	if msgType == "text" {
		outMsg.Text = content
	} else {
		// Media message: resolve data from base64, url, or content fallback
		var mediaData []byte
		if mediaBase64 != "" {
			var decErr error
			var mime string
			mediaData, mime, decErr = base64Decode(mediaBase64)
			if decErr != nil {
				return mcp.NewToolResultError("invalid base64: " + decErr.Error()), nil
			}
			if mime != "" && fileName == "" {
				fileName = defaultFileNameFromMIME(mime)
			}
		} else if mediaURL != "" {
			var dlErr error
			var mime string
			mediaData, mime, dlErr = downloadURL(ctx, mediaURL)
			if dlErr != nil {
				return mcp.NewToolResultError("download failed: " + dlErr.Error()), nil
			}
			if mime != "" && fileName == "" {
				fileName = defaultFileNameFromMIME(mime)
			}
		} else {
			// Fallback: send content as text
			outMsg.Text = content
			itemType = "text"
		}
		if mediaData != nil {
			outMsg.Data = mediaData
			outMsg.FileName = fileName
			if outMsg.FileName == "" {
				outMsg.FileName = defaultFileName(msgType, mediaData)
			}
		}
	}

	clientID, err := botInst.Send(ctx, outMsg)
	if err != nil {
		slog.Error("mcp: send failed", "bot_id", inst.BotID, "err", err)
		return mcp.NewToolResultError("send failed: " + err.Error()), nil
	}

	// Store media to ObjectStore if available
	mediaStatus := ""
	mediaKeys := json.RawMessage(`{}`)
	mediaKey := ""
	if len(outMsg.Data) > 0 && s.ObjectStore != nil {
		ct := detectContentType(itemType)
		ext := detectExt(outMsg.FileName, itemType)
		now := time.Now()
		var rnd [4]byte
		rand.Read(rnd[:])
		key := fmt.Sprintf("%s/%s/out_%d_%x%s", inst.BotID,
			now.Format("2006/01/02"), now.UnixMilli(), rnd, ext)
		if _, err := s.ObjectStore.Put(ctx, key, ct, outMsg.Data); err == nil {
			mediaStatus = "ready"
			mediaKeys, _ = json.Marshal(map[string]string{"0": key})
			mediaKey = key
		} else {
			slog.Warn("mcp: objectstore put failed", "key", key, "err", err)
		}
	}

	// Save outbound message to DB
	item := map[string]any{"type": itemType}
	if outMsg.Text != "" {
		item["text"] = outMsg.Text
	}
	if outMsg.FileName != "" {
		item["file_name"] = outMsg.FileName
	}
	itemList, err := json.Marshal([]any{item})
	if err != nil {
		slog.Warn("mcp: failed to marshal message item", "err", err)
	}
	if _, err := s.Store.SaveMessage(&store.Message{
		BotID:       inst.BotID,
		Direction:   "outbound",
		ToUserID:    to,
		MessageType: 2,
		ItemList:    itemList,
		MediaStatus: mediaStatus,
		MediaKeys:   mediaKeys,
	}); err != nil {
		slog.Warn("mcp: failed to save outbound message", "bot_id", inst.BotID, "err", err)
	}

	// Append span to message trace
	replyContent := content
	if msgType != "text" {
		replyContent = "[" + msgType + "] " + outMsg.FileName
	}
	spanAttrs := map[string]any{
		"app.name":      inst.AppName,
		"reply.type":    itemType,
		"reply.to":      to,
		"reply.content": replyContent,
	}
	if mediaKey != "" {
		spanAttrs["reply.media_key"] = mediaKey
	}
	_ = s.Store.AppendSpan(traceID, inst.BotID, "MCP send_message", store.SpanKindServer, store.StatusOK, "", spanAttrs)

	slog.Info("mcp: message sent", "bot_id", inst.BotID, "to", to, "type", itemType, "client_id", clientID)
	return mcp.NewToolResultText(fmt.Sprintf("Message sent successfully (client_id: %s, trace_id: %s)", clientID, traceID)), nil
}

// mcpListContacts handles the list_contacts MCP tool call.
func (s *Server) mcpListContacts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	inst := installationFromContext(ctx)
	if inst == nil {
		return mcp.NewToolResultError("unauthorized"), nil
	}

	if !s.requireScope(inst, "contact:read") {
		return mcp.NewToolResultError("missing scope: contact:read"), nil
	}

	contacts, err := s.Store.ListRecentContacts(inst.BotID, 100)
	if err != nil {
		slog.Error("mcp: list contacts failed", "bot_id", inst.BotID, "err", err)
		return mcp.NewToolResultError("failed to list contacts"), nil
	}

	data, err := json.Marshal(contacts)
	if err != nil {
		return mcp.NewToolResultError("failed to serialize contacts"), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// mcpGetBotInfo handles the get_bot_info MCP tool call.
func (s *Server) mcpGetBotInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	inst := installationFromContext(ctx)
	if inst == nil {
		return mcp.NewToolResultError("unauthorized"), nil
	}

	if !s.requireScope(inst, "bot:read") {
		return mcp.NewToolResultError("missing scope: bot:read"), nil
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
	data, err := json.Marshal(info)
	if err != nil {
		return mcp.NewToolResultError("failed to serialize bot info"), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
