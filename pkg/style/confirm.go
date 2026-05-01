package style

import (
	"os"
	"strings"

	"golang.org/x/term"
)

// ReadSingleKey reads a single keypress from stdin without echoing it to the
// terminal and returns the lowercased string representation. Caller is
// responsible for interpreting the value. This is intentionally small and
// focused for use in confirmation prompts.
func ReadSingleKey() (string, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer term.Restore(fd, oldState)

	b := make([]byte, 1)
	if _, err := os.Stdin.Read(b); err != nil {
		return "", err
	}
	return strings.ToLower(string(b)), nil
}
