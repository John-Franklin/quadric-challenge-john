// Package tasks implements the job actions a runner can execute.
package tasks

import (
	"context"
	"strings"
)

// Logf emits one log line to the backend; a trailing newline is added if missing.
type Logf func(format string, args ...any)

// Input carries per-job data from the backend; fields are empty when an action does not use them.
type Input struct {
	Code   string
	Digits int
	Words  int
}

// Task runs a job action. Its last log line on success must end in " completed",
// which the backend uses to mark the job completed.
type Task func(ctx context.Context, in Input, logf Logf) error

var registry = map[string]Task{
	"calculate_pi": CalculatePi,
	"lorem_ipsum":  LoremIpsum,
	"run_python":   RunPython,
}

// Lookup returns the task for an action name, case-insensitively.
func Lookup(action string) (Task, bool) {
	t, ok := registry[strings.ToLower(action)]
	return t, ok
}
