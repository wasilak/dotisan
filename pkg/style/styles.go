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

	// (TableLine declared below after NoChangesBorder so it aliases the
	// current palette value. This avoids duplicated definitions.)

	// Banner/info boxes
	SuccessBox = NewStyle(DefaultColors.SuccessBox)
	InfoBox    = NewStyle(DefaultColors.InfoBox)

	// Plan/table/diff roles
	TableStatusAdd     = NewStyle(DefaultColors.TableStatusAdd)
	TableStatusRemove  = NewStyle(DefaultColors.TableStatusRemove)
	TableStatusUpdate  = NewStyle(DefaultColors.TableStatusUpdate)
	TableStatusCleanup = NewStyle(DefaultColors.TableStatusCleanup)
	TableStatusDrift   = NewStyle(DefaultColors.TableStatusDrift)
	// Table cell color (names/info)
	TableCell = NewStyle(DefaultColors.TableCell)

	// Group label style
	GroupLabel = NewStyle(DefaultColors.GroupLabel)

	// Version color (separate role for package versions)
	VersionColor = NewStyle(DefaultColors.VersionColor)

	DiffBadgeAdd    = NewStyle(DefaultColors.DiffBadgeAdd)
	DiffBadgeRemove = NewStyle(DefaultColors.DiffBadgeRemove)
	DiffBadgeUpdate = NewStyle(DefaultColors.DiffBadgeUpdate)

	DiffBadgeAddBg    = NewStyle(DefaultColors.DiffBadgeAddBg)
	DiffBadgeRemoveBg = NewStyle(DefaultColors.DiffBadgeRemoveBg)
	DiffBadgeUpdateBg = NewStyle(DefaultColors.DiffBadgeUpdateBg)

	TableStatusAddBg    = NewStyle(DefaultColors.TableStatusAddBg)
	TableStatusRemoveBg = NewStyle(DefaultColors.TableStatusRemoveBg)
	TableStatusUpdateBg = NewStyle(DefaultColors.TableStatusUpdateBg)
	DiffPath            = NewStyle(DefaultColors.DiffPath)
	DiffProvider        = NewStyle(DefaultColors.DiffProvider)

	// Combined header-kind styles
	HeaderKindAdd    = NewStyle(DefaultColors.HeaderKindAdd)
	HeaderKindRemove = NewStyle(DefaultColors.HeaderKindRemove)
	HeaderKindUpdate = NewStyle(DefaultColors.HeaderKindUpdate)

	// No changes
	NoChangesBorder = NewStyle(DefaultColors.NoChangesBorder)
	// Decorative accent for the no-changes banner (separate from the
	// structural border). Use this for top/bottom decorative lines.
	NoChangesAccent = NewStyle(DefaultColors.NoChangesAccent)
	// Generic border style (alias of NoChangesBorder).
	Border = NoChangesBorder
	// TableLine alias of the palette border role so callers can refer to the
	// table-specific name if preferred.
	TableLine = NewStyle(DefaultColors.NoChangesBorder)

	// Extra bg/fg convenience styles (wrap palette values for callers)
	BgBlack = NewStyle(DefaultColors.BgBlack)
	White   = NewStyle(DefaultColors.White)
)

// RefreshStyles reapplies DefaultColors to the exported Style wrappers. Call
// this after mutating DefaultColors (e.g., via ApplyPalette) so styles will
// reflect the updated palette values at render time.
func RefreshStyles() {
	Error = NewStyle(DefaultColors.Error)
	Warning = NewStyle(DefaultColors.Warning)
	Info = NewStyle(DefaultColors.Info)
	DimStyle = NewStyle(DefaultColors.Dim)
	RowSuccess = NewStyle(DefaultColors.RowSuccess)
	RowError = NewStyle(DefaultColors.RowError)
	RowWarning = NewStyle(DefaultColors.RowWarning)
	Success = NewStyle(DefaultColors.Success)
	Header = NewStyle(DefaultColors.Header)
	TableHeader = NewStyle(DefaultColors.TableHeader)

	SuccessBox = NewStyle(DefaultColors.SuccessBox)
	InfoBox = NewStyle(DefaultColors.InfoBox)

	TableStatusAdd = NewStyle(DefaultColors.TableStatusAdd)
	TableStatusRemove = NewStyle(DefaultColors.TableStatusRemove)
	TableStatusUpdate = NewStyle(DefaultColors.TableStatusUpdate)
	TableStatusCleanup = NewStyle(DefaultColors.TableStatusCleanup)
	TableStatusDrift = NewStyle(DefaultColors.TableStatusDrift)

	// Table cell color (names/info)
	TableCell = NewStyle(DefaultColors.TableCell)

	// Group label style
	GroupLabel = NewStyle(DefaultColors.GroupLabel)

	// Version color (separate role for package versions)
	VersionColor = NewStyle(DefaultColors.VersionColor)

	DiffBadgeAdd = NewStyle(DefaultColors.DiffBadgeAdd)
	DiffBadgeRemove = NewStyle(DefaultColors.DiffBadgeRemove)
	DiffBadgeUpdate = NewStyle(DefaultColors.DiffBadgeUpdate)

	DiffBadgeAddBg = NewStyle(DefaultColors.DiffBadgeAddBg)
	DiffBadgeRemoveBg = NewStyle(DefaultColors.DiffBadgeRemoveBg)
	DiffBadgeUpdateBg = NewStyle(DefaultColors.DiffBadgeUpdateBg)

	TableStatusAddBg = NewStyle(DefaultColors.TableStatusAddBg)
	TableStatusRemoveBg = NewStyle(DefaultColors.TableStatusRemoveBg)
	TableStatusUpdateBg = NewStyle(DefaultColors.TableStatusUpdateBg)
	DiffPath = NewStyle(DefaultColors.DiffPath)
	DiffProvider = NewStyle(DefaultColors.DiffProvider)

	HeaderKindAdd = NewStyle(DefaultColors.HeaderKindAdd)
	HeaderKindRemove = NewStyle(DefaultColors.HeaderKindRemove)
	HeaderKindUpdate = NewStyle(DefaultColors.HeaderKindUpdate)

	NoChangesBorder = NewStyle(DefaultColors.NoChangesBorder)
	NoChangesAccent = NewStyle(DefaultColors.NoChangesAccent)
	// Border remains the same as NoChangesBorder (used for banners).
	Border = NewStyle(DefaultColors.NoChangesBorder)
	// TableLine should remain the canonical table/box border SGR 34 so that
	// table rendering and tree outputs maintain a consistent structural
	// appearance across the UI and tests that assert on SGR 34 continue to
	// pass. Keep TableLine pinned to SGR 34 (blue) regardless of the
	// NoChangesBorder accent used for the decorative banner.
	TableLine = NewStyle("\033[34m")
}

// ApplyPalette replaces the global DefaultColors palette and refreshes the
// package-level style wrappers so subsequent Render calls use the new palette.
func ApplyPalette(p ColorPalette) {
	DefaultColors = p
	RefreshStyles()
}

// (legacy ANSI constants removed — use style.BgBlack and style.White wrappers)

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
