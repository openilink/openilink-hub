package api

import (
	"encoding/json"

	"github.com/openilink/openilink-hub/internal/bot"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/relay"
)

func parseRelayItems(itemList json.RawMessage, apiKey string) []relay.MessageItem {
	var items []provider.MessageItem
	if err := json.Unmarshal(itemList, &items); err != nil {
		return nil
	}
	return bot.ConvertRelayItems(items, apiKey)
}

func marshalChannelItemList(itemList json.RawMessage, apiKey string) json.RawMessage {
	items := parseRelayItems(itemList, apiKey)
	if items == nil {
		return itemList
	}
	encoded, err := json.Marshal(items)
	if err != nil {
		return itemList
	}
	return encoded
}
