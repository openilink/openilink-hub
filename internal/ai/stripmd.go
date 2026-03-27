package ai

import "regexp"

var (
	reCodeBlock  = regexp.MustCompile("(?s)```.*?```")
	reBold1      = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	// reBold2 captures surrounding non-alnum boundary chars so they are restored in replacement,
	// preventing the pattern from eating adjacent spaces or mangling __dunder__ identifiers.
	reBold2      = regexp.MustCompile(`([^a-zA-Z0-9]|^)__([^_\n]+)__([^a-zA-Z0-9]|$)`)
	reItalic1    = regexp.MustCompile(`\*([^*]+)\*`)
	// reItalic2 uses the same boundary-preserving technique to avoid mangling snake_case.
	reItalic2    = regexp.MustCompile(`([^a-zA-Z0-9]|^)_([^_\n]+)_([^a-zA-Z0-9]|$)`)
	reStrike     = regexp.MustCompile(`~~(.*?)~~`)
	reInlineCode = regexp.MustCompile("`([^`]+)`")
	reAtxHeader  = regexp.MustCompile(`(?m)^#{1,6}\s+(.+?)\s*#*$`)
	reImage      = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	reLink       = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	reBlockquote = regexp.MustCompile(`(?m)^>\s?`)
	reListLeader = regexp.MustCompile(`(?m)^[\s\t]*(?:[-*+]|\d+\.)\s+`)
	reSetext     = regexp.MustCompile(`(?m)^[=\-]{2,}\s*$`)
	reMultiNL    = regexp.MustCompile(`\n{3,}`)
)

// StripMarkdown removes Markdown formatting from s and returns plain text.
// Formatting characters are stripped; link and image alt text is preserved.
func StripMarkdown(s string) string {
	s = reCodeBlock.ReplaceAllString(s, "")
	s = reBold1.ReplaceAllString(s, "$1")
	s = reBold2.ReplaceAllString(s, "${1}${2}${3}") // $1 and $3 restore boundary chars
	s = reBold2.ReplaceAllString(s, "${1}${2}${3}") // second pass for adjacent spans
	s = reItalic1.ReplaceAllString(s, "$1")
	s = reItalic2.ReplaceAllString(s, "${1}${2}${3}") // $1 and $3 restore boundary chars
	s = reItalic2.ReplaceAllString(s, "${1}${2}${3}") // second pass for adjacent spans
	s = reStrike.ReplaceAllString(s, "$1")
	s = reInlineCode.ReplaceAllString(s, "$1")
	s = reAtxHeader.ReplaceAllString(s, "$1")
	s = reImage.ReplaceAllString(s, "$1")
	s = reLink.ReplaceAllString(s, "$1")
	s = reBlockquote.ReplaceAllString(s, "  ")
	s = reListLeader.ReplaceAllString(s, "")
	s = reSetext.ReplaceAllString(s, "")
	s = reMultiNL.ReplaceAllString(s, "\n\n")
	return s
}
