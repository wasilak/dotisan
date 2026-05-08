package ui

import (
	"fmt"
	"github.com/wasilak/nim/pkg/style"
	"math/rand"
	"strings"
	"unicode/utf8"
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

	// Option 2: Rainbow Banner (no table, decorative horizontal lines)
	// Build body lines (preserve paragraphs)
	bodyLines := strings.Split(entry.body, "\n")

	// Compute approximate width based on title and body (rune counts)
	rawTitleLen := utf8.RuneCountInString(entry.title)
	maxLine := rawTitleLen
	for _, l := range bodyLines {
		ln := utf8.RuneCountInString(l)
		if ln > maxLine {
			maxLine = ln
		}
	}
	// Minimum width and padding
	padding := 8
	width := maxLine + padding
	if width < 40 {
		width = 40
	}

	// Top decorative line (colored with palette border role)
	line := strings.Repeat("━", width)
	// left margin so the banner doesn't touch the screen edge
	margin := 4
	prefix := strings.Repeat(" ", margin)
	fmt.Println(prefix)
	// Use decorative accent for the top/bottom lines so the banner has a
	// pop distinct from structural table borders.
	fmt.Println(prefix + style.NoChangesAccent.Render(line))
	fmt.Println(prefix)

	// Center the rainbow title (use uncolored rune length for centering)
	leftPad := (width - rawTitleLen) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	// print per-char rainbow title (already colored)
	fmt.Print(prefix + strings.Repeat(" ", leftPad))
	fmt.Println(title)
	// Add one blank line between the title/header and the body for breathing room
	fmt.Println(prefix)

	// Print body lines, styled as TableCell for consistent colors.
	// Trim any accidental leading indentation from the stored messages so
	// paragraphs line up consistently.
	for _, l := range bodyLines {
		if strings.TrimSpace(l) == "" {
			fmt.Println(prefix)
			continue
		}
		clean := strings.TrimLeft(l, " \t")
		// left align with 4-space indent (after margin)
		fmt.Println(prefix + "    " + style.TableCell.Render(clean))
	}

	// Trailing blank line and bottom decorative line
	fmt.Println(prefix)
	fmt.Println(prefix + style.NoChangesAccent.Render(line))
	fmt.Println(prefix)

}
