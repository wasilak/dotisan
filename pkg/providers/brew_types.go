package providers

// Types for parsing `brew info --json=v2` output.

// brewInfoOutput is the top-level structure returned by `brew info --json=v2`.
type brewInfoOutput struct {
	Formulae []brewFormulaInfo `json:"formulae"`
	Casks    []brewCaskInfo    `json:"casks"`
}

type brewFormulaInfo struct {
	Name      string                 `json:"name"`
	Versions  brewFormulaVersions    `json:"versions"`
	Installed []brewFormulaInstalled `json:"installed"`
}

type brewFormulaVersions struct {
	Stable string `json:"stable"`
}

type brewFormulaInstalled struct {
	Version string `json:"version"`
}

type brewCaskInfo struct {
	Token     string   `json:"token"`
	Name      []string `json:"name"`
	Installed *string  `json:"installed"` // version string or null
}

// InstalledVersion returns the installed version for a formula, or the stable
// version if no installed version is present.
func (f brewFormulaInfo) InstalledVersion() string {
	if len(f.Installed) > 0 {
		return f.Installed[0].Version
	}
	return f.Versions.Stable
}

// InstalledVersion returns the installed version for a cask if present.
func (c brewCaskInfo) InstalledVersion() string {
	if c.Installed != nil {
		return *c.Installed
	}
	return ""
}

// DisplayName returns a human-readable name for the cask: first element of Name
// if present, otherwise Token.
func (c brewCaskInfo) DisplayName() string {
	if len(c.Name) > 0 && c.Name[0] != "" {
		return c.Name[0]
	}
	return c.Token
}
