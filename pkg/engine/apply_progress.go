package engine

// MessageLevel represents the severity/type of an OnMessage callback.
type MessageLevel int

// MessageLevel constants for OnMessage
const (
	MessageLevelInfo MessageLevel = iota
	MessageLevelSuccess
	MessageLevelError
	MessageLevelWarning
)

// ApplyOptions contains options for the Apply operation.
type ApplyOptions struct {
	Confirm bool

	// Callbacks for progress tracking. Can be nil.
	OnItemStart    func(kind, group, item string)
	OnItemComplete func(kind, group, item string, err error)
	// OnMessage is an optional callback to receive final or summary messages
	// from the apply engine. If provided, messages will be delivered as a
	// (level, message) pair where level is a MessageLevel constant.
	OnMessage func(level MessageLevel, msg string)
}
