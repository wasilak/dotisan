package cmd

import "testing"

func TestParseResourceRef(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantKey string
		wantHas bool
	}{
		{"no brackets", "zshrc", "zshrc", "", false},
		{"string key", "core-tools[ripgrep]", "core-tools", "ripgrep", true},
		{"numeric key", "core-tools[0]", "core-tools", "0", true},
		{"path key", "dotfiles[/path/to/file]", "dotfiles", "/path/to/file", true},
		{"hyphen name", "my-config[setting]", "my-config", "setting", true},
		{"underscore name", "my_config[key]", "my_config", "key", true},
		{"mixed", "packages[golang/go]", "packages", "golang/go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotKey, gotHas := parseResourceRef(tt.input)
			if got != tt.want {
				t.Errorf("parseResourceRef(%q) name = %q, want %q", tt.input, got, tt.want)
			}
			if gotKey != tt.wantKey {
				t.Errorf("parseResourceRef(%q) itemKey = %q, want %q", tt.input, gotKey, tt.wantKey)
			}
			if gotHas != tt.wantHas {
				t.Errorf("parseResourceRef(%q) hasItemKey = %v, want %v", tt.input, gotHas, tt.wantHas)
			}
		})
	}
}
