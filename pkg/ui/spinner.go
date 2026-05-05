package ui

import (
	"github.com/briandowns/spinner"
	"time"
)

// Spinner wraps the github.com/briandowns/spinner Spinner.
type Spinner struct {
	s *spinner.Spinner
}

func NewSpinner() *Spinner {
	s := spinner.New(spinner.CharSets[28], 125*time.Millisecond)
	return &Spinner{s: s}
}

func (s *Spinner) Start(msg string) {
	s.s.Suffix = " " + msg
	s.s.Start()
}
func (s *Spinner) UpdateText(msg string) {
	s.s.Suffix = " " + msg
}
func (s *Spinner) Stop() {
	s.s.Stop()
}
func (s *Spinner) Success(msg string) {
	s.s.FinalMSG = msg + "\n"
	s.s.Stop()
}
func (s *Spinner) Fail(msg string) {
	s.s.FinalMSG = msg + "\n"
	s.s.Stop()
}
