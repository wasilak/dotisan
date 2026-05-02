package style

import (
	"fmt"
	"github.com/pterm/pterm"
)

type ApplyProgress struct {
	progressBar *pterm.ProgressbarPrinter
	spinner     *pterm.SpinnerPrinter
	multi       *pterm.MultiPrinter
	current     int
	total       int
	item        string
}

// NewApplyProgress creates a progressbar + spinner for apply steps.
func NewApplyProgress(total int) *ApplyProgress {
	multi := &pterm.DefaultMultiPrinter

	pb := pterm.DefaultProgressbar.WithWriter(multi.NewWriter())
	pb = pb.WithTitle("Applying resources").WithTotal(total).WithRemoveWhenDone(false).WithShowCount(true).WithShowElapsedTime(true).WithShowTitle(true).WithMaxWidth(pterm.GetTerminalWidth())
	progressBar, _ := pb.Start()

	spinner := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).WithShowTimer(false)
	spinnerPrinter, _ := spinner.Start("Waiting for resource...")

	multi.Start()

	ap := &ApplyProgress{
		progressBar: progressBar,
		spinner:     spinnerPrinter,
		multi:       multi,
		total:       total,
	}
	return ap
}

// StartItem sets spinner text to current resource
func (ap *ApplyProgress) StartItem(kind, group, item string) {
	ap.item = fmt.Sprintf("[%s] %s/%s", kind, group, item)
	ap.spinner.UpdateText(ap.item)
}

// CompleteItem advances the progress bar
func (ap *ApplyProgress) CompleteItem(err error) {
	ap.current++
	status := ""
	if err != nil {
		status = pterm.LightRed("[FAILED]")
	} else {
		status = pterm.LightGreen("[OK]")
	}
	ap.progressBar.Increment() // show always, for every item
	ap.spinner.UpdateText(fmt.Sprintf("%s %s", ap.item, status))
}

// Stop shuts down the progress UI
func (ap *ApplyProgress) Stop() {
	ap.spinner.Stop()
	ap.progressBar.Stop()
	ap.multi.Stop()
}
