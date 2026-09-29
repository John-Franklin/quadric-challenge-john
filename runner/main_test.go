package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

// fakeBackend records log uploads and reports "cancelled" once cancel is set.
type fakeBackend struct {
	cancelled atomic.Bool
	mu        sync.Mutex
	uploads   []string
}

func (b *fakeBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	b.mu.Lock()
	b.uploads = append(b.uploads, string(body))
	b.mu.Unlock()
	status := "running"
	if b.cancelled.Load() {
		status = jobStatusCancelled
	}
	w.Header().Set("Job-Status", status)
	w.WriteHeader(http.StatusNoContent)
}

func (b *fakeBackend) lastUpload() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.uploads[len(b.uploads)-1]
}

func withFakeBackend(t *testing.T) *fakeBackend {
	t.Helper()
	backend := &fakeBackend{}
	srv := httptest.NewServer(backend)
	t.Cleanup(srv.Close)

	prevClient, prevInterval := client, statusCheckInterval
	client = resty.New().SetBaseURL(srv.URL)
	statusCheckInterval = 100 * time.Millisecond
	t.Cleanup(func() { client, statusCheckInterval = prevClient, prevInterval })
	return backend
}

func runJob(t *testing.T, job Job) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- executeJob(context.Background(), job) }()
	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("job was not stopped after cancellation")
		return nil
	}
}

// A script that never prints sends no log uploads, so the runner must poll for Job-Status.
func TestExecuteJobCancelsSilentPythonScript(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	backend := withFakeBackend(t)
	time.AfterFunc(500*time.Millisecond, func() { backend.cancelled.Store(true) })

	job := Job{ID: 1, Action: "run_python", Code: "import time\ntime.sleep(60)"}
	if err := runJob(t, job); err != nil {
		t.Fatalf("executeJob returned %v, want nil for a cancelled job", err)
	}
	if got, want := backend.lastUpload(), "Job cancelled by user, runner stopped the task\n"; got != want {
		t.Fatalf("last upload = %q, want %q", got, want)
	}
}

func TestExecuteJobCancelsPrintingPythonScript(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	backend := withFakeBackend(t)
	time.AfterFunc(500*time.Millisecond, func() { backend.cancelled.Store(true) })

	job := Job{ID: 1, Action: "run_python", Code: "import time\nwhile True:\n    print('tick')\n    time.sleep(0.05)"}
	if err := runJob(t, job); err != nil {
		t.Fatalf("executeJob returned %v, want nil for a cancelled job", err)
	}
	if got, want := backend.lastUpload(), "Job cancelled by user, runner stopped the task\n"; got != want {
		t.Fatalf("last upload = %q, want %q", got, want)
	}
}
