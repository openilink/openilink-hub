package api

import (
	"testing"

	"github.com/openilink/openilink-hub/internal/store"
)

// TestApplyReplyPrefix covers the per-installation reply-prefix behavior
// introduced for #248: when ReplyPrefixHandle is on and Handle is non-empty,
// outbound text gets "@handle " prepended; otherwise text is unchanged.
func TestApplyReplyPrefix(t *testing.T) {
	cases := []struct {
		name string
		inst *store.AppInstallation
		text string
		want string
	}{
		{
			name: "disabled — no prefix",
			inst: &store.AppInstallation{Handle: "claw", ReplyPrefixHandle: false},
			text: "hello",
			want: "hello",
		},
		{
			name: "enabled with handle — prepends",
			inst: &store.AppInstallation{Handle: "claw", ReplyPrefixHandle: true},
			text: "hello",
			want: "@claw hello",
		},
		{
			name: "enabled but handle empty — no prefix",
			inst: &store.AppInstallation{Handle: "", ReplyPrefixHandle: true},
			text: "hello",
			want: "hello",
		},
		{
			name: "idempotent — already prefixed, leaves alone",
			inst: &store.AppInstallation{Handle: "claw", ReplyPrefixHandle: true},
			text: "@claw hello",
			want: "@claw hello",
		},
		{
			name: "different handle in text — still prefixes (no false match)",
			inst: &store.AppInstallation{Handle: "claw", ReplyPrefixHandle: true},
			text: "@other hello",
			want: "@claw @other hello",
		},
		{
			name: "empty text — only prefix",
			inst: &store.AppInstallation{Handle: "claw", ReplyPrefixHandle: true},
			text: "",
			want: "@claw ",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := applyReplyPrefix(tc.inst, tc.text)
			if got != tc.want {
				t.Errorf("applyReplyPrefix(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}
