package audit

import (
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// parseTemplateString attempts to parse a Go template string (with Sprig funcs)
// and returns error if any. It registers a small set of generator helper stubs
// (like readFile) so that templates referencing them can be parsed for syntax
// without requiring full runtime helpers.
func parseTemplateString(s string) (bool, error) {
	funcMap := sprig.TxtFuncMap()
	// Add safe stub for readFile commonly used in generator templates.
	funcMap["readFile"] = func(path string) (string, error) { return "", nil }
	_, err := template.New("t").Funcs(funcMap).Parse(s)
	if err != nil {
		return false, err
	}
	return true, nil
}
