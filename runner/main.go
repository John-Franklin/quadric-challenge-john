// Command runner polls the backend for jobs and executes them.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"

	"runner/tasks"
)

const (
	backendURL   = "http://localhost:8080"
	pollInterval = 5 * time.Second

	jobStatusCancelled = "cancelled"

	// Matches the backend's default; set RUNNER_TOKEN on both outside local development.
	devRunnerToken = "dev-runner-token"
)

// statusCheckInterval is a var so tests can shorten it.
var statusCheckInterval = 2 * time.Second

// Runner is the registration payload sent to the backend.
type Runner struct {
	ID string `json:"id"`
}

// Job is a job acquired from the backend.
type Job struct {
	ID     uint64 `json:"id"`
	Action string `json:"action"`
	Code   string `json:"code"`
	Digits int    `json:"digits"`
	Words  int    `json:"words"`
}

var client *resty.Client

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	runnerToken := os.Getenv("RUNNER_TOKEN")
	if runnerToken == "" {
		runnerToken = devRunnerToken
	}
	client = resty.New().
		SetBaseURL(backendURL).
		SetHeader("X-Runner-Token", runnerToken).
		SetTimeout(30 * time.Second)

	runnerID, err := registerRunner(ctx)
	if err != nil {
		return fmt.Errorf("register runner: %w", err)
	}
	log.Printf("Runner registered with ID: %s", runnerID)

	for {
		if err := pollAndExecute(ctx, runnerID); err != nil {
			log.Printf("Error polling/executing job: %v", err)
		}
		select {
		case <-ctx.Done():
			log.Println("Shutting down runner...")
			return nil
		case <-time.After(pollInterval):
		}
	}
}

func registerRunner(ctx context.Context) (string, error) {
	hostname, _ := os.Hostname()
	runnerID := fmt.Sprintf("runner-%s-%d", hostname, time.Now().Unix())

	resp, err := client.R().
		SetContext(ctx).
		SetBody(Runner{ID: runnerID}).
		Post("/api/runner")
	if err != nil {
		return "", err
	}
	if resp.IsError() {
		return "", fmt.Errorf("failed to register runner: %d %s", resp.StatusCode(), resp.String())
	}
	return runnerID, nil
}

func pollAndExecute(ctx context.Context, runnerID string) error {
	var job Job
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("X-Runner-ID", runnerID).
		SetResult(&job).
		Get("/api/jobs/acquire")
	if err != nil {
		return err
	}
	if resp.StatusCode() == http.StatusNoContent {
		return nil
	}
	if resp.IsError() {
		return fmt.Errorf("failed to acquire job: %d %s", resp.StatusCode(), resp.String())
	}

	log.Printf("Acquired job %d: %s", job.ID, job.Action)
	return executeJob(ctx, job)
}

func executeJob(ctx context.Context, job Job) error {
	task, ok := tasks.Lookup(job.Action)
	if !ok {
		err := fmt.Errorf("unknown action: %s", job.Action)
		sendFinalLog(ctx, job.ID, fmt.Sprintf("ERROR: %v\n", err))
		return err
	}

	// jobCtx is cancelled when the backend reports the job as cancelled, or on shutdown.
	jobCtx, cancelJob := context.WithCancel(ctx)
	defer cancelJob()
	var cancelledByBackend atomic.Bool

	// Logs are shipped in order by a single sender so the task never blocks on HTTP.
	// It uses ctx rather than jobCtx so logs queued before a cancellation still arrive.
	logsChan := make(chan string, 64)
	senderDone := make(chan struct{})
	go func() {
		defer close(senderDone)
		send := func(logData string) {
			status, err := sendLogs(ctx, job.ID, logData)
			if err != nil {
				log.Printf("Failed to send logs: %v", err)
				return
			}
			if status == jobStatusCancelled && cancelledByBackend.CompareAndSwap(false, true) {
				log.Printf("Job %d cancelled by backend, stopping", job.ID)
				cancelJob()
			}
		}

		// Job-Status only arrives with a log upload, so a task that goes quiet (e.g. a silent
		// Python script) gets an empty upload periodically to still notice a cancellation.
		ticker := time.NewTicker(statusCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case logLine, ok := <-logsChan:
				if !ok {
					return
				}
				send(logLine)
				ticker.Reset(statusCheckInterval)
			case <-ticker.C:
				if !cancelledByBackend.Load() {
					send("")
				}
			}
		}
	}()

	logf := func(format string, args ...any) {
		line := fmt.Sprintf(format, args...)
		if !strings.HasSuffix(line, "\n") {
			line += "\n"
		}
		logsChan <- line
	}

	err := task(jobCtx, tasks.Input{Code: job.Code, Digits: job.Digits, Words: job.Words}, logf)
	close(logsChan)
	<-senderDone

	// Already cancelled on the backend, so record why the task stopped instead of an ERROR.
	if cancelledByBackend.Load() {
		sendFinalLog(ctx, job.ID, "Job cancelled by user, runner stopped the task\n")
		return nil
	}
	// Sent last, after all task logs, because the backend treats it as terminal.
	if err != nil {
		sendFinalLog(ctx, job.ID, fmt.Sprintf("ERROR: %v\n", err))
	}
	return err
}

// sendFinalLog uploads a job's last log line even if ctx is cancelled (e.g. during shutdown).
func sendFinalLog(ctx context.Context, jobID uint64, line string) {
	if _, err := sendLogs(context.WithoutCancel(ctx), jobID, line); err != nil {
		log.Printf("Failed to send logs: %v", err)
	}
}

// sendLogs uploads a log chunk and returns the job status from the Job-Status response header.
func sendLogs(ctx context.Context, jobID uint64, logData string) (string, error) {
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "text/plain").
		SetBody(logData).
		Post(fmt.Sprintf("/api/jobs/%d/logs", jobID))
	if err != nil {
		return "", err
	}
	if resp.IsError() {
		return "", fmt.Errorf("failed to send logs: %d %s", resp.StatusCode(), resp.String())
	}
	return resp.Header().Get("Job-Status"), nil
}
