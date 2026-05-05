package style

// Style represents a color/ANSI wrapper.
type Style struct {
	seq string
}

func NewStyle(seq string) Style {
	return Style{seq: seq}
}

func (s Style) Render(x string) string {
	return s.seq + x + Reset
}

var (
	// Core color role styles
	Error       = NewStyle(DefaultColors.Error)
	Warning     = NewStyle(DefaultColors.Warning)
	Info        = NewStyle(DefaultColors.Info)
	DimStyle    = NewStyle(DefaultColors.Dim)
	Bold        = NewStyle(BoldSeq)
	RowSuccess  = NewStyle(DefaultColors.RowSuccess)
	RowError    = NewStyle(DefaultColors.RowError)
	RowWarning  = NewStyle(DefaultColors.RowWarning)
	Success     = NewStyle(DefaultColors.Success)
	Header      = NewStyle(DefaultColors.Header)
	TableHeader = NewStyle(DefaultColors.TableHeader)

	// Banner/info boxes
	SuccessBox = NewStyle(DefaultColors.SuccessBox)
	InfoBox    = NewStyle(DefaultColors.InfoBox)

	// Plan/table/diff roles
	TableStatusAdd     = NewStyle(DefaultColors.TableStatusAdd)
	TableStatusRemove  = NewStyle(DefaultColors.TableStatusRemove)
	TableStatusUpdate  = NewStyle(DefaultColors.TableStatusUpdate)
	TableStatusCleanup = NewStyle(DefaultColors.TableStatusCleanup)
	TableStatusDrift   = NewStyle(DefaultColors.TableStatusDrift)

	DiffBadgeAdd    = NewStyle(DefaultColors.DiffBadgeAdd)
	DiffBadgeRemove = NewStyle(DefaultColors.DiffBadgeRemove)
	DiffBadgeUpdate = NewStyle(DefaultColors.DiffBadgeUpdate)
	DiffPath        = NewStyle(DefaultColors.DiffPath)
	DiffProvider    = NewStyle(DefaultColors.DiffProvider)

	// Combined header-kind styles
	HeaderKindAdd    = NewStyle(DefaultColors.HeaderKindAdd)
	HeaderKindRemove = NewStyle(DefaultColors.HeaderKindRemove)
	HeaderKindUpdate = NewStyle(DefaultColors.HeaderKindUpdate)

	// No changes
	NoChangesBorder = NewStyle(DefaultColors.NoChangesBorder)
)

// Extra bg/fg for legacy compatibility
const (
	BgBlack = "\033[40m"
	White   = "\033[97m"
)

// Emoji/icon constants
const (
	IconSuccess = "✔"
	IconError   = "✖"
	IconWarning = "⚠"
	IconInfo    = "ℹ"
	EmojiInfo   = "🛈"
	EmojiAdd    = "➕"
	EmojiRemove = "➖"
	EmojiUpdate = "✎"
)

// Styled icons - decorated with color for tables/plans
var (
	StyledIconAdd    = TableStatusAdd.Render(EmojiAdd)
	StyledIconRemove = TableStatusRemove.Render(EmojiRemove)
	StyledIconUpdate = TableStatusUpdate.Render(EmojiUpdate)
	// Styled icons for generic success/error states
	StyledIconSuccess = Success.Render(IconSuccess)
	StyledIconError   = Error.Render(IconError)
)

// Main exported palette (singleton)
var DefaultColors = DefaultPalette()

// NoChanges rainbow function
func RenderNoChangesRainbowChar(ch rune, idx int) string {
	return DefaultColors.GetRainbow(idx) + string(ch) + Reset
}
