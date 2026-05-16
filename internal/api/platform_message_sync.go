package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/openilink/openilink-hub/internal/supamemory"
)

func (s *Server) writePlatformOutboundMessage(
	ctx context.Context,
	botID string,
	providerBotID string,
	recipient string,
	contextToken string,
	content string,
	itemList json.RawMessage,
	providerMessageID string,
	meta map[string]any,
) {
	if s == nil || s.SupaMemory == nil {
		return
	}
	botID = strings.TrimSpace(botID)
	if botID == "" {
		return
	}
	bot, err := s.Store.GetBot(botID)
	if err != nil || bot == nil {
		return
	}
	if strings.TrimSpace(providerBotID) == "" {
		providerBotID = strings.TrimSpace(bot.ProviderID)
	}
	if providerBotID == "" || strings.TrimSpace(recipient) == "" {
		return
	}

	bctx, err := s.SupaMemory.ResolveBindingContext(ctx, providerBotID, strings.TrimSpace(recipient))
	if err != nil || bctx == nil {
		return
	}
	if strings.TrimSpace(bctx.UserID) == "" || strings.TrimSpace(bctx.RoleID) == "" {
		return
	}
	conversationID, err := s.SupaMemory.ResolveConversationID(ctx, bctx.UserID, bctx.RoleID, strings.TrimSpace(contextToken), strings.TrimSpace(recipient))
	if err != nil {
		conversationID = ""
	}
	if strings.TrimSpace(conversationID) == "" {
		conversationID = fallbackConversationIDAPI(bctx.UserID, bctx.RoleID, recipient)
	}
	if meta == nil {
		meta = map[string]any{}
	}
	if _, exists := meta["bot_id"]; !exists {
		meta["bot_id"] = botID
	}
	if _, exists := meta["source"]; !exists {
		meta["source"] = "openilink_hub_api"
	}

	if err := s.SupaMemory.WritePlatformMessage(ctx, supamemory.PlatformMessageInput{
		UserID:            bctx.UserID,
		RoleID:            bctx.RoleID,
		ConversationID:    conversationID,
		Platform:          "openilink",
		Direction:         "outbound",
		Role:              "assistant",
		Content:           strings.TrimSpace(content),
		ItemList:          itemList,
		ProviderMessageID: strings.TrimSpace(providerMessageID),
		ExternalChatID:    strings.TrimSpace(providerBotID),
		ExternalUserID:    strings.TrimSpace(recipient),
		ContextToken:      strings.TrimSpace(contextToken),
		Meta:              meta,
		MessageAt:         time.Now().UTC(),
	}); err != nil {
		slog.Warn("platform outbound sync failed", "bot_id", botID, "recipient", recipient, "err", err)
	}
}

func fallbackConversationIDAPI(userID, roleID, sender string) string {
	u := strings.TrimSpace(userID)
	r := strings.TrimSpace(roleID)
	s := sanitizeTokenAPI(sender)
	if u == "" || r == "" {
		return ""
	}
	if s == "" {
		s = "sender_unknown"
	}
	return fmt.Sprintf("fallback_%s_%s_%s", u, r, s)
}

func sanitizeTokenAPI(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}
	replacer := strings.NewReplacer(" ", "_", "@", "_", ":", "_", "/", "_", "\\", "_")
	out := replacer.Replace(v)
	if len(out) > 80 {
		out = out[:80]
	}
	return out
}
