package models

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

var Actions = []string{"calculate_pi", "lorem_ipsum"}

var ErrNotFound = errors.New("not found")

// Runner is responsible for executing remote task actions.
type Runner struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	LastHeardAt time.Time `json:"last_heard_at"`
}

// Register inserts the runner, or refreshes last_heard_at if it already exists.
func (r *Runner) Register(ctx context.Context, db *sql.DB) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO runners (id) VALUES ($1)
		ON CONFLICT (id) DO UPDATE SET last_heard_at = now()
		RETURNING created_at, last_heard_at`,
		r.ID,
	).Scan(&r.CreatedAt, &r.LastHeardAt)
}

type Job struct {
	ID         uint64     `json:"id"`
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Status     string     `json:"status"`
	RunnerID   string     `json:"runner_id,omitempty"`

	Action string `json:"action"`
	Notes  string `json:"notes,omitempty"`
	Logs   string `json:"logs,omitempty"`
}

const jobColumns = `id, created_at, started_at, finished_at, status, COALESCE(runner_id, ''), action, notes`

func (j *Job) scanFrom(row interface{ Scan(...any) error }) error {
	return row.Scan(&j.ID, &j.CreatedAt, &j.StartedAt, &j.FinishedAt, &j.Status, &j.RunnerID, &j.Action, &j.Notes)
}

// Register inserts a new pending job and fills in the generated fields.
func (j *Job) Register(ctx context.Context, db *sql.DB) error {
	row := db.QueryRowContext(ctx, `
		INSERT INTO jobs (action, notes) VALUES ($1, $2)
		RETURNING `+jobColumns,
		j.Action, j.Notes,
	)
	return j.scanFrom(row)
}

// ListJobs returns all jobs without their logs.
func ListJobs(ctx context.Context, db *sql.DB) ([]Job, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+jobColumns+` FROM jobs ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []Job{}
	for rows.Next() {
		var j Job
		if err := j.scanFrom(rows); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// AcquireJob claims the oldest pending job for the runner and marks it running.
// SKIP LOCKED lets concurrent runners each claim a different job.
// Returns ErrNotFound when there is nothing to acquire.
func AcquireJob(ctx context.Context, db *sql.DB, runnerID string) (*Job, error) {
	var j Job
	err := j.scanFrom(db.QueryRowContext(ctx, `
		UPDATE jobs SET status = 'running', started_at = now(), runner_id = $1
		WHERE id = (
			SELECT id FROM jobs WHERE status = 'pending'
			ORDER BY created_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING `+jobColumns,
		runnerID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// AppendLogs appends data to the job's logs. The runner never reports completion,
// so a running job is finalized when the chunk contains the runner's terminal line.
func (j *Job) AppendLogs(ctx context.Context, db *sql.DB, data string) error {
	res, err := db.ExecContext(ctx, `
		UPDATE jobs SET
			logs = logs || $2,
			status      = CASE WHEN status = 'running' AND $3::text <> '' THEN $3::text ELSE status END,
			finished_at = CASE WHEN status = 'running' AND $3::text <> '' THEN now() ELSE finished_at END
		WHERE id = $1`,
		j.ID, data, terminalStatus(data),
	)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	j.Logs += data
	return nil
}

// terminalStatus recognizes the final log lines the runner emits: "ERROR: ..." on
// failure, and "<action> completed" on success. Returns "" for anything else.
func terminalStatus(data string) string {
	status := ""
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ERROR: ") {
			return StatusFailed
		}
		if strings.HasSuffix(line, " completed") {
			status = StatusCompleted
		}
	}
	return status
}
