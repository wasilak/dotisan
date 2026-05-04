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
	{
		title: "🎯  NOTHING TO DO  🎯",
		body: "Zero changes. Absolute zero. The universe\n" +
			"is in perfect thermodynamic equilibrium.\n\n" +
			"   Scientists are baffled. Philosophers weep. 🧪",
	},
	{
		title: "🏆  DOTFILES CHAMPION  🏆",
		body: "Your configuration is so well-managed that\n" +
			"even Ansible is a little jealous. 😤\n\n" +
			"   No drift detected. You may rest, hero. ⚔️",
	},
	{
		title: "🤖  ALL SYSTEMS NOMINAL  🤖",
		body: "Ran the numbers. Checked the state.\n" +
			"Everything matches. I have nothing to do.\n\n" +
			"      This is fine. 🔥  (No, really, it is.)",
	},
	{
		title: "🧠  BIG BRAIN ENERGY  🧠",
		body: "Only a truly disciplined engineer keeps\n" +
			"their dotfiles this impeccably in sync.\n\n" +
			"   Or you just imported everything. No judgment. 👀",
	},
	{
		title: "🌈  IDEMPOTENCY ACHIEVED  🌈",
		body: "Apply it once. Apply it twice. Apply it\n" +
			"a thousand times. Same result every time.\n\n" +
			"   You have ascended. 🧘 Nothing to change.",
	},
	{
		title: "💅  FLAWLESS  💅",
		body: "State matches config. Config matches reality.\n" +
			"Reality matches your vision. You matched.\n\n" +
			"      No notes. Truly no notes. 👏",
	},
	{
		title: "🚀  LAUNCH CONDITIONS MET  🚀",
		body: "All systems go. Dotfiles locked in.\n" +
			"Configuration confirmed. Awaiting mission.\n\n" +
			"   (There is no mission. Enjoy the silence.) 🌌",
	},
	{
		title: "😴  BORING IN THE BEST WAY  😴",
		body: "No incidents. No surprises. No changes.\n" +
			"This is the on-call dream. Pure SRE nirvana.\n\n" +
			"   The ops team has left the building. 🏖️",
	},
	{
		title: "🎲  ROLLED A NAT 20  🎲",
		body: "Critical success on the dotfiles check.\n" +
			"The dungeon master nods in silent approval.\n\n" +
			"   Your charisma modifier is: in sync. ⚔️",
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
