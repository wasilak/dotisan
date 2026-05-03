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
		APIVersion: "github.com/wasilak/dotisan/v1",
		Kind:       KindHomeBrewPackages,
		Metadata:   Metadata{Name: "test", Namespace: "default"},
	}

	if got := br.GetAPIVersion(); got != "github.com/wasilak/dotisan/v1" {
		t.Errorf("GetAPIVersion() = %q, want %q", got, "github.com/wasilak/dotisan/v1")
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
		{"github.com/wasilak/dotisan/v1", true},
		{"github.com/wasilak/dotisan/v2", false},
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
		struct_ interface{}
		wantErr bool
	}{
		{
			name: "valid BaseResource",
			struct_: BaseResource{
				APIVersion: "github.com/wasilak/dotisan/v1",
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
				APIVersion: "github.com/wasilak/dotisan/v1",
				Metadata:   Metadata{Name: "test"},
			},
			wantErr: true,
		},
		{
			name: "missing metadata.name",
			struct_: BaseResource{
				APIVersion: "github.com/wasilak/dotisan/v1",
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
