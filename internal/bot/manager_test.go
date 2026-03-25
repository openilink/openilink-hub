package bot

import (
	"testing"

	"github.com/openilink/openilink-hub/internal/store"
)

func TestParseMentions(t *testing.T) {
	tests := []struct {
		text string
		want []string
	}{
		{"hello", nil},
		{"@客服 你好", []string{"客服"}},
		{"@support hello", []string{"support"}},
		{"@a @b test", []string{"a", "b"}},
		{"no mentions here", nil},
		{"@客服", []string{"客服"}},
		{"hi @bot1 and @bot2 ok", []string{"bot1", "bot2"}},
		{"", nil},
	}

	for _, tt := range tests {
		got := parseMentions(tt.text)
		if len(got) != len(tt.want) {
			t.Errorf("parseMentions(%q) = %v, want %v", tt.text, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseMentions(%q)[%d] = %q, want %q", tt.text, i, got[i], tt.want[i])
			}
		}
	}
}

func TestMatchFilter(t *testing.T) {
	tests := []struct {
		name    string
		rule    store.FilterRule
		sender  string
		text    string
		msgType string
		want    bool
	}{
		{"empty filter matches all", store.FilterRule{}, "user1", "hello", "text", true},
		{"user match", store.FilterRule{UserIDs: []string{"user1"}}, "user1", "hello", "text", true},
		{"user no match", store.FilterRule{UserIDs: []string{"user2"}}, "user1", "hello", "text", false},
		{"keyword match", store.FilterRule{Keywords: []string{"hello"}}, "user1", "Hello World", "text", true},
		{"keyword no match", store.FilterRule{Keywords: []string{"bye"}}, "user1", "Hello World", "text", false},
		{"msgtype match", store.FilterRule{MessageTypes: []string{"image"}}, "user1", "", "image", true},
		{"msgtype no match", store.FilterRule{MessageTypes: []string{"image"}}, "user1", "", "text", false},
		{"combined match", store.FilterRule{UserIDs: []string{"u1"}, Keywords: []string{"hi"}}, "u1", "hi there", "text", true},
		{"combined user fail", store.FilterRule{UserIDs: []string{"u2"}, Keywords: []string{"hi"}}, "u1", "hi there", "text", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchFilter(tt.rule, tt.sender, tt.text, tt.msgType)
			if got != tt.want {
				t.Errorf("matchFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}
