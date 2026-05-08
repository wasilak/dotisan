package resource

import (
	"testing"
)

func TestHomeBrewPackages_Validate(t *testing.T) {
	tests := []struct {
		name    string
		pkg     HomeBrewPackages
		wantErr bool
	}{
		{
			name: "valid with formulae",
			pkg: HomeBrewPackages{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/nim/v1",
					Kind:       "HomeBrewPackages",
					Metadata:   Metadata{Name: "core-tools"},
				},
				Spec: HomeBrewPackagesSpec{
					Formulae: []Package{{Name: "ripgrep"}, {Name: "fd"}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid with taps only",
			pkg: HomeBrewPackages{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/nim/v1",
					Kind:       "HomeBrewPackages",
					Metadata:   Metadata{Name: "fonts"},
				},
				Spec: HomeBrewPackagesSpec{
					// no formulae, but taps may live in RawSpec for HomeBrewTaps
					Formulae: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "missing metadata.name",
			pkg: HomeBrewPackages{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/nim/v1",
					Kind:       "HomeBrewPackages",
					Metadata:   Metadata{},
				},
				Spec: HomeBrewPackagesSpec{},
			},
			wantErr: true,
		},
		{
			name: "empty spec allowed",
			pkg: HomeBrewPackages{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/nim/v1",
					Kind:       "HomeBrewPackages",
					Metadata:   Metadata{Name: "empty"},
				},
				Spec: HomeBrewPackagesSpec{},
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
					APIVersion: "github.com/wasilak/nim/v1",
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
					APIVersion: "github.com/wasilak/nim/v1",
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
					APIVersion: "github.com/wasilak/nim/v1",
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
					APIVersion: "github.com/wasilak/nim/v1",
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
					APIVersion: "github.com/wasilak/nim/v1",
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
					APIVersion: "github.com/wasilak/nim/v1",
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
					APIVersion: "github.com/wasilak/nim/v1",
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
					APIVersion: "github.com/wasilak/nim/v1",
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
					APIVersion: "github.com/wasilak/nim/v1",
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
					APIVersion: "github.com/wasilak/nim/v1",
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
					APIVersion: "github.com/wasilak/nim/v1",
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
			APIVersion: "github.com/wasilak/nim/v1",
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
		if item.FileExtra == nil {
			t.Fatalf("item[%d].FileExtra is nil", i)
		}
		fe := item.FileExtra
		if fe.Source != "(inline)" {
			t.Errorf("item[%d].FileExtra.Source = %q, want \"(inline)\"", i, fe.Source)
		}
		if fe.Inline == "" {
			t.Errorf("item[%d].FileExtra.Inline is empty", i)
		}
		if fe.Destination == "" {
			t.Errorf("item[%d].FileExtra.Destination is empty", i)
		}
		if fe.Mode != "0644" {
			t.Errorf("item[%d].FileExtra.Mode = %q, want \"0644\"", i, fe.Mode)
		}
	}

	if group.Items[0].FileExtra.Inline != "skill: bash\nindex: 0" {
		t.Errorf("item[0].FileExtra.Inline = %q", group.Items[0].FileExtra.Inline)
	}
	if group.Items[1].FileExtra.Destination != "/tmp/skills/python.md" {
		t.Errorf("item[1].FileExtra.Destination = %q", group.Items[1].FileExtra.Destination)
	}
}

func TestValidateDependsOnAddresses(t *testing.T) {
	base := BaseResource{
		APIVersion: "github.com/wasilak/nim/v1",
		Kind:       "HomeBrewPackages",
		Metadata:   Metadata{Name: "test"},
	}
	spec := HomeBrewPackagesSpec{}

	cases := []struct {
		name      string
		dependsOn []string
		wantErr   bool
	}{
		{"nil dependsOn", nil, false},
		{"empty dependsOn", []string{}, false},
		{"valid Kind/Group", []string{"HomeBrewPackages/core-tools"}, false},
		{"valid Kind/Group[Item]", []string{"HomeBrewPackages/core-tools[ripgrep]"}, false},
		{"valid namespace/Kind/Group", []string{"default/GoPackages/tools"}, false},
		{"multiple valid", []string{"HomeBrewPackages/tools", "GoPackages/dev"}, false},
		{"empty string is invalid", []string{""}, true},
		{"missing kind (bare slash)", []string{"/group"}, true},
		{"unclosed bracket", []string{"Kind/Group[unclosed"}, true},
		{"empty kind with bracket", []string{"[item]"}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := base
			b.Metadata.DependsOn = c.dependsOn
			r := HomeBrewPackages{BaseResource: b, Spec: spec}
			err := r.Validate()
			if (err != nil) != c.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}

func TestValidateDependsOn_AllResourceTypes(t *testing.T) {
	invalid := []string{"[bad]"}
	meta := Metadata{Name: "test", DependsOn: invalid}
	br := BaseResource{APIVersion: "github.com/wasilak/nim/v1", Kind: "K", Metadata: meta}

	resources := []Resource{
		HomeBrewPackages{BaseResource: br, Spec: HomeBrewPackagesSpec{}},
		HomeBrewCasks{BaseResource: br, Spec: HomeBrewCasksSpec{}},
		HomeBrewTaps{BaseResource: br, Spec: HomeBrewTapsSpec{}},
		NpmPackages{BaseResource: br, Spec: NpmPackagesSpec{Packages: []Package{{Name: "x"}}}},
		GoPackages{BaseResource: br, Spec: GoPackagesSpec{Packages: []GoPackage{{Module: "x"}}}},
		CargoPackages{BaseResource: br, Spec: CargoPackagesSpec{Packages: []Package{{Name: "x"}}}},
		ManagedFile{BaseResource: br, Spec: ManagedFileSpec{Source: "s", Destination: "/d"}},
	}

	for _, r := range resources {
		if err := r.Validate(); err == nil {
			t.Errorf("%T.Validate() should reject invalid dependsOn, got nil", r)
		}
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
