package style

// ANSI color constants for palette roles
const (
	Green     = "\033[32m"
	Red       = "\033[31m"
	Orange    = "\033[38;5;208m" // True orange
	Yellow    = "\033[33m"
	Gray      = "\033[38;5;245m"
	RowGreen  = "\033[48;5;22m"
	RowRed    = "\033[48;5;52m"
	RowYellow = "\033[48;5;94m"
	Magenta   = "\033[35m"
	BoldSeq   = "\033[1m"
	Reset     = "\033[0m"
	Dim       = "\033[2m"
)

// ColorPalette centralizes all color/style mappings for the UI.
type ColorPalette struct {
	// Primary/main color for prominent UI elements (spinner, headers)
	Main         string
	Success      string
	Error        string
	Warning      string
	Info         string
	Dim          string
	RowSuccess   string
	RowError     string
	RowWarning   string
	Header       string
	TableHeader  string
	TableRow     string
	TableCell    string
	VersionColor string

	// Group label color used for resource groups (e.g. homebrew-taps)
	GroupLabel string

	// Combined roles for common pairings
	HeaderKindAdd    string
	HeaderKindRemove string
	HeaderKindUpdate string

	// Box wrappers for banner/info output
	SuccessBox string
	InfoBox    string

	// Plan/diff summary/status/badge roles
	TableStatusAdd     string
	TableStatusRemove  string
	TableStatusUpdate  string
	TableStatusDrift   string
	TableStatusCleanup string

	DiffBadgeAdd    string
	DiffBadgeRemove string
	DiffBadgeUpdate string
	DiffProvider    string
	DiffPath        string

	// Badge + background (common paired roles)
	DiffBadgeAddBg    string
	DiffBadgeRemoveBg string
	DiffBadgeUpdateBg string

	TableStatusAddBg    string
	TableStatusRemoveBg string
	TableStatusUpdateBg string

	// Extra bg/fg convenience roles
	BgBlack string
	White   string

	// No changes (kudos card)
	NoChangesBorder  string
	NoChangesRainbow []string // ANSI codes for per-letter rainbow, palette order
	// Accent used for decorative no-changes banner (separate from border)
	NoChangesAccent string
	// Spinner color name (spinner library color/attribute string)
	SpinnerColor string
}

// DefaultPalette returns a ColorPalette populated with current ANSI codes.
func DefaultPalette() ColorPalette {
	return ColorPalette{
		// Charm-inspired palette
		// Main: Soft Purple
		Main: "\033[38;5;141m",
		// Success: Mint/Cyan
		Success: "\033[38;5;86m",
		// Error: Soft Pink
		Error: "\033[38;5;204m",
		// Warning: Peachy Coral
		Warning: "\033[38;5;209m",
		// Info: Hot Pink
		Info:       "\033[38;5;205m",
		Dim:        Dim,
		RowSuccess: RowGreen,
		RowError:   RowRed,
		RowWarning: RowYellow,
		Header:     BoldSeq,
		// Table header: bold + purple tint
		TableHeader: BoldSeq + "\033[38;5;183m",
		// Combined header+status styles
		HeaderKindAdd:    BoldSeq + "\033[38;5;86m",
		HeaderKindRemove: BoldSeq + "\033[38;5;204m",
		HeaderKindUpdate: BoldSeq + "\033[38;5;209m",
		SuccessBox:       "\033[1;48;5;86m",  // Bold + mint bg
		InfoBox:          "\033[1;48;5;205m", // Bold + hot-pink bg

		TableRow: "",
		// Soft lavender table cells
		TableCell: "\033[38;5;183m",
		// Group labels use peachy coral
		GroupLabel:   "\033[38;5;209m",
		VersionColor: "\033[38;5;205m",
		// Specific status and badge roles
		TableStatusAdd:     "\033[38;5;86m",
		TableStatusRemove:  "\033[38;5;204m",
		TableStatusUpdate:  "\033[38;5;209m",
		TableStatusDrift:   "\033[38;5;183m",
		TableStatusCleanup: Dim,
		DiffBadgeAdd:       "\033[38;5;86m",
		DiffBadgeRemove:    "\033[38;5;204m",
		DiffBadgeUpdate:    "\033[38;5;209m",
		// paired badge + background combos (bold fg + black bg for now)
		DiffBadgeAddBg:    "\033[1m" + "\033[38;5;86m" + "\033[40m",
		DiffBadgeRemoveBg: "\033[1m" + "\033[38;5;204m" + "\033[40m",
		DiffBadgeUpdateBg: "\033[1m" + "\033[38;5;209m" + "\033[40m",

		TableStatusAddBg:    "\033[1m" + "\033[38;5;86m" + "\033[40m",
		TableStatusRemoveBg: "\033[1m" + "\033[38;5;204m" + "\033[40m",
		TableStatusUpdateBg: "\033[1m" + "\033[38;5;209m" + "\033[40m",
		DiffProvider:        Gray,
		DiffPath:            BoldSeq,
		// NoChanges specifics
		// Palette convenience
		BgBlack: "\033[40m",
		White:   "\033[97m",
		// Keep table border blue for structural elements; use a separate
		// decorative accent role for the no-changes banner.
		NoChangesBorder: "\033[34m",
		NoChangesAccent: "\033[38;5;213m",
		// Charm rainbow: Hot Pink, Soft Pink, Lavender, Mint, Cyan, Purple
		NoChangesRainbow: []string{"\033[38;5;205m", "\033[38;5;204m", "\033[38;5;183m", "\033[38;5;86m", "\033[38;5;81m", "\033[38;5;141m"},
		SpinnerColor:     "fgHiMagenta",
	}
}

// Get returns the ANSI code for the given role ("success", "error", ...).
func (p ColorPalette) Get(role string) string {
	switch role {
	case "success":
		return p.Success
	case "error":
		return p.Error
	case "warning":
		return p.Warning
	case "info":
		return p.Info
	case "dim":
		return p.Dim
	case "row_success":
		return p.RowSuccess
	case "row_error":
		return p.RowError
	case "row_warning":
		return p.RowWarning
	case "header":
		return p.Header
	case "table_header":
		return p.TableHeader
	case "table_row":
		return p.TableRow
	case "table_cell":
		return p.TableCell
	case "group_label":
		return p.GroupLabel
	case "version_color":
		return p.VersionColor
	case "table_status_add":
		return p.TableStatusAdd
	case "table_status_remove":
		return p.TableStatusRemove
	case "table_status_update":
		return p.TableStatusUpdate
	case "table_status_drift":
		return p.TableStatusDrift
	case "table_status_cleanup":
		return p.TableStatusCleanup
	case "diff_badge_add":
		return p.DiffBadgeAdd
	case "diff_badge_remove":
		return p.DiffBadgeRemove
	case "diff_badge_update":
		return p.DiffBadgeUpdate
	case "diff_provider":
		return p.DiffProvider
	case "diff_path":
		return p.DiffPath
	case "nochanges_border":
		return p.NoChangesBorder
	case "spinner_color":
		return p.SpinnerColor
	case "bg_black":
		return p.BgBlack
	case "white":
		return p.White
	default:
		return ""
	}
}

// GetRainbow returns the ANSI sequence for the i-th rainbow color
func (p ColorPalette) GetRainbow(i int) string {
	if len(p.NoChangesRainbow) == 0 {
		return ""
	}
	return p.NoChangesRainbow[i%len(p.NoChangesRainbow)]
}

// GetColor returns the ANSI sequence for a named role. Prefer using the
// Style wrappers in pkg/style/styles.go (they wrap these values and provide
// Render helpers), but this is useful for dynamic access or tests.
func (p ColorPalette) GetColor(name string) string {
	return p.Get(name)
}

// SetColor sets the ANSI sequence for a named role on the palette. It does
// nothing if the role is unknown. Use a pointer receiver to mutate the
// palette in-place.
func (p *ColorPalette) SetColor(name, seq string) {
	switch name {
	case "success":
		p.Success = seq
	case "error":
		p.Error = seq
	case "warning":
		p.Warning = seq
	case "info":
		p.Info = seq
	case "dim":
		p.Dim = seq
	case "row_success":
		p.RowSuccess = seq
	case "row_error":
		p.RowError = seq
	case "row_warning":
		p.RowWarning = seq
	case "header":
		p.Header = seq
	case "table_header":
		p.TableHeader = seq
	case "table_row":
		p.TableRow = seq
	case "table_cell":
		p.TableCell = seq
	case "group_label":
		p.GroupLabel = seq
	case "version_color":
		p.VersionColor = seq
	case "success_box":
		p.SuccessBox = seq
	case "info_box":
		p.InfoBox = seq
	case "table_status_add":
		p.TableStatusAdd = seq
	case "table_status_remove":
		p.TableStatusRemove = seq
	case "table_status_update":
		p.TableStatusUpdate = seq
	case "table_status_drift":
		p.TableStatusDrift = seq
	case "table_status_cleanup":
		p.TableStatusCleanup = seq
	case "diff_badge_add":
		p.DiffBadgeAdd = seq
	case "diff_badge_remove":
		p.DiffBadgeRemove = seq
	case "diff_badge_update":
		p.DiffBadgeUpdate = seq
	case "diff_provider":
		p.DiffProvider = seq
	case "diff_path":
		p.DiffPath = seq
	case "nochanges_border":
		p.NoChangesBorder = seq
	case "spinner_color":
		p.SpinnerColor = seq
	case "bg_black":
		p.BgBlack = seq
	case "white":
		p.White = seq
	default:
		// Unknown role — no-op
	}
}

// ApplyToDefaults merges this palette into the global DefaultColors and
// refreshes exported styles. This is a convenience for tests and runtime
// configuration where you want to update the palette in-place.
func (p *ColorPalette) ApplyToDefaults() {
	// merge fields selectively so zero-values don't overwrite existing
	// defaults when omitted
	q := DefaultPalette()
	// copy all fields from p that are non-empty
	if p.Success != "" {
		q.Success = p.Success
	}
	if p.Error != "" {
		q.Error = p.Error
	}
	if p.Warning != "" {
		q.Warning = p.Warning
	}
	if p.Info != "" {
		q.Info = p.Info
	}
	if p.Dim != "" {
		q.Dim = p.Dim
	}
	if p.RowSuccess != "" {
		q.RowSuccess = p.RowSuccess
	}
	if p.RowError != "" {
		q.RowError = p.RowError
	}
	if p.RowWarning != "" {
		q.RowWarning = p.RowWarning
	}
	if p.Header != "" {
		q.Header = p.Header
	}
	if p.TableHeader != "" {
		q.TableHeader = p.TableHeader
	}
	if p.HeaderKindAdd != "" {
		q.HeaderKindAdd = p.HeaderKindAdd
	}
	if p.HeaderKindRemove != "" {
		q.HeaderKindRemove = p.HeaderKindRemove
	}
	if p.HeaderKindUpdate != "" {
		q.HeaderKindUpdate = p.HeaderKindUpdate
	}
	if p.SuccessBox != "" {
		q.SuccessBox = p.SuccessBox
	}
	if p.InfoBox != "" {
		q.InfoBox = p.InfoBox
	}
	if len(p.NoChangesRainbow) > 0 {
		q.NoChangesRainbow = p.NoChangesRainbow
	}
	// replace global DefaultColors and refresh styles
	ApplyPalette(q)
}
