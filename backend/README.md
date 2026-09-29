# Backend

Go API for jobs and runners, backed by PostgreSQL via `pgx` (`database/sql` driver).

## Running

```bash
docker compose up postgres   # from the repo root
cd backend
go mod tidy
go run main.go
```

Set `DATABASE_URL` to override the default connection string. The schema in `db/schema.sql`
is applied on startup (`CREATE TABLE IF NOT EXISTS`), so no separate migration step is needed.

## Job lifecycle

`pending` → `running` → `completed` | `failed`

- **Acquire:** the oldest pending job is claimed with `UPDATE ... WHERE id = (SELECT ... FOR UPDATE SKIP LOCKED)`,
  so multiple runners polling concurrently never receive the same job.
- **Completion:** the runner never reports a result, so the backend infers it from the logs it streams.
  A log line starting with `ERROR: ` (sent by the runner when an action returns an error) marks the job
  `failed`; a line ending in ` completed` (the last line of each successful action) marks it `completed`.
  Only `running` jobs transition, so a late log chunk cannot overwrite a terminal status.
- **Limitation:** a runner that crashes mid-job leaves the job `running`. A heartbeat timeout using
  `runners.last_heard_at` (refreshed on every poll) would be the natural next step.

## Errors

JSON endpoints return `{"error": "<message>"}` with an appropriate status code (400 for validation,
404 for unknown jobs, 500 for database failures — details are logged, not returned). The logs endpoint
responds in `text/plain`.
