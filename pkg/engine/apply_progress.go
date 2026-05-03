package engine

// ApplyOptions contains options for the Apply operation.
type ApplyOptions struct {
	Confirm bool

	// Callbacks for progress tracking. Can be nil.
	OnItemStart    func(kind, group, item string)
	OnItemComplete func(kind, group, item string, err error)
}
