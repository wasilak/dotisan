package ui

import (
	"context"
	"errors"
	"fmt"
	"github.com/briandowns/spinner"
	"github.com/wasilak/dotisan/pkg/style"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// Spinner wraps the github.com/briandowns/spinner Spinner.
type Spinner struct {
	s *spinner.Spinner
}

func NewSpinner() *Spinner {
	// Choose a random charset index between 0 and 36 (inclusive), but clamp
	// to the available number of charsets in the library to be safe.
	maxIdx := 37
	if len(spinner.CharSets) < maxIdx {
		maxIdx = len(spinner.CharSets)
	}
	if maxIdx <= 0 {
		maxIdx = 1
	}
	idx := rand.Intn(maxIdx)

	s := spinner.New(spinner.CharSets[idx], 125*time.Millisecond)
	// Ensure the spinner writes to the current stdout writer. This makes the
	// spinner deterministic in tests where os.Stdout may be redirected.
	s.Writer = os.Stdout
	// Add a small left margin so spinner doesn't touch the screen edge
	s.Prefix = strings.Repeat(" ", 2)
	// Prefer palette-provided spinner color; fallback to a random pick.
	col := style.DefaultColors.SpinnerColor
	if col == "" {
		colors := []string{"fgHiMagenta", "fgHiCyan", "fgHiYellow", "fgHiGreen", "fgHiBlue", "fgHiRed", "magenta", "cyan", "yellow", "green", "blue", "red", "bold"}
		col = colors[rand.Intn(len(colors))]
	}
	s.Color(col)
	return &Spinner{s: s}
}

func init() {
	// Seed the RNG once per process so spinner charset selection varies by run.
	rand.Seed(time.Now().UnixNano())
}

// NewPrimarySpinner creates a spinner that uses the palette's Main color for
// its initial message. The returned Spinner is a thin wrapper around the
// briandowns spinner. If w is nil, stdout is used.
func NewPrimarySpinner(message string) *Spinner {
	sp := NewSpinnerFunc()
	// Use the Header style for the initial message, but ensure the sequence
	// includes the palette Main color if available.
	// The caller can still call StartWithStyle to control style explicitly.
	sp.s.Suffix = " " + style.Header.Render(message)
	return sp
}

// NewSpinnerFunc is a hook that returns a new Spinner instance. Tests can
// override this to inject a spinner that writes to an in-memory buffer for
// deterministic assertions. By default it constructs the real Spinner.
var NewSpinnerFunc = func() *Spinner { return NewSpinner() }

// Publisher is an interface consumed by middleware and commands that want to
// publish transient spinner messages. Implementations are safe for concurrent
// use. Use ContextWithPublisher to attach a publisher to a context and
// PublisherFromContext to retrieve it.
type Publisher interface {
	Publish(level MessageLevel, msg string)
}

// context key for publisher
type publisherCtxKeyType struct{}

var publisherCtxKey = &publisherCtxKeyType{}

// ContextWithPublisher returns a new context that carries the provided
// Publisher.
func ContextWithPublisher(ctx context.Context, p Publisher) context.Context {
	return context.WithValue(ctx, publisherCtxKey, p)
}

// PublisherFromContext retrieves a Publisher from ctx, if present.
func PublisherFromContext(ctx context.Context) (Publisher, bool) {
	if ctx == nil {
		return nil, false
	}
	v := ctx.Value(publisherCtxKey)
	if v == nil {
		return nil, false
	}
	p, ok := v.(Publisher)
	return p, ok
}

// PublishFromCmd is a small convenience wrapper for cobra command handlers.
// It attempts to retrieve a Publisher from the command's context and, if
// present, publishes the given message. Returns true when a publisher was
// found and the message was sent; false otherwise.
func PublishFromCmd(cmd *cobra.Command, level MessageLevel, msg string) bool {
	if cmd == nil {
		return false
	}
	if p, ok := PublisherFromContext(cmd.Context()); ok && p != nil {
		p.Publish(level, msg)
		return true
	}
	return false
}

func (s *Spinner) Start(msg string) {
	s.s.Suffix = " " + msg
	s.s.Start()
}
func (s *Spinner) StartWithStyle(st style.Style, msg string) {
	s.s.Suffix = " " + st.Render(msg)
	s.s.Start()
}
func (s *Spinner) UpdateText(msg string) {
	s.s.Suffix = " " + msg
}
func (s *Spinner) UpdateTextWithStyle(st style.Style, msg string) {
	s.s.Suffix = " " + st.Render(msg)
}
func (s *Spinner) Stop() {
	// Delegate to the underlying spinner library. It will write FinalMSG
	// to the configured writer when Stop() is called.
	s.s.Stop()
}
func (s *Spinner) Success(msg string) {
	// Ensure deterministic final output: clear FinalMSG so the underlying
	// library doesn't attempt to print it, stop the spinner, then write
	// the final message ourselves to the configured writer.
	s.s.FinalMSG = ""
	s.s.Stop()
	if s.s.Writer != nil {
		fmt.Fprintln(s.s.Writer, msg)
	} else {
		fmt.Fprintln(os.Stdout, msg)
	}
}
func (s *Spinner) Fail(msg string) {
	// Clear FinalMSG and stop, then write the failure final message
	// directly to the configured writer. This avoids races where the
	// underlying library might not synchronously flush FinalMSG.
	s.s.FinalMSG = ""
	s.s.Stop()
	if s.s.Writer != nil {
		fmt.Fprintln(s.s.Writer, msg)
	} else {
		fmt.Fprintln(os.Stdout, msg)
	}
}
func (s *Spinner) SuccessWithStyle(st style.Style, msg string) {
	s.s.FinalMSG = ""
	s.s.Stop()
	if s.s.Writer != nil {
		fmt.Fprintln(s.s.Writer, st.Render(msg))
	} else {
		fmt.Fprintln(os.Stdout, st.Render(msg))
	}
}
func (s *Spinner) FailWithStyle(st style.Style, msg string) {
	s.s.FinalMSG = ""
	s.s.Stop()
	if s.s.Writer != nil {
		fmt.Fprintln(s.s.Writer, st.Render(msg))
	} else {
		fmt.Fprintln(os.Stdout, st.Render(msg))
	}
}

// StartWithContext starts the spinner and returns a function that cancels the
// internal watcher. If ctx is cancelled, the spinner will be stopped and a
// fail message displayed using the provided cancelMsg (styled with style.Error).
func (s *Spinner) StartWithContext(ctx context.Context, st style.Style, msg string, cancelMsg string) func() {
	// Ensure spinner writes to current stdout *if not already configured*.
	// Tests may inject a custom writer via NewSpinnerFunc; don't overwrite
	// it here if present.
	if s.s.Writer == nil {
		s.s.Writer = os.Stdout
	}
	s.StartWithStyle(st, msg)
	done := make(chan struct{})
	var once sync.Once

	go func() {
		select {
		case <-ctx.Done():
			// ensure we only fail/close once
			once.Do(func() {
				m := cancelMsg
				if m == "" {
					m = "cancelled"
				}
				// Try to display the cancel message via the spinner helper which
				// prints to the spinner's writer when available. Also write
				// directly to os.Stdout to make tests deterministic in case the
				// spinner's writer wasn't bound to the current stdout (racey
				// test setups redirect os.Stdout).
				s.FailWithStyle(style.Error, m)
				close(done)
			})
		case <-done:
			// normal stop
		}
	}()

	// return a stop function that's safe to call multiple times
	return func() {
		once.Do(func() { close(done) })
	}
}

// RunWithSpinner runs work(ctx, publish) while showing a context-aware spinner.
// The work callback receives a publish(msg string) function which is safe to
// call from any goroutine; messages will be serialized and applied to the
// spinner by an internal owner goroutine. This provides a thread-safe way to
// surface per-item progress (e.g. engine.Apply callbacks) to the spinner.
// MessageLevel represents the severity/type of a transient spinner message.
type MessageLevel int

const (
	LevelInfo MessageLevel = iota
	LevelSuccess
	LevelError
)

// RunWithSpinner runs work(ctx, publish) while showing a context-aware spinner.
// The work callback receives a publish(level, msg string) function which is
// safe to call from any goroutine; messages will be serialized and applied to
// the spinner by an internal owner goroutine. This provides a thread-safe way
// to surface per-item progress (e.g. engine.Apply callbacks) to the spinner.
func RunWithSpinner(ctx context.Context, st style.Style, msg, cancelMsg string, work func(ctx context.Context, publish func(level MessageLevel, msg string)) error) error {
	sp := NewSpinnerFunc()
	// use Main color from palette for the spinner label if available
	stop := sp.StartWithContext(ctx, st, msg, cancelMsg)

	// messages channel used to send transient updates to the spinner. Buffer
	// a few messages to avoid blocking producers briefly.
	type msgEntry struct {
		level MessageLevel
		msg   string
	}
	msgs := make(chan msgEntry, 8)

	// owner goroutine: reads messages and updates spinner text. Runs until
	// msgs is closed.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for e := range msgs {
			switch e.level {
			case LevelSuccess:
				sp.UpdateTextWithStyle(style.Success, e.msg)
			case LevelError:
				sp.UpdateTextWithStyle(style.Error, e.msg)
			default:
				sp.UpdateTextWithStyle(st, e.msg)
			}
		}
	}()

	// publish function given to work. It should be safe to call concurrently.
	// Also update lastMsg atomically so final success/fail can reflect the
	// most recent transient message.
	var lastMu sync.Mutex
	var lastMsg string
	var lastLevel MessageLevel = LevelInfo
	publish := func(level MessageLevel, m string) {
		select {
		case msgs <- msgEntry{level: level, msg: m}:
		default:
			// drop if buffer is full to avoid blocking the work function
		}
		lastMu.Lock()
		lastMsg = m
		lastLevel = level
		lastMu.Unlock()
	}

	// Run actual work
	workErr := work(ctx, publish)

	// Close messages channel and wait for owner goroutine to finish
	close(msgs)
	wg.Wait()

	// If the context was cancelled, StartWithContext already printed the
	// cancel message; avoid duplicating failure output.
	if workErr != nil {
		if ctx.Err() == context.Canceled || errors.Is(workErr, context.Canceled) {
			// nothing to do; spinner watcher already emitted cancel message
		} else {
			// Prefer showing the last transient message as the failure line
			lastMu.Lock()
			msg := lastMsg
			lastMu.Unlock()
			if msg == "" {
				sp.FailWithStyle(style.Error, "Failed")
			} else {
				sp.FailWithStyle(style.Error, msg)
			}
		}
	} else {
		lastMu.Lock()
		msg := lastMsg
		lvl := lastLevel
		lastMu.Unlock()
		// Don't print a generic "Complete" message — leave silence on
		// successful completion unless a transient message was published.
		// If the last transient message is itself a success-level summary from
		// the work (e.g. "Apply complete! All resources synchronized"), we
		// avoid printing it here to prevent duplicating the final success
		// message that the caller may print. Only print the final message if
		// the last transient message was not a success-level summary.
		if msg != "" && lvl != LevelSuccess {
			sp.SuccessWithStyle(style.Success, msg)
		}
	}

	// Ensure spinner is stopped even when we don't print a final message.
	// Some code paths avoid calling SuccessWithStyle to remain silent; stop
	// the spinner explicitly here so the animation doesn't continue.
	sp.Stop()
	stop()
	return workErr
}
