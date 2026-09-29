package models

import "testing"

func TestTerminalStatus(t *testing.T) {
	tests := map[string]string{
		"Starting PI calculation...\n":                    "",
		"Word 3: dolor\n":                                 "",
		"Error: Calculation overflow detected\n":          "",
		"PI calculation completed\n":                      StatusCompleted,
		"Lorem Ipsum generation completed\n":              StatusCompleted,
		"ERROR: calculation failed: overflow\n":           StatusFailed,
		"Word 9: x\nLorem Ipsum generation completed\n":   StatusCompleted,
		"PI calculation completed\nERROR: late failure\n": StatusFailed,
	}
	for input, want := range tests {
		if got := terminalStatus(input); got != want {
			t.Errorf("terminalStatus(%q) = %q, want %q", input, got, want)
		}
	}
}
