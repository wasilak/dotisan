package resource

// ResourceGroup represents a level-2 entity (e.g., "core-tools" containing multiple packages)
type ResourceGroup struct {
	Kind  string
	Name  string
	Items []ResourceItem
	RawSpec interface{} // Original resource spec for provider use
}

// ResourceItem represents a level-3 entity (e.g., "ripgrep" within "core-tools")
type ResourceItem struct {
	Name     string
	Version  string
	Checksum string
	Extra    map[string]interface{} // Provider-specific data
}

// ItemState represents the state of an individual item
type ItemState struct {
	Name     string                 `json:"name"`
	Version  string                 `json:"version,omitempty"`
	Checksum string                 `json:"checksum,omitempty"`
	Status   string                 `json:"status,omitempty"` // present, absent, etc.
	Extra    map[string]interface{} `json:"extra,omitempty"`
}

// ToGroup converts a Resource to its Group representation
// Each resource type should implement this method
func (br *BaseResource) ToGroup(items []ResourceItem) ResourceGroup {
	return ResourceGroup{
		Kind:  br.Kind,
		Name:  br.Metadata.Name,
		Items: items,
	}
}
