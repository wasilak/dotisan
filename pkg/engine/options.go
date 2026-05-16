package engine

import (
	"fmt"
	"regexp"
	"strings"
)

// PlanOptions contains options for the Plan operation.
type PlanOptions struct {
	Targets  []string
	ShowDiff bool // Show file/package diffs (unified, syntax-highlighted)
}

// TargetMatch represents a parsed target expression. A field may be a
// /pattern/ regex; in that case re is non-nil and the literal fields are empty.
type TargetMatch struct {
	Kind  string
	Group string
	Item  string
	// re is set when the raw target was wrapped in /…/ — it is matched
	// case-insensitively against the full identifier "Kind/Group[Item]".
	re *regexp.Regexp
}

// IsRegex reports whether this target uses regex matching.
func (t TargetMatch) IsRegex() bool { return t.re != nil }

// ParseTargets converts raw target strings into TargetMatch structs.
// Supports:
//   - /pattern/ — case-insensitive regex matched against "Kind/Group[Item]"
//   - Kind, Kind/Group, Kind/Group[Item] — exact (case-insensitive) literal match
func ParseTargets(targets []string) ([]TargetMatch, error) {
	var out []TargetMatch
	for _, raw := range targets {
		raw = strings.TrimSpace(raw)

		// /pattern/ syntax: compile as case-insensitive regex.
		if len(raw) >= 2 && raw[0] == '/' && raw[len(raw)-1] == '/' {
			pattern := raw[1 : len(raw)-1]
			re, err := regexp.Compile("(?i)" + pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid regex target %q: %w", raw, err)
			}
			out = append(out, TargetMatch{re: re})
			continue
		}

		t := strings.ReplaceAll(raw, "\\", "/")
		tm := TargetMatch{}
		// Handle bracketed item notation: Kind/Group[Item]
		bracketStart := strings.LastIndex(t, "[")
		bracketEnd := strings.LastIndex(t, "]")
		if bracketStart != -1 && bracketEnd != -1 && bracketEnd == len(t)-1 && bracketStart < bracketEnd {
			prefix := t[:bracketStart]
			tm.Item = t[bracketStart+1 : bracketEnd]
			parts := strings.SplitN(prefix, "/", 2)
			if len(parts) >= 1 {
				tm.Kind = parts[0]
			}
			if len(parts) == 2 {
				tm.Group = parts[1]
			}
		} else {
			// No brackets - support legacy Kind/Group/Item or shorter
			parts := strings.SplitN(t, "/", 3)
			if len(parts) >= 1 {
				tm.Kind = parts[0]
			}
			if len(parts) >= 2 {
				tm.Group = parts[1]
			}
			if len(parts) == 3 {
				tm.Item = parts[2]
			}
		}
		out = append(out, tm)
	}
	return out, nil
}

// Matches returns true if the given kind/group/item satisfies this target.
// For regex targets the check is against the full identifier "Kind/Group[Item]"
// (or "Kind/Group" when item is empty).
func (t TargetMatch) Matches(kind, group, item string) bool {
	if t.re != nil {
		id := kind + "/" + group
		if item != "" {
			id += "[" + item + "]"
		}
		return t.re.MatchString(id)
	}
	if t.Kind != "" && !strings.EqualFold(t.Kind, kind) {
		return false
	}
	if t.Group != "" && !strings.EqualFold(t.Group, group) {
		return false
	}
	if t.Item != "" && !strings.EqualFold(t.Item, item) {
		return false
	}
	return true
}

// Note: no alias normalization is performed. Users must specify canonical
// Kind names (case-insensitive). Aliases like "npm" are not supported.
