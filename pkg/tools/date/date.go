package date

import (
	"context"
	"time"

	"github.com/tj/go-naturaldate"
	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/tools"
)

// Tool defines a tool implementation for determining a date based on a relative reference.
type Tool struct {
	CallbacksHandler callbacks.Handler
}

func New() *Tool {
	return &Tool{}
}

// Ensure the thing implements the interface
var _ tools.Tool = Tool{}

// Name returns a name for the tool.
func (t Tool) Name() string {
	return "Date"
}

// Description returns a description for the tool.
func (t Tool) Description() string {
	return `Determines the date associated with a given relative date string.  
	For instance, 'today' would return the current date, 'tomorrow' would return the date for the next day, and 'yesterday' would return the date for the previous day.
	You must pass a relative date string as input.

	Examples of valid input include:
	now
	today
	yesterday
	5 minutes ago
	three days ago
	last month
	next month
	one year from now
	yesterday at 10am
	last sunday at 5:30pm
	sunday at 22:45
	next January
	last February
	December 25th at 7:30am
	10am
	10:05pm
	10:05:22pm
	5 days from now
	the 25th of December at 7:30am
	in two weeks`
}

// Call generates todays date, formats it as a string, and returns it.
func (t Tool) Call(ctx context.Context, input string) (string, error) {
	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolStart(ctx, input)
	}

	// I've seen instances where the data is wrapped in quotes, so going to try to remove them
	input = removeQuotes(input)

	// Parse the date string
	refTime := time.Now()
	d, err := naturaldate.Parse(input, refTime, naturaldate.WithDirection(naturaldate.Future))
	if err != nil {
		if t.CallbacksHandler != nil {
			t.CallbacksHandler.HandleToolError(ctx, err)
		}
		return "", err
	}

	targetDate := d.Format("2006-01-02")
	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolEnd(ctx, targetDate)
	}
	return targetDate, nil
}

func removeQuotes(s string) string {
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}
