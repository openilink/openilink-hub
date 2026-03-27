package ai

import (
	"testing"
)

func TestStripMarkdown(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"bold", "**hello**", "hello"},
		{"italic star", "*hello*", "hello"},
		{"italic underscore", "_hello_", "hello"},
		{"bold underscore", "__hello__", "hello"},
		{"atx header", "# Hello World", "Hello World"},
		{"atx header h2", "## Hello", "Hello"},
		{"inline code", "`code`", "code"},
		{"code block", "```\ncode\n```", ""},
		{"link", "[text](http://example.com)", "text"},
		{"image", "![alt](http://example.com/img.png)", "alt"},
		{"blockquote", "> quote", "  quote"},
		{"strikethrough", "~~text~~", "text"},
		{"unordered list", "- item", "item"},
		{"numbered list", "1. item", "item"},
		{"plain text unchanged", "hello world", "hello world"},
		// Identifiers with underscores must not be mangled
		{"snake_case preserved", "use set_config_value here", "use set_config_value here"},
		{"env var preserved", "set ENV_VAR_NAME=1", "set ENV_VAR_NAME=1"},
		// __dunder__ surrounded by spaces is valid Markdown bold and is stripped
		{"dunder in prose stripped", "call __init__ method", "call init method"},
		// Horizontal rule and setext heading
		{"horizontal rule", "---", ""},
		{"setext heading", "Title\n---", "Title\n"},
		// Spaces adjacent to italic markers must be preserved
		{"italic spaces preserved", "Use _italic_ here", "Use italic here"},
		{"consecutive italic", "a _foo_ and _bar_ b", "a foo and bar b"},
		{"italic at start", "_start_ of line", "start of line"},
		{"italic at end", "end of _line_", "end of line"},
		{"consecutive italic adjacent", "a _x_ _y_ b", "a x y b"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := StripMarkdown(c.input)
			if got != c.want {
				t.Errorf("StripMarkdown(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}
