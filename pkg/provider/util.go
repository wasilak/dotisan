package provider

import (
	"fmt"
	"os/exec"
)

// CheckExecutable checks if an executable is available in the system PATH.
// Returns true if available, false with a descriptive message if not.
// This is the standard implementation for the Available() method.
func CheckExecutable(name string) (bool, string) {
	path, err := exec.LookPath(name)
	if err != nil {
		return false, fmt.Sprintf("%s not found in PATH: %v", name, err)
	}
	return true, fmt.Sprintf("%s found at %s", name, path)
}

// CheckExecutables checks multiple executables and returns a summary.
// Returns true only if all executables are available.
// Useful for providers that require multiple tools.
func CheckExecutables(names ...string) (bool, string) {
	var missing []string
	var found []string

	for _, name := range names {
		available, message := CheckExecutable(name)
		if available {
			found = append(found, message)
		} else {
			missing = append(missing, message)
		}
	}

	if len(missing) > 0 {
		return false, fmt.Sprintf("Missing %d of %d executables:\n%s",
			len(missing), len(names), joinMessages(missing))
	}

	return true, fmt.Sprintf("All %d executables found:\n%s",
		len(names), joinMessages(found))
}

// joinMessages joins messages with newlines.
func joinMessages(messages []string) string {
	result := ""
	for i, msg := range messages {
		if i > 0 {
			result += "\n"
		}
		result += "  - " + msg
	}
	return result
}
