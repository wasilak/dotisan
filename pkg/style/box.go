package style

import "strings"
import "fmt"

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

// StyledIconText composes an icon (already styled) and a text styled via s.
// It ensures consistent spacing and returns a ready-to-print string.
func StyledIconText(icon string, s Style, text string) string {
    return fmt.Sprintf("%s %s", icon, s.Render(text))
}

// Iconf formats text using format and args, then styles it with s and prefixes with icon.
func Iconf(icon string, s Style, format string, a ...interface{}) string {
    txt := fmt.Sprintf(format, a...)
    return fmt.Sprintf("%s %s", icon, s.Render(txt))
}

// PromptPrefix formats a prompt title with an inline hint of [y/N]: and returns
// the ready-to-print prompt string. Use fmt.Print(style.PromptPrefix("Message"))
// when prompting the user for confirmation.
func PromptPrefix(title string) string {
    // Use DimStyle for the bracketed hint and Header for the title.
    return Header.Render(title) + " " + DimStyle.Render("[y/N]: ")
}
