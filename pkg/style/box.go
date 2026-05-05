package style

import "strings"

// FormatKudosCard returns a pre-colored kudos card using the palette's
// NoChangesBorder for the border and leaving body text unmodified. The
// title is expected to already contain any per-character coloring (e.g.
// rainbow). Width controls the length of the border line.
func FormatKudosCard(coloredTitle, body string, width int) string {
    if width <= 0 {
        width = 42
    }
    var b strings.Builder
    b.WriteString("\n")
    b.WriteString(coloredTitle)
    b.WriteString("\n")
    border := NoChangesBorder.Render(strings.Repeat("─", width))
    b.WriteString(border)
    b.WriteString("\n")
    b.WriteString(body)
    b.WriteString("\n")
    b.WriteString(border)
    b.WriteString("\n\n")
    return b.String()
}
