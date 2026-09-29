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

Redis (the `redis` compose service) is required for sessions: `docker compose up postgres redis`.

| Variable       | Default                | Purpose                                                        |
| -------------- | ---------------------- | -------------------------------------------------------------- |
| `REDIS_ADDR`   | `localhost:6379`       | Session store                                                  |
| `RUNNER_TOKEN` | `dev-runner-token`     | Shared secret runners send as `X-Runner-Token` (set in prod)   |
| `ADMIN_EMAIL`  | _(none)_               | The account registered with this email becomes an admin        |

Integration tests run against a throwaway Postgres schema and need Postgres and Redis running:
`TEST_DATABASE_URL='postgres://jobsuser:jobspass@localhost:5432/jobsdb?sslmode=disable' go test ./...`

## Authentication

- **Accounts:** `POST /api/auth/register` and `POST /api/auth/login` take `{"email", "password"}`
  (password 8–72 bytes, bcrypt-hashed) and start a session; `POST /api/auth/logout` ends it and
  `GET /api/auth/me` returns the current user.
- **Sessions:** a random 256-bit token in an `HttpOnly`, `SameSite=Lax` cookie. Redis stores only the
  token's SHA-256 hash, mapped to the user ID, with a 7-day TTL. The user row is reloaded on each
  request, so admin changes take effect immediately.
- **Roles:** every user has an `is_admin` flag. There is no active job limit.
  - admin: can cancel any user's job, and can change any user's admin flag via
    `GET /api/admin/users` and `PATCH /api/admin/users/{id}` with `{"is_admin": bool}`.
    Admins cannot remove their own admin flag.
- **Job visibility:** every user can list and fetch every job (each includes `user_id` and
  `owner_email`), but non-admins can only cancel their own; cancelling someone else's job returns
  `403`. Jobs created before accounts existed have no owner, so only admins can cancel them.
- **Runners:** `/api/runner`, `/api/jobs/acquire`, and `/api/jobs/{id}/logs` require the shared
  `X-Runner-Token` header instead of a user session.
- **Not implemented:** payments, email verification, password reset,
  and login rate limiting.

## Job lifecycle

`pending` → `running` → `completed` | `failed`, and `pending` | `running` → `cancelled`

- **Acquire:** the oldest pending job is claimed with `UPDATE ... WHERE id = (SELECT ... FOR UPDATE SKIP LOCKED)`,
  so multiple runners polling concurrently never receive the same job.
- **Completion:** the runner never reports a result, so the backend infers it from the logs it streams.
  A log line starting with `ERROR: ` (sent by the runner when an action returns an error) marks the job
  `failed`; a line ending in ` completed` (the last line of each successful action) marks it `completed`.
  Only `running` jobs transition, so a late log chunk cannot overwrite a terminal status.
  Lines starting with `|` are user program output (from `run_python`) and are never terminal.
- **Python jobs:** `run_python` requires a non-empty `code` field (max 64 KiB); other actions
  reject `code`. It is returned by acquire and executed by the runner.
- **Job sizes:** `calculate_pi` accepts an optional `digits` (1-10000, default 1000) and
  `lorem_ipsum` an optional `words` (1-5000, default 200); other actions reject them. The
  stored value is returned by list/get/acquire.
- **Cancellation:** `POST /api/jobs/{id}/cancel` marks a pending or running job `cancelled` (409 for jobs
  that already finished). Every log upload response carries a `Job-Status` header; when it reads
  `cancelled` the runner stops the task, prints it to its console, and appends a final log line.
  While a task is silent the runner sends an empty log upload every 2 seconds, so it still sees the header.
  A pending job that is cancelled is never handed to a runner.
- **Limitation:** a runner that crashes mid-job leaves the job `running`. A heartbeat timeout using
  `runners.last_heard_at` (refreshed on every poll) would be the natural next step.

## Listing jobs

`GET /api/jobs` returns the 20 newest jobs; pass `?before=<next_before>` for each following page:

```json
{ "jobs": [ ... ], "has_more": true, "next_before": 38, "total": 57 }
```

Paging uses a keyset cursor (`id < before`) rather than `OFFSET`, so pages stay consistent while new
jobs are created: the frontend's infinite scroll never sees duplicated or skipped rows.

`GET /api/jobs?ids=4,9,12` (up to 100 ids, not combinable with `before`) returns `{ "jobs": [ ... ] }`
for just those jobs, skipping unknown ids. The frontend uses it to refresh still-active rows that have
scrolled past the newest page.

## Errors

JSON endpoints return `{"error": "<message>"}` with an appropriate status code (400 for validation,
404 for unknown jobs, 500 for database failures — details are logged, not returned). The logs endpoint
responds in `text/plain`.
