package ui

import (
	"fmt"
	"math/rand"
	"strings"
	"github.com/wasilak/dotisan/pkg/style"
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

// purpleBorderRGB is the RGB color used for the single-color box border.
// dark-ish purple used for the box border.
// We'll construct the style inline where needed to keep usage obvious.

// RenderNoChanges prints a random rainbow-bordered kudos card.
func RenderNoChanges() {
	entry := noChangesMessages[rand.Intn(len(noChangesMessages))]

	// Per-character rainbow title — now palette-driven
	var b strings.Builder
	idx := 0
	for _, ch := range entry.title {
		if ch != ' ' {
			b.WriteString(style.RenderNoChangesRainbowChar(ch, idx))
			idx++
		} else {
			b.WriteRune(' ')
		}
	}
	title := b.String()
// (Palette rainbow logic applied)


	// Single-color dark purple border and white body text.
	// We build pre-colored border characters using RGB and keep BoxStyle empty
	// so the box printer doesn't re-wrap them (which would mangle RGB escapes).
	// TODO: Replace with new border print logic from color palette
	fmt.Println()
	fmt.Printf("%s\n", title)
	border := style.NoChangesBorder.Render(strings.Repeat("─", 42))
	fmt.Println(border)
	fmt.Println(entry.body)
	fmt.Println(border)

	fmt.Println()
	// fmt.Println(box) // replaced above with direct print for now
	fmt.Println()

}
