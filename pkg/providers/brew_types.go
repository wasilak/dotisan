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
	Token     string              `json:"token"`
	Name      string              `json:"name"`
	Installed []brewCaskInstalled `json:"installed"`
}

type brewCaskInstalled struct {
	Version string `json:"version"`
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
	if len(c.Installed) > 0 {
		return c.Installed[0].Version
	}
	return ""
}

// DisplayName returns the best identifier for the cask: prefer Name then Token.
func (c brewCaskInfo) DisplayName() string {
	if c.Name != "" {
		return c.Name
	}
	return c.Token
}
