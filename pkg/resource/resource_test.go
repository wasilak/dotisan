package resource

import (
	"testing"
)

func TestMetadata_GetNamespace(t *testing.T) {
	tests := []struct {
		name     string
		metadata Metadata
		want     string
	}{
		{
			name:     "empty namespace returns default",
			metadata: Metadata{Name: "test"},
			want:     "default",
		},
		{
			name:     "explicit namespace returned",
			metadata: Metadata{Name: "test", Namespace: "work"},
			want:     "work",
		},
		{
			name:     "default namespace returned as-is",
			metadata: Metadata{Name: "test", Namespace: "default"},
			want:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metadata.GetNamespace()
			if got != tt.want {
				t.Errorf("GetNamespace() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMetadata_ResourceID(t *testing.T) {
	m := Metadata{Name: "core-tools", Namespace: "brew"}
	want := "brew/core-tools"
	if got := m.ResourceID(); got != want {
		t.Errorf("ResourceID() = %q, want %q", got, want)
	}
}

func TestBaseResource_Getters(t *testing.T) {
	br := BaseResource{
		APIVersion: "github.com/wasilak/nim/v1",
		Kind:       KindHomeBrewPackages,
		Metadata:   Metadata{Name: "test", Namespace: "default"},
	}

	if got := br.GetAPIVersion(); got != "github.com/wasilak/nim/v1" {
		t.Errorf("GetAPIVersion() = %q, want %q", got, "github.com/wasilak/nim/v1")
	}

	if got := br.GetKind(); got != KindHomeBrewPackages {
		t.Errorf("GetKind() = %q, want %q", got, KindHomeBrewPackages)
	}

	meta := br.GetMetadata()
	if meta.Name != "test" {
		t.Errorf("GetMetadata().Name = %q, want %q", meta.Name, "test")
	}
}

func TestIsSupportedAPIVersion(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"github.com/wasilak/nim/v1", true},
		{"github.com/wasilak/nim/v2", false},
		{"terraform/v1", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := IsSupportedAPIVersion(tt.version); got != tt.want {
				t.Errorf("IsSupportedAPIVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestValidateStruct(t *testing.T) {
	tests := []struct {
		name    string
		struct_ any
		wantErr bool
	}{
		{
			name: "valid BaseResource",
			struct_: BaseResource{
				APIVersion: "github.com/wasilak/nim/v1",
				Kind:       KindHomeBrewPackages,
				Metadata:   Metadata{Name: "test"},
			},
			wantErr: false,
		},
		{
			name: "missing apiVersion",
			struct_: BaseResource{
				Kind:     KindHomeBrewPackages,
				Metadata: Metadata{Name: "test"},
			},
			wantErr: true,
		},
		{
			name: "missing kind",
			struct_: BaseResource{
				APIVersion: "github.com/wasilak/nim/v1",
				Metadata:   Metadata{Name: "test"},
			},
			wantErr: true,
		},
		{
			name: "missing metadata.name",
			struct_: BaseResource{
				APIVersion: "github.com/wasilak/nim/v1",
				Kind:       KindHomeBrewPackages,
				Metadata:   Metadata{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.struct_)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompileNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "bare string returns nil error and nil namespaceRe",
			namespace: "work",
			wantErr:   false,
		},
		{
			name:      "empty string returns nil error",
			namespace: "",
			wantErr:   false,
		},
		{
			name:      "pattern compiles successfully",
			namespace: "/work.*/",
			wantErr:   false,
		},
		{
			name:      "single char pattern is valid",
			namespace: "/w/",
			wantErr:   false,
		},
		{
			name:      "alternation pattern compiles",
			namespace: "/(work|personal)/",
			wantErr:   false,
		},
		{
			name:      "invalid regex returns error",
			namespace: "/invalid[/",
			wantErr:   true,
			errMsg:    `parsing namespace "/invalid[/"`,
		},
		{
			name:      "just slashes is empty pattern - valid",
			namespace: "//",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Metadata{Name: "test", Namespace: tt.namespace}
			err := m.CompileNamespace()
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if err.Error()[:len(tt.errMsg)] != tt.errMsg {
					t.Errorf("CompileNamespace() error message = %q, want prefix %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestMatchesNamespace(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		activeNS   string
		wantMatch  bool
	}{
		// Branch 1: regex match (namespaceRe != nil)
		{
			name:       "regex /work.*/ matches work-laptop",
			namespace:  "/work.*/",
			activeNS:   "work-laptop",
			wantMatch:  true,
		},
		{
			name:       "regex /work.*/ does not match personal",
			namespace:  "/work.*/",
			activeNS:   "personal",
			wantMatch:  false,
		},
		{
			name:       "regex /(work|personal)/ matches work",
			namespace:  "/(work|personal)/",
			activeNS:   "work",
			wantMatch:  true,
		},
		{
			name:       "regex /(work|personal)/ matches personal",
			namespace:  "/(work|personal)/",
			activeNS:   "personal",
			wantMatch:  true,
		},
		{
			name:       "regex /(work|personal)/ does not match other",
			namespace:  "/(work|personal)/",
			activeNS:   "other",
			wantMatch:  false,
		},
		{
			name:       "regex is case-insensitive with (?i) prefix",
			namespace:  "/work.*/",
			activeNS:   "WORK-LAPTOP",
			wantMatch:  true,
		},
		// Branch 2: implicit default (namespaceRe == nil, Namespace == "")
		{
			name:       "empty namespace matches default",
			namespace:  "",
			activeNS:   "default",
			wantMatch:  true,
		},
		{
			name:       "empty namespace does not match work",
			namespace:  "",
			activeNS:   "work",
			wantMatch:  false,
		},
		// Branch 3: exact match (namespaceRe == nil, Namespace != "")
		{
			name:       "explicit work matches work",
			namespace:  "work",
			activeNS:   "work",
			wantMatch:  true,
		},
		{
			name:       "explicit work does not match personal",
			namespace:  "work",
			activeNS:   "personal",
			wantMatch:  false,
		},
		{
			name:       "explicit default matches default",
			namespace:  "default",
			activeNS:   "default",
			wantMatch:  true,
		},
		{
			name:       "explicit default does not match work",
			namespace:  "default",
			activeNS:   "work",
			wantMatch:  false,
		},
		// Edge cases
		{
			name:       "regex // (empty pattern) matches anything",
			namespace:  "//",
			activeNS:   "anything",
			wantMatch:  true,
		},
		{
			name:       "regex /work/ matches work (substring match)",
			namespace:  "/work/",
			activeNS:   "work",
			wantMatch:  true,
		},
		{
			name:       "regex /work/ matches work-laptop (substring)",
			namespace:  "/work/",
			activeNS:   "work-laptop",
			wantMatch:  true,
		},
		{
			name:       "regex /^work$/ matches work exactly (anchored)",
			namespace:  "/^work$/",
			activeNS:   "work",
			wantMatch:  true,
		},
		{
			name:       "regex /^work$/ does not match work-laptop (anchored)",
			namespace:  "/^work$/",
			activeNS:   "work-laptop",
			wantMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create metadata and compile namespace (replicating real parse-time flow)
			m := &Metadata{Name: "test", Namespace: tt.namespace}
			if err := m.CompileNamespace(); err != nil {
				t.Fatalf("CompileNamespace() failed: %v", err)
			}

			// Create BaseResource with the compiled metadata
			br := BaseResource{
				APIVersion: "github.com/wasilak/nim/v1",
				Kind:       KindHomeBrewPackages,
				Metadata:   *m,
			}

			// Test MatchesNamespace
			got := br.MatchesNamespace(tt.activeNS)
			if got != tt.wantMatch {
				t.Errorf("MatchesNamespace(%q) = %v, want %v", tt.activeNS, got, tt.wantMatch)
			}
		})
	}
}
