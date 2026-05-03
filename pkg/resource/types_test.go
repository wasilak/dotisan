package resource

import (
	"testing"
)

func TestBrewPackages_Validate(t *testing.T) {
	tests := []struct {
		name    string
		pkg     BrewPackages
		wantErr bool
	}{
		{
			name: "valid with formulae",
			pkg: BrewPackages{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "BrewPackages",
					Metadata:   Metadata{Name: "core-tools"},
				},
				Spec: BrewPackagesSpec{
					Formulae: []Package{{Name: "ripgrep"}, {Name: "fd"}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid with taps only",
			pkg: BrewPackages{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "BrewPackages",
					Metadata:   Metadata{Name: "fonts"},
				},
				Spec: BrewPackagesSpec{
					Taps: []Tap{{Name: "homebrew/cask-fonts"}},
				},
			},
			wantErr: false,
		},
		{
			name: "missing metadata.name",
			pkg: BrewPackages{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "BrewPackages",
					Metadata:   Metadata{},
				},
				Spec: BrewPackagesSpec{},
			},
			wantErr: true,
		},
		{
			name: "empty spec allowed",
			pkg: BrewPackages{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "BrewPackages",
					Metadata:   Metadata{Name: "empty"},
				},
				Spec: BrewPackagesSpec{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pkg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNpmPackages_Validate(t *testing.T) {
	tests := []struct {
		name    string
		pkg     NpmPackages
		wantErr bool
	}{
		{
			name: "valid with packages",
			pkg: NpmPackages{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "NpmPackages",
					Metadata:   Metadata{Name: "globals"},
				},
				Spec: NpmPackagesSpec{
					Packages: []Package{
						{Name: "typescript", Version: "5.4.0"},
						{Name: "prettier"},
					},
				},
			},
			wantErr: false,
		},
		// Note: empty packages slice is allowed (may be populated at runtime)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pkg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGoPackages_Validate(t *testing.T) {
	tests := []struct {
		name    string
		pkg     GoPackages
		wantErr bool
	}{
		{
			name: "valid with modules",
			pkg: GoPackages{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "GoPackages",
					Metadata:   Metadata{Name: "tools"},
				},
				Spec: GoPackagesSpec{
					Packages: []GoPackage{
						{Module: "golang.org/x/tools/gopls", Version: "latest"},
						{Module: "github.com/air-verse/air"},
					},
				},
			},
			wantErr: false,
		},
		// Note: empty packages slice is allowed (may be populated at runtime)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pkg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManagedFile_Validate(t *testing.T) {
	tests := []struct {
		name    string
		file    ManagedFile
		wantErr bool
	}{
		{
			name: "valid with mode",
			file: ManagedFile{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "ManagedFile",
					Metadata:   Metadata{Name: "zshrc"},
				},
				Spec: ManagedFileSpec{
					Source:      "templates/zshrc.tmpl",
					Destination: "~/.zshrc",
					Template:    true,
					Mode:        "0644",
				},
			},
			wantErr: false,
		},
		{
			name: "valid without mode",
			file: ManagedFile{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "ManagedFile",
					Metadata:   Metadata{Name: "config"},
				},
				Spec: ManagedFileSpec{
					Source:      "static/config.yaml",
					Destination: "~/.config/app/config.yaml",
					Template:    false,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			file: ManagedFile{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "ManagedFile",
					Metadata:   Metadata{Name: "bad-mode"},
				},
				Spec: ManagedFileSpec{
					Source:      "test.txt",
					Destination: "~/test.txt",
					Mode:        "abc", // invalid
				},
			},
			wantErr: true,
		},
		{
			name: "missing source",
			file: ManagedFile{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "ManagedFile",
					Metadata:   Metadata{Name: "no-source"},
				},
				Spec: ManagedFileSpec{
					Destination: "~/test.txt",
				},
			},
			wantErr: true,
		},
		{
			name: "valid generator",
			file: ManagedFile{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "ManagedFile",
					Metadata:   Metadata{Name: "gen"},
				},
				Spec: ManagedFileSpec{
					Generator: &GeneratorSpec{
						SourceKey:          "skills",
						Template:           "content: {{ .item }}",
						DestinationPattern: "~/.claude/skills/{{ .item }}.md",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "generator with mode",
			file: ManagedFile{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "ManagedFile",
					Metadata:   Metadata{Name: "gen-mode"},
				},
				Spec: ManagedFileSpec{
					Generator: &GeneratorSpec{
						SourceKey:          "agents",
						Template:           "{{ .item.name }}",
						DestinationPattern: "~/.claude/agents/{{ .item.name }}.md",
						Mode:               "0644",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "generator conflicts with source",
			file: ManagedFile{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "ManagedFile",
					Metadata:   Metadata{Name: "gen-conflict-source"},
				},
				Spec: ManagedFileSpec{
					Source: "inline content",
					Generator: &GeneratorSpec{
						SourceKey:          "skills",
						Template:           "{{ .item }}",
						DestinationPattern: "~/out/{{ .item }}.md",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "generator conflicts with sourceFile",
			file: ManagedFile{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "ManagedFile",
					Metadata:   Metadata{Name: "gen-conflict-sourcefile"},
				},
				Spec: ManagedFileSpec{
					SourceFile: "shell/zshrc.sh",
					Generator: &GeneratorSpec{
						SourceKey:          "skills",
						Template:           "{{ .item }}",
						DestinationPattern: "~/out/{{ .item }}.md",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "generator conflicts with files",
			file: ManagedFile{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "ManagedFile",
					Metadata:   Metadata{Name: "gen-conflict-files"},
				},
				Spec: ManagedFileSpec{
					Files: []FileSpec{{Source: "x", Destination: "~/x.txt"}},
					Generator: &GeneratorSpec{
						SourceKey:          "skills",
						Template:           "{{ .item }}",
						DestinationPattern: "~/out/{{ .item }}.md",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.file.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Note: ManagedDirectory has been removed. No validation tests remain for it.

// TestManagedFile_GeneratorExpanded_ToGroup verifies that a generator-expanded ManagedFile
// (Generator cleared, Files populated) passes through ToGroup() and GetFiles() correctly,
// producing ResourceItems with inline content — exactly as the FileProvider expects.
func TestManagedFile_GeneratorExpanded_ToGroup(t *testing.T) {
	mf := &ManagedFile{
		BaseResource: BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedFile",
			Metadata:   Metadata{Name: "gen-skills", Namespace: "files"},
		},
		Spec: ManagedFileSpec{
			// Simulates state after expandGenerator: Generator is nil, Files is populated.
			Files: []FileSpec{
				{Source: "skill: bash\nindex: 0", Destination: "/tmp/skills/bash.md", Mode: "0644"},
				{Source: "skill: python\nindex: 1", Destination: "/tmp/skills/python.md", Mode: "0644"},
			},
		},
	}

	// GetFiles should return the generated Files slice directly.
	files := mf.GetFiles()
	if len(files) != 2 {
		t.Fatalf("GetFiles() returned %d files, want 2", len(files))
	}
	if files[0].Source != "skill: bash\nindex: 0" {
		t.Errorf("files[0].Source = %q", files[0].Source)
	}
	if files[1].Destination != "/tmp/skills/python.md" {
		t.Errorf("files[1].Destination = %q", files[1].Destination)
	}

	// ToGroup should produce ResourceItems with inline content (no external source).
	group := mf.ToGroup()

	if group.Kind != "ManagedFile" {
		t.Errorf("group.Kind = %q", group.Kind)
	}
	if group.Name != "gen-skills" {
		t.Errorf("group.Name = %q", group.Name)
	}
	if len(group.Items) != 2 {
		t.Fatalf("group.Items len = %d, want 2", len(group.Items))
	}

	// Both items should have inline content, no external source.
	for i, item := range group.Items {
		source, _ := item.Extra["source"].(string)
		inline, _ := item.Extra["inline"].(string)
		dest, _ := item.Extra["destination"].(string)
		mode, _ := item.Extra["mode"].(string)

		if source != "(inline)" {
			t.Errorf("item[%d].Extra[source] = %q, want \"(inline)\"", i, source)
		}
		if inline == "" {
			t.Errorf("item[%d].Extra[inline] is empty", i)
		}
		if dest == "" {
			t.Errorf("item[%d].Extra[destination] is empty", i)
		}
		if mode != "0644" {
			t.Errorf("item[%d].Extra[mode] = %q, want \"0644\"", i, mode)
		}
	}

	if inline, _ := group.Items[0].Extra["inline"].(string); inline != "skill: bash\nindex: 0" {
		t.Errorf("item[0].Extra[inline] = %q", inline)
	}
	if dest, _ := group.Items[1].Extra["destination"].(string); dest != "/tmp/skills/python.md" {
		t.Errorf("item[1].Extra[destination] = %q", dest)
	}
}

// TestManagedFile_GeneratorExpanded_GetFiles_NoFallback verifies that GetFiles does NOT
// fall back to the single-file fields when Files is populated (generator case).
func TestManagedFile_GeneratorExpanded_GetFiles_NoFallback(t *testing.T) {
	mf := &ManagedFile{
		Spec: ManagedFileSpec{
			Source:      "this should be ignored",
			Destination: "/ignored",
			Files: []FileSpec{
				{Source: "generated", Destination: "/tmp/gen.md"},
			},
		},
	}

	files := mf.GetFiles()
	if len(files) != 1 {
		t.Fatalf("GetFiles() returned %d files, want 1", len(files))
	}
	if files[0].Source != "generated" {
		t.Errorf("GetFiles() returned single-file fallback instead of Files slice")
	}
}
