package tasks

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const pi100 = "3.1415926535897932384626433832795028841971693993751058209749445923078164062862089986280348253421170679"

func collect() (*[]string, Logf) {
	var lines []string
	return &lines, func(format string, args ...any) {
		lines = append(lines, fmt.Sprintf(format, args...))
	}
}

func TestComputePiKnownDigits(t *testing.T) {
	_, logf := collect()
	got, err := computePi(context.Background(), 100, logf)
	if err != nil {
		t.Fatal(err)
	}
	if got != pi100 {
		t.Fatalf("computePi(100) =\n%s\nwant\n%s", got, pi100)
	}
}

// Guard digits must keep truncation error out of the reported digits.
func TestComputePiStableAcrossPrecision(t *testing.T) {
	_, logf := collect()
	short, err := computePi(context.Background(), defaultPiDigits, logf)
	if err != nil {
		t.Fatal(err)
	}
	long, err := computePi(context.Background(), defaultPiDigits+200, logf)
	if err != nil {
		t.Fatal(err)
	}
	if len(short) != defaultPiDigits+2 {
		t.Fatalf("len = %d, want %d", len(short), defaultPiDigits+2)
	}
	if !strings.HasPrefix(long, short) {
		t.Fatal("1000-digit result is not a prefix of the 1200-digit result")
	}
}

func TestCalculatePiLogs(t *testing.T) {
	lines, logf := collect()
	if err := CalculatePi(context.Background(), Input{}, logf); err != nil {
		t.Fatal(err)
	}
	if last := (*lines)[len(*lines)-1]; last != "PI calculation completed" {
		t.Fatalf("last line = %q", last)
	}
	joined := strings.Join(*lines, "\n")
	if !strings.Contains(joined, pi100[:52]) {
		t.Fatal("logs do not contain the first 50 digits of pi")
	}
}

func TestLoremIpsum(t *testing.T) {
	lines, logf := collect()
	if err := generateLorem(context.Background(), rand.New(rand.NewPCG(1, 2)), 250, logf); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix((*lines)[1], "Paragraph 1: "+loremOpening) {
		t.Fatalf("first paragraph = %q", (*lines)[1])
	}
	words := 0
	for _, l := range *lines {
		if text, ok := strings.CutPrefix(l, "Paragraph "); ok {
			_, text, _ = strings.Cut(text, ": ")
			words += len(strings.Fields(text))
			if !strings.HasSuffix(l, ".") {
				t.Errorf("paragraph does not end with a period: %q", l)
			}
		}
	}
	if words != 250 {
		t.Fatalf("words = %d, want 250", words)
	}
	if last := (*lines)[len(*lines)-1]; last != "Lorem Ipsum generation completed" {
		t.Fatalf("last line = %q", last)
	}
}

func TestLoremIpsumFewerWordsThanOpening(t *testing.T) {
	lines, logf := collect()
	if err := generateLorem(context.Background(), rand.New(rand.NewPCG(1, 2)), 5, logf); err != nil {
		t.Fatal(err)
	}
	if got, want := (*lines)[1], "Paragraph 1: Lorem ipsum dolor sit amet."; got != want {
		t.Fatalf("paragraph = %q, want %q", got, want)
	}
}

func TestCalculatePiDigits(t *testing.T) {
	lines, logf := collect()
	if err := CalculatePi(context.Background(), Input{Digits: 10}, logf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(*lines, "\n"), "PI = "+pi100[:12]+"...") {
		t.Fatalf("logs = %q", *lines)
	}
}

func TestTasksStopWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for name, task := range registry {
		_, logf := collect()
		if err := task(ctx, Input{Code: "print(1)"}, logf); !errors.Is(err, context.Canceled) {
			t.Errorf("%s: err = %v, want context.Canceled", name, err)
		}
	}
}

func TestLookup(t *testing.T) {
	for _, action := range []string{"calculate_pi", "LOREM_IPSUM", "run_python"} {
		if _, ok := Lookup(action); !ok {
			t.Errorf("Lookup(%q) not found", action)
		}
	}
	if _, ok := Lookup("nope"); ok {
		t.Error("Lookup(\"nope\") found a task")
	}
}

func requirePython(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not installed")
	}
}

func TestRunPythonStreamsOutput(t *testing.T) {
	requirePython(t)
	lines, logf := collect()
	code := "import sys\nprint('hello')\nprint('ERROR: not terminal')\nprint('oops', file=sys.stderr)\n"
	if err := RunPython(context.Background(), Input{Code: code}, logf); err != nil {
		t.Fatal(err)
	}
	got := (*lines)[1 : len(*lines)-1]
	want := []string{OutputPrefix + "hello", OutputPrefix + "ERROR: not terminal", OutputPrefix + "oops"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("output = %q, want %q", got, want)
	}
	if last := (*lines)[len(*lines)-1]; last != "Python script completed" {
		t.Fatalf("last line = %q", last)
	}
}

func TestRunPythonNonZeroExit(t *testing.T) {
	requirePython(t)
	lines, logf := collect()
	err := RunPython(context.Background(), Input{Code: "raise SystemExit(3)"}, logf)
	if err == nil || !strings.Contains(err.Error(), "exit status 3") {
		t.Fatalf("err = %v, want exit status 3", err)
	}
	if last := (*lines)[len(*lines)-1]; strings.HasSuffix(last, " completed") {
		t.Fatalf("failed script logged a completion line: %q", last)
	}
}

func TestRunPythonStopsOnCancel(t *testing.T) {
	requirePython(t)
	ctx, cancel := context.WithCancel(context.Background())
	_, logf := collect()
	done := make(chan error, 1)
	go func() { done <- RunPython(ctx, Input{Code: "import time\ntime.sleep(60)"}, logf) }()
	time.Sleep(200 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("script was not stopped after cancellation")
	}
}

func TestRunPythonRequiresCode(t *testing.T) {
	_, logf := collect()
	if err := RunPython(context.Background(), Input{Code: "  \n"}, logf); err == nil {
		t.Fatal("expected an error for empty code")
	}
}
