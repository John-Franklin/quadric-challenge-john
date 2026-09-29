// Package models reads and writes the application's PostgreSQL tables.
package models

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// Terminal job statuses derived from runner logs; the other statuses are only set in SQL.
const (
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// Job actions supported by the runner.
const (
	ActionCalculatePi = "calculate_pi"
	ActionLoremIpsum  = "lorem_ipsum"
	ActionRunPython   = "run_python"
)

// Actions lists every valid job action.
var Actions = []string{ActionCalculatePi, ActionLoremIpsum, ActionRunPython}

// outputPrefix marks lines of user program output (see runner/tasks/python.go); they are
// never treated as terminal, so a script cannot finish its own job by printing a status line.
const outputPrefix = "|"

// Errors returned by the model functions.
var (
	ErrNotFound       = errors.New("not found")
	ErrNotCancellable = errors.New("job is not pending or running")
	ErrForbidden      = errors.New("forbidden")
)

type scanner interface {
	Scan(dest ...any) error
}

// notFound maps sql.ErrNoRows to ErrNotFound and returns other errors unchanged.
func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

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

// Job is a unit of work executed by a runner.
type Job struct {
	ID         uint64     `json:"id"`
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Status     string     `json:"status"`
	RunnerID   string     `json:"runner_id,omitempty"`
	UserID     *uint64    `json:"user_id,omitempty"`
	// OwnerEmail is only filled in by ListJobs and GetJob.
	OwnerEmail string `json:"owner_email,omitempty"`

	Action string `json:"action"`
	Notes  string `json:"notes,omitempty"`
	Code   string `json:"code,omitempty"`
	Digits *int   `json:"digits,omitempty"`
	Words  *int   `json:"words,omitempty"`
	Logs   string `json:"logs,omitempty"`
}

// Qualified so queries can join users without ambiguity.
const jobColumns = `jobs.id, jobs.created_at, jobs.started_at, jobs.finished_at, jobs.status,
	COALESCE(jobs.runner_id, ''), jobs.user_id, jobs.action, jobs.notes, jobs.code,
	jobs.digits, jobs.words`

func (j *Job) scanFrom(row scanner, extra ...any) error {
	return row.Scan(append([]any{
		&j.ID, &j.CreatedAt, &j.StartedAt, &j.FinishedAt, &j.Status, &j.RunnerID, &j.UserID, &j.Action, &j.Notes, &j.Code,
		&j.Digits, &j.Words,
	}, extra...)...)
}

// Register inserts a new pending job owned by owner and fills in the generated fields.
func (j *Job) Register(ctx context.Context, db *sql.DB, owner *User) error {
	if err := j.scanFrom(db.QueryRowContext(ctx, `
		INSERT INTO jobs (action, notes, code, digits, words, user_id) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+jobColumns,
		j.Action, j.Notes, j.Code, j.Digits, j.Words, owner.ID,
	)); err != nil {
		return err
	}
	j.OwnerEmail = owner.Email
	return nil
}

// GetJob returns a single job including its logs.
func GetJob(ctx context.Context, db *sql.DB, id uint64) (*Job, error) {
	var j Job
	err := j.scanFrom(db.QueryRowContext(ctx, `
		SELECT `+jobColumns+`, COALESCE(users.email, ''), jobs.logs
		FROM jobs LEFT JOIN users ON users.id = jobs.user_id
		WHERE jobs.id = $1`,
		id,
	), &j.OwnerEmail, &j.Logs)
	if err != nil {
		return nil, notFound(err)
	}
	return &j, nil
}

// JobsPageSize is the number of jobs per ListJobs page.
const JobsPageSize = 20

// MaxJobsByIDs caps how many jobs ListJobsByIDs returns in one call.
const MaxJobsByIDs = 100

// ListJobsByIDs returns the requested jobs, newest first, without their logs.
// Unknown ids are skipped.
func ListJobsByIDs(ctx context.Context, db *sql.DB, ids []int64) ([]Job, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT `+jobColumns+`, COALESCE(users.email, '')
		FROM jobs LEFT JOIN users ON users.id = jobs.user_id
		WHERE jobs.id = ANY($1)
		ORDER BY jobs.id DESC`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []Job{}
	for rows.Next() {
		var j Job
		if err := j.scanFrom(rows, &j.OwnerEmail); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// JobPage is one page of ListJobs results.
type JobPage struct {
	Jobs    []Job `json:"jobs"`
	HasMore bool  `json:"has_more"`
	// NextBefore is the cursor for the following page; set only when HasMore.
	NextBefore uint64 `json:"next_before,omitempty"`
	Total      int    `json:"total"`
}

// ListJobs returns up to JobsPageSize jobs, newest first, without their logs.
// With before > 0 only jobs with a smaller id are returned.
func ListJobs(ctx context.Context, db *sql.DB, before uint64) (*JobPage, error) {
	result := &JobPage{Jobs: []Job{}}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM jobs`).Scan(&result.Total); err != nil {
		return nil, err
	}

	// Keyset paging (id < before) keeps pages stable while new jobs are inserted at the
	// top, unlike OFFSET. One extra row tells us whether another page exists.
	rows, err := db.QueryContext(ctx, `
		SELECT `+jobColumns+`, COALESCE(users.email, '')
		FROM jobs LEFT JOIN users ON users.id = jobs.user_id
		WHERE $1::bigint = 0 OR jobs.id < $1::bigint
		ORDER BY jobs.id DESC LIMIT $2`,
		before, JobsPageSize+1,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var j Job
		if err := j.scanFrom(rows, &j.OwnerEmail); err != nil {
			return nil, err
		}
		result.Jobs = append(result.Jobs, j)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(result.Jobs) > JobsPageSize {
		result.Jobs = result.Jobs[:JobsPageSize]
		result.HasMore = true
		result.NextBefore = result.Jobs[JobsPageSize-1].ID
	}
	return result, nil
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
	if err != nil {
		return nil, notFound(err)
	}
	return &j, nil
}

// AppendLogs appends data to the job's logs and sets j.Status to the job's status
// afterwards. The runner never reports completion, so a running job is finalized
// when the chunk contains the runner's terminal line.
func (j *Job) AppendLogs(ctx context.Context, db *sql.DB, data string) error {
	err := db.QueryRowContext(ctx, `
		UPDATE jobs SET
			logs = logs || $2,
			status      = CASE WHEN status = 'running' AND $3::text <> '' THEN $3::text ELSE status END,
			finished_at = CASE WHEN status = 'running' AND $3::text <> '' THEN now() ELSE finished_at END
		WHERE id = $1
		RETURNING status`,
		j.ID, data, terminalStatus(data),
	).Scan(&j.Status)
	return notFound(err)
}

// CancelJob marks a pending or running job cancelled. A running job's runner is
// told on its next log upload via the Job-Status response header.
// A nil owner may cancel any job; otherwise other users' jobs are reported as ErrForbidden.
func CancelJob(ctx context.Context, db *sql.DB, id uint64, owner *uint64) (*Job, error) {
	var j Job
	err := j.scanFrom(db.QueryRowContext(ctx, `
		UPDATE jobs SET status = 'cancelled', finished_at = now()
		WHERE id = $1 AND status IN ('pending', 'running') AND ($2::bigint IS NULL OR user_id = $2)
		RETURNING `+jobColumns,
		id, owner,
	))
	if err == nil {
		return &j, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	var userID *uint64
	if err := db.QueryRowContext(ctx, `SELECT user_id FROM jobs WHERE id = $1`, id).Scan(&userID); err != nil {
		return nil, notFound(err)
	}
	if owner != nil && (userID == nil || *userID != *owner) {
		return nil, ErrForbidden
	}
	return nil, ErrNotCancellable
}

// terminalStatus recognizes the final log lines the runner emits: "ERROR: ..." on
// failure, and "<action> completed" on success. Returns "" for anything else.
func terminalStatus(data string) string {
	status := ""
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, outputPrefix) {
			continue
		}
		if strings.HasPrefix(line, "ERROR: ") {
			return StatusFailed
		}
		if strings.HasSuffix(line, " completed") {
			status = StatusCompleted
		}
	}
	return status
}
