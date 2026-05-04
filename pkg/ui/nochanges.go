package ui

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/pterm/pterm"
)

// noChangesEntry holds a title and body for a "nothing to do" kudos card.
// Add new entries to the slice below to expand the rotation — that's all.
type noChangesEntry struct {
	title string
	body  string
}

var noChangesMessages = []noChangesEntry{
	{
		title: "✨  PERFECTLY IN SYNC  ✨",
		body: "Your dotfiles are so tidy, Marie Kondo\n" +
			"shed a single tear of joy. 🧘\n\n" +
			"      No changes. You win. Go touch grass. 🌿",
	},
}

// rainbowColors cycles through terminal hues for character-level rainbow.
var rainbowColors = []pterm.Color{
	pterm.FgRed,
	pterm.FgYellow,
	pterm.FgGreen,
	pterm.FgCyan,
	pterm.FgBlue,
	pterm.FgMagenta,
}

func rainbowString(s string) string {
	var b strings.Builder
	i := 0
	for _, ch := range s {
		if ch != ' ' {
			b.WriteString(pterm.NewStyle(rainbowColors[i%len(rainbowColors)], pterm.Bold).Sprint(string(ch)))
			i++
		} else {
			b.WriteRune(' ')
		}
	}
	return b.String()
}

// RenderNoChanges prints a random rainbow-bordered kudos card.
func RenderNoChanges() {
	entry := noChangesMessages[rand.Intn(len(noChangesMessages))]

	title := rainbowString(entry.title)

	box := pterm.DefaultBox.
		WithTitle(title).
		WithTitleTopCenter().
		WithRightPadding(6).
		WithLeftPadding(6).
		WithTopPadding(1).
		WithBottomPadding(1).
		Sprint(pterm.NewStyle(pterm.FgWhite).Sprint(entry.body))

	fmt.Println()
	fmt.Println(box)
	fmt.Println()
}
