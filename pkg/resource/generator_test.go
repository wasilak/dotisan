package resource

import (
	"os"
	"strings"
	"testing"

	"github.com/wasilak/dotisan/pkg/config"
)

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/foo/bar", home + "/foo/bar"},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"no-tilde", "no-tilde"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := expandTilde(tt.input)
			if got != tt.want {
				t.Errorf("expandTilde(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveSourceKey(t *testing.T) {
	values := map[string]any{
		"skills": []any{"bash", "python", "go"},
		"agents": map[string]any{
			"list": []any{
				map[string]any{"name": "coder"},
				map[string]any{"name": "reviewer"},
			},
		},
		"meta": map[string]any{
			"nested": map[string]any{
				"items": []any{"a", "b"},
			},
		},
		"notAList": "just a string",
		"number":   42,
	}

	tests := []struct {
		name      string
		key       string
		wantLen   int
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "top-level list",
			key:     "skills",
			wantLen: 3,
		},
		{
			name:    "nested list one level",
			key:     "agents.list",
			wantLen: 2,
		},
		{
			name:    "nested list two levels",
			key:     "meta.nested.items",
			wantLen: 2,
		},
		{
			name:      "missing top-level key",
			key:       "missing",
			wantErr:   true,
			errSubstr: `key "missing" not found`,
		},
		{
			name:      "missing nested key",
			key:       "agents.missing",
			wantErr:   true,
			errSubstr: `key "missing" not found`,
		},
		{
			name:      "intermediate node is not a map",
			key:       "notAList.child",
			wantErr:   true,
			errSubstr: "is not a map",
		},
		{
			name:      "value is not a list — string",
			key:       "notAList",
			wantErr:   true,
			errSubstr: "is not a list",
		},
		{
			name:      "value is not a list — number",
			key:       "number",
			wantErr:   true,
			errSubstr: "is not a list",
		},
		{
			name:      "empty key",
			key:       "",
			wantErr:   true,
			errSubstr: "must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveSourceKey(tt.key, values)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != tt.wantLen {
				t.Errorf("got %d items, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestExpandGenerator(t *testing.T) {
	ctx := &config.TemplateContext{
		Values: map[string]any{
			"skills": []any{"bash", "python"},
			"agents": []any{
				map[string]any{"name": "coder", "desc": "writes code"},
				map[string]any{"name": "reviewer", "desc": "reviews code"},
			},
		},
		Env: map[string]string{"HOME": "/home/user"},
		OS:  config.OSInfo{GOOS: "linux"},
	}

	baseMF := func(gen *GeneratorSpec) *ManagedFile {
		return &ManagedFile{
			BaseResource: BaseResource{
				APIVersion: "github.com/wasilak/dotisan/v1",
				Kind:       "ManagedFile",
				Metadata:   Metadata{Name: "test"},
			},
			Spec: ManagedFileSpec{Generator: gen},
		}
	}

	t.Run("nil generator is a no-op", func(t *testing.T) {
		mf := baseMF(nil)
		if err := expandGenerator(mf, ctx, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mf.Spec.Files != nil {
			t.Error("Files should remain nil")
		}
	})

	t.Run("scalar list produces correct FileSpecs", func(t *testing.T) {
		mf := baseMF(&GeneratorSpec{
			SourceKey:          "skills",
			Template:           "skill: {{ .Item }}",
			DestinationPattern: "/out/{{ .Item }}.md",
		})
		if err := expandGenerator(mf, ctx, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mf.Spec.Files) != 2 {
			t.Fatalf("want 2 files, got %d", len(mf.Spec.Files))
		}
		if mf.Spec.Files[0].Source != "skill: bash" {
			t.Errorf("file[0].Source = %q", mf.Spec.Files[0].Source)
		}
		if mf.Spec.Files[0].Destination != "/out/bash.md" {
			t.Errorf("file[0].Destination = %q", mf.Spec.Files[0].Destination)
		}
		if mf.Spec.Files[1].Source != "skill: python" {
			t.Errorf("file[1].Source = %q", mf.Spec.Files[1].Source)
		}
		if mf.Spec.Generator != nil {
			t.Error("Generator should be cleared after expansion")
		}
	})

	t.Run("map list produces correct FileSpecs", func(t *testing.T) {
		mf := baseMF(&GeneratorSpec{
			SourceKey:          "agents",
			Template:           "name: {{ .Item.name }}\ndesc: {{ .Item.desc }}",
			DestinationPattern: "/out/{{ .Item.name }}.md",
			Mode:               "0644",
		})
		if err := expandGenerator(mf, ctx, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mf.Spec.Files) != 2 {
			t.Fatalf("want 2 files, got %d", len(mf.Spec.Files))
		}
		if mf.Spec.Files[0].Mode != "0644" {
			t.Errorf("mode not propagated: %q", mf.Spec.Files[0].Mode)
		}
		if mf.Spec.Files[0].Destination != "/out/coder.md" {
			t.Errorf("destination = %q", mf.Spec.Files[0].Destination)
		}
	})

	t.Run("tilde in destination is expanded via GetFiles", func(t *testing.T) {
		home, _ := os.UserHomeDir()
		mf := baseMF(&GeneratorSpec{
			SourceKey:          "skills",
			Template:           "{{ .Item }}",
			DestinationPattern: "~/.claude/{{ .Item }}.md",
		})
		if err := expandGenerator(mf, ctx, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Raw FileSpec stores the un-expanded path; expansion happens in GetFiles().
		if mf.Spec.Files[0].Destination != "~/.claude/bash.md" {
			t.Errorf("raw destination = %q, want unexpanded ~", mf.Spec.Files[0].Destination)
		}
		want := home + "/.claude/bash.md"
		if got := mf.GetFiles()[0].Destination; got != want {
			t.Errorf("GetFiles()[0].Destination = %q, want %q", got, want)
		}
	})

	t.Run("index is available in template", func(t *testing.T) {
		mf := baseMF(&GeneratorSpec{
			SourceKey:          "skills",
			Template:           "{{ .Index }}:{{ .Item }}",
			DestinationPattern: "/out/{{ .Index }}.md",
		})
		if err := expandGenerator(mf, ctx, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mf.Spec.Files[0].Source != "0:bash" {
			t.Errorf("file[0].Source = %q", mf.Spec.Files[0].Source)
		}
		if mf.Spec.Files[1].Source != "1:python" {
			t.Errorf("file[1].Source = %q", mf.Spec.Files[1].Source)
		}
	})

	t.Run("values and env available in template", func(t *testing.T) {
		mf := baseMF(&GeneratorSpec{
			SourceKey:          "skills",
			Template:           "home={{ .Env.HOME }} os={{ .OS.GOOS }}",
			DestinationPattern: "/out/{{ .Item }}.md",
		})
		if err := expandGenerator(mf, ctx, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mf.Spec.Files[0].Source != "home=/home/user os=linux" {
			t.Errorf("file[0].Source = %q", mf.Spec.Files[0].Source)
		}
	})

	t.Run("missing sourceKey returns error", func(t *testing.T) {
		mf := baseMF(&GeneratorSpec{
			SourceKey:          "missing",
			Template:           "{{ .Item }}",
			DestinationPattern: "/out/{{ .Item }}.md",
		})
		err := expandGenerator(mf, ctx, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "missing") {
			t.Errorf("error %q does not mention missing key", err.Error())
		}
	})

	t.Run("template syntax error returns error", func(t *testing.T) {
		mf := baseMF(&GeneratorSpec{
			SourceKey:          "skills",
			Template:           "{{ .Item",
			DestinationPattern: "/out/{{ .Item }}.md",
		})
		err := expandGenerator(mf, ctx, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty destination pattern returns error", func(t *testing.T) {
		mf := baseMF(&GeneratorSpec{
			SourceKey:          "skills",
			Template:           "content",
			DestinationPattern: "   ",
		})
		err := expandGenerator(mf, ctx, "")
		if err == nil {
			t.Fatal("expected error for empty destination, got nil")
		}
		if !strings.Contains(err.Error(), "empty string") {
			t.Errorf("error %q should mention empty string", err.Error())
		}
	})

	t.Run("dependsOn inherited by all generated FileSpecs", func(t *testing.T) {
		mf := baseMF(&GeneratorSpec{
			SourceKey:          "skills",
			Template:           "skill: {{ .Item }}",
			DestinationPattern: "/out/{{ .Item }}.md",
			DependsOn:          []string{"homebrew-packages", "dotfiles-base"},
		})
		if err := expandGenerator(mf, ctx, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mf.Spec.Files) != 2 {
			t.Fatalf("want 2 files, got %d", len(mf.Spec.Files))
		}
		for i, f := range mf.Spec.Files {
			if len(f.DependsOn) != 2 {
				t.Errorf("file[%d].DependsOn len = %d, want 2", i, len(f.DependsOn))
				continue
			}
			if f.DependsOn[0] != "homebrew-packages" || f.DependsOn[1] != "dotfiles-base" {
				t.Errorf("file[%d].DependsOn = %v", i, f.DependsOn)
			}
		}
	})

	t.Run("nil dependsOn not inherited (stays nil)", func(t *testing.T) {
		mf := baseMF(&GeneratorSpec{
			SourceKey:          "skills",
			Template:           "skill: {{ .Item }}",
			DestinationPattern: "/out/{{ .Item }}.md",
		})
		if err := expandGenerator(mf, ctx, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for i, f := range mf.Spec.Files {
			if f.DependsOn != nil {
				t.Errorf("file[%d].DependsOn should be nil, got %v", i, f.DependsOn)
			}
		}
	})
}

func TestRenderGeneratorTemplate_ReadFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/skill.md", []byte("skill body content"), 0644); err != nil {
		t.Fatal(err)
	}
	ctx := generatorContext{Item: "bash", Index: 0, Values: map[string]any{}, Env: map[string]string{}, OS: config.OSInfo{}}

	t.Run("relative path reads file", func(t *testing.T) {
		got, err := renderGeneratorTemplate("test", `{{ readFile "skill.md" }}`, ctx, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "skill body content" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("absolute path reads file", func(t *testing.T) {
		got, err := renderGeneratorTemplate("test", `{{ readFile "`+dir+`/skill.md" }}`, ctx, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "skill body content" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		_, err := renderGeneratorTemplate("test", `{{ readFile "missing.md" }}`, ctx, dir)
		if err == nil {
			t.Fatal("expected error for missing file")
		}
		if !strings.Contains(err.Error(), "readFile") {
			t.Errorf("error %q should mention readFile", err.Error())
		}
	})

	t.Run("dynamic path via printf", func(t *testing.T) {
		if err := os.WriteFile(dir+"/bash.md", []byte("bash skill"), 0644); err != nil {
			t.Fatal(err)
		}
		got, err := renderGeneratorTemplate("test", `{{ readFile (printf "%s.md" .Item) }}`, ctx, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "bash skill" {
			t.Errorf("got %q", got)
		}
	})
}

func TestRenderGeneratorTemplate(t *testing.T) {
	baseCtx := generatorContext{
		Item:   "bash",
		Index:  0,
		Values: map[string]any{"repo": "dotfiles"},
		Env:    map[string]string{"HOME": "/home/user"},
		OS:     config.OSInfo{GOOS: "linux"},
	}

	tests := []struct {
		name      string
		tmpl      string
		ctx       generatorContext
		want      string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "scalar item",
			tmpl: "skill: {{ .Item }}",
			ctx:  baseCtx,
			want: "skill: bash",
		},
		{
			name: "index access",
			tmpl: "index={{ .Index }}",
			ctx:  baseCtx,
			want: "index=0",
		},
		{
			name: "non-zero index",
			tmpl: "index={{ .Index }}",
			ctx:  generatorContext{Item: "x", Index: 3, Values: map[string]any{}, Env: map[string]string{}, OS: config.OSInfo{}},
			want: "index=3",
		},
		{
			name: "values access",
			tmpl: "repo={{ .Values.repo }}",
			ctx:  baseCtx,
			want: "repo=dotfiles",
		},
		{
			name: "env access",
			tmpl: "home={{ .Env.HOME }}",
			ctx:  baseCtx,
			want: "home=/home/user",
		},
		{
			name: "os access",
			tmpl: "os={{ .OS.GOOS }}",
			ctx:  baseCtx,
			want: "os=linux",
		},
		{
			name: "map item field access",
			tmpl: "name={{ .Item.name }}",
			ctx: generatorContext{
				Item:   map[string]any{"name": "coder"},
				Index:  0,
				Values: map[string]any{},
				Env:    map[string]string{},
				OS:     config.OSInfo{},
			},
			want: "name=coder",
		},
		{
			name: "sprig function available",
			tmpl: `{{ .Item | upper }}`,
			ctx:  baseCtx,
			want: "BASH",
		},
		{
			name: "destination pattern with item",
			tmpl: "~/.claude/skills/{{ .Item }}.md",
			ctx:  baseCtx,
			want: "~/.claude/skills/bash.md",
		},
		{
			name:      "syntax error",
			tmpl:      "{{ .Item",
			ctx:       baseCtx,
			wantErr:   true,
			errSubstr: "template parse error",
		},
		{
			name:      "runtime error — invalid field on scalar",
			tmpl:      "{{ .Item.nonexistent }}",
			ctx:       baseCtx,
			wantErr:   true,
			errSubstr: "template render error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderGeneratorTemplate("test", tt.tmpl, tt.ctx, t.TempDir())

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
