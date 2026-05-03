package resource

import (
	"testing"
)

func TestUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		check   func(t *testing.T, r Resource)
	}{
		{
			name: "valid HomeBrewPackages",
			yaml: `
apiVersion: github.com/wasilak/dotisan/v1
kind: HomeBrewPackages
metadata:
  name: core-tools
spec:
  formulae:
    - name: ripgrep
    - name: fd
`,
			wantErr: false,
			check: func(t *testing.T, r Resource) {
				bp, ok := r.(*HomeBrewPackages)
				if !ok {
					t.Errorf("expected *HomeBrewPackages, got %T", r)
					return
				}
				if bp.GetMetadata().Name != "core-tools" {
					t.Errorf("Name = %q, want %q", bp.GetMetadata().Name, "core-tools")
				}
				if len(bp.Spec.Formulae) != 2 {
					t.Errorf("len(Formulae) = %d, want 2", len(bp.Spec.Formulae))
				}
			},
		},
		{
			name: "valid ManagedFile",
			yaml: `
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: zshrc
  namespace: config
spec:
  source: templates/zshrc.tmpl
  destination: ~/.zshrc
  template: true
  mode: "0644"
`,
			wantErr: false,
			check: func(t *testing.T, r Resource) {
				mf, ok := r.(*ManagedFile)
				if !ok {
					t.Errorf("expected *ManagedFile, got %T", r)
					return
				}
				if mf.GetMetadata().Name != "zshrc" {
					t.Errorf("Name = %q, want %q", mf.GetMetadata().Name, "zshrc")
				}
				if mf.GetMetadata().GetNamespace() != "config" {
					t.Errorf("Namespace = %q, want %q", mf.GetMetadata().GetNamespace(), "config")
				}
				if mf.Spec.Source != "templates/zshrc.tmpl" {
					t.Errorf("Source = %q, want %q", mf.Spec.Source, "templates/zshrc.tmpl")
				}
				if !mf.Spec.Template {
					t.Error("Template should be true")
				}
			},
		},
		{
			name: "invalid ManagedDirectory (removed)",
			yaml: `
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedDirectory
metadata:
  name: skills
spec:
  source: skills/
  destination: ~/.claude/skills
  recursive: true
  clean: true
`,
			wantErr: true,
			check:   nil,
		},
		{
			name: "valid NpmPackages",
			yaml: `
apiVersion: github.com/wasilak/dotisan/v1
kind: NpmPackages
metadata:
  name: globals
spec:
  packages:
    - name: typescript
      version: "5.4.0"
    - name: prettier
`,
			wantErr: false,
			check: func(t *testing.T, r Resource) {
				np, ok := r.(*NpmPackages)
				if !ok {
					t.Errorf("expected *NpmPackages, got %T", r)
					return
				}
				if len(np.Spec.Packages) != 2 {
					t.Errorf("len(Packages) = %d, want 2", len(np.Spec.Packages))
				}
				if np.Spec.Packages[0].Version != "5.4.0" {
					t.Errorf("Packages[0].Version = %q, want %q", np.Spec.Packages[0].Version, "5.4.0")
				}
			},
		},
		{
			name: "valid GoPackages",
			yaml: `
apiVersion: github.com/wasilak/dotisan/v1
kind: GoPackages
metadata:
  name: tools
spec:
  packages:
    - module: golang.org/x/tools/gopls
      version: latest
`,
			wantErr: false,
			check: func(t *testing.T, r Resource) {
				gp, ok := r.(*GoPackages)
				if !ok {
					t.Errorf("expected *GoPackages, got %T", r)
					return
				}
				if gp.Spec.Packages[0].Module != "golang.org/x/tools/gopls" {
					t.Errorf("Module = %q, want %q", gp.Spec.Packages[0].Module, "golang.org/x/tools/gopls")
				}
			},
		},
		{
			name: "valid CargoPackages",
			yaml: `
apiVersion: github.com/wasilak/dotisan/v1
kind: CargoPackages
metadata:
  name: rust-tools
spec:
  packages:
    - name: ripgrep
    - name: tokei
`,
			wantErr: false,
			check: func(t *testing.T, r Resource) {
				cp, ok := r.(*CargoPackages)
				if !ok {
					t.Errorf("expected *CargoPackages, got %T", r)
					return
				}
				if len(cp.Spec.Packages) != 2 {
					t.Errorf("len(Packages) = %d, want 2", len(cp.Spec.Packages))
				}
			},
		},
		{
			name: "valid HomeBrewPackages",
			yaml: `
apiVersion: github.com/wasilak/dotisan/v1
kind: HomeBrewPackages
metadata:
  name: core-tools
spec:
  formulae:
    - name: ripgrep
    - name: fd
`,
			wantErr: false,
			check: func(t *testing.T, r Resource) {
				hp, ok := r.(*HomeBrewPackages)
				if !ok {
					t.Errorf("expected *HomeBrewPackages, got %T", r)
					return
				}
				if hp.GetMetadata().Name != "core-tools" {
					t.Errorf("Name = %q, want %q", hp.GetMetadata().Name, "core-tools")
				}
				if len(hp.Spec.Formulae) != 2 {
					t.Errorf("len(Formulae) = %d, want 2", len(hp.Spec.Formulae))
				}
			},
		},
		{
			name: "valid HomeBrewCasks",
			yaml: `
apiVersion: github.com/wasilak/dotisan/v1
kind: HomeBrewCasks
metadata:
  name: apps
spec:
  casks:
    - name: wezterm
    - name: raycast
`,
			wantErr: false,
			check: func(t *testing.T, r Resource) {
				c, ok := r.(*HomeBrewCasks)
				if !ok {
					t.Errorf("expected *HomeBrewCasks, got %T", r)
					return
				}
				if c.GetMetadata().Name != "apps" {
					t.Errorf("Name = %q, want %q", c.GetMetadata().Name, "apps")
				}
				if len(c.Spec.Casks) != 2 {
					t.Errorf("len(Casks) = %d, want 2", len(c.Spec.Casks))
				}
			},
		},
		{
			name: "valid HomeBrewTaps",
			yaml: `
apiVersion: github.com/wasilak/dotisan/v1
kind: HomeBrewTaps
metadata:
  name: taps
spec:
  taps:
    - name: homebrew/cask-fonts
`,
			wantErr: false,
			check: func(t *testing.T, r Resource) {
				tp, ok := r.(*HomeBrewTaps)
				if !ok {
					t.Errorf("expected *HomeBrewTaps, got %T", r)
					return
				}
				if tp.GetMetadata().Name != "taps" {
					t.Errorf("Name = %q, want %q", tp.GetMetadata().Name, "taps")
				}
				if len(tp.Spec.Taps) != 1 {
					t.Errorf("len(Taps) = %d, want 1", len(tp.Spec.Taps))
				}
			},
		},
		{
			name: "unsupported apiVersion",
			yaml: `
apiVersion: dotisan/v2
kind: HomeBrewPackages
metadata:
  name: test
spec: {}
`,
			wantErr: true,
			check:   nil,
		},
		{
			name: "unknown kind",
			yaml: `
apiVersion: github.com/wasilak/dotisan/v1
kind: UnknownResource
metadata:
  name: test
spec: {}
`,
			wantErr: true,
			check:   nil,
		},
		{
			name: "invalid YAML",
			yaml: `
invalid: yaml: [{
`,
			wantErr: true,
			check:   nil,
		},
		{
			name: "missing apiVersion",
			yaml: `
kind: HomeBrewPackages
metadata:
  name: test
spec: {}
`,
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, err := UnmarshalYAML([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return // expected error
			}
			if tt.check != nil {
				tt.check(t, resource)
			}
		})
	}
}

func TestValidResourceKinds(t *testing.T) {
	kinds := ValidResourceKinds()
	expected := []string{
		"HomeBrewPackages",
		"HomeBrewCasks",
		"HomeBrewTaps",
		"NpmPackages",
		"GoPackages",
		"CargoPackages",
		"ManagedFile",
	}

	if len(kinds) != len(expected) {
		t.Errorf("ValidResourceKinds() returned %d kinds, expected %d", len(kinds), len(expected))
	}

	for _, exp := range expected {
		found := false
		for _, kind := range kinds {
			if kind == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected kind %q not found in ValidResourceKinds()", exp)
		}
	}
}
