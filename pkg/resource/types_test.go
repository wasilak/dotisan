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

func TestManagedDirectory_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dir     ManagedDirectory
		wantErr bool
	}{
		{
			name: "valid with recursive and clean",
			dir: ManagedDirectory{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "ManagedDirectory",
					Metadata:   Metadata{Name: "skills"},
				},
				Spec: ManagedDirectorySpec{
					Source:      "skills/",
					Destination: "~/.claude/skills",
					Recursive:   true,
					Clean:       true,
				},
			},
			wantErr: false,
		},
		{
			name: "valid minimal",
			dir: ManagedDirectory{
				BaseResource: BaseResource{
					APIVersion: "github.com/wasilak/dotisan/v1",
					Kind:       "ManagedDirectory",
					Metadata:   Metadata{Name: "minimal"},
				},
				Spec: ManagedDirectorySpec{
					Source:      "configs/",
					Destination: "~/.configs",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.dir.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
