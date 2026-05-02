package diff

import (
	"bytes"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"strings"
)

// HighlightUnifiedDiff highlights a unified diff using Chroma and returns a colorized terminal string.
// Theme can be e.g. "github-dark". Falls back to defaults if not found.
func HighlightUnifiedDiff(diffText, theme string) (string, error) {
	lexer := lexers.Get("diff")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	style := styles.Get(theme)
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}
	// Some formatters/lexers can collapse or mishandle multi-line input
	// in certain terminals. To guarantee newline preservation we format
	// line-by-line using SplitAfter which keeps trailing newlines and
	// format each line separately. This produces identical colorization
	// while ensuring the output keeps the same number of line breaks.
	var buf bytes.Buffer
	lines := strings.SplitAfter(diffText, "\n")
	for _, line := range lines {
		// Tokenise the single line and format it. We ignore tokenise errors
		// for single lines as the lexer should handle diff tokens gracefully.
		it, _ := lexer.Tokenise(nil, line)
		if err := formatter.Format(&buf, style, it); err != nil {
			return "", err
		}
		// Some formatters may omit writing a trailing newline even when the
		// input line contained one. Ensure we preserve line breaks so the
		// highlighted output has the same number of lines as the source.
		if strings.HasSuffix(line, "\n") {
			b := buf.Bytes()
			if len(b) == 0 || b[len(b)-1] != '\n' {
				buf.WriteByte('\n')
			}
		}
	}
	return buf.String(), nil
}
