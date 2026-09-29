# Full-Stack Challenge: Remote Job System

A remote job system with a Go backend, a React (Vite + MUI) frontend, and a Go runner that executes jobs.

## 1. Setup Instructions

### Prerequisites

- Go 1.26+ (backend) and Go 1.25+ (runner)
- Node.js 20+ and npm
- Docker or Podman (for Postgres and Redis/DragonflyDB)
- `python3` on `PATH` (for the runner's `run_python` jobs)

### External services

From the repo root:

```bash
docker compose up postgres redis
```

### Backend

```bash
cd backend
go mod tidy
go run main.go        # listens on :8080
```

The schema is applied automatically on startup. Optional environment variables:

| Variable       | Default            | Purpose                                                  |
| -------------- | ------------------ | -------------------------------------------------------- |
| `DATABASE_URL` | local compose DB   | Postgres connection string                               |
| `REDIS_ADDR`   | `localhost:6379`   | Sessions and live job updates                            |
| `RUNNER_TOKEN` | `dev-runner-token` | Shared secret runners send as `X-Runner-Token`           |
| `ADMIN_EMAIL`  | _(none)_           | The account registered with this email becomes an admin  |

Tests (require Postgres and Redis running):

```bash
TEST_DATABASE_URL='postgres://jobsuser:jobspass@localhost:5432/jobsdb?sslmode=disable' go test ./...
```

### Frontend

```bash
cd frontend
npm install
npm run dev           # http://localhost:3000, proxies /api to :8080
```

### Runner

```bash
cd runner
go run main.go
```

See `backend/README.md` and `runner/README.md` for API, auth, and job lifecycle details.

## 2. Time Spent

4 hours.

## 3. AI Usage

AI was used to accelerate development across the project. All code has been reviewed and can be explained and justified during the review call.

## 4. Implementation Notes

### Assumptions

- Current usage thresholds do not require a message queue or multiple runners.
- Current user usage isn't sufficient that polling on the jobs page will cause performance issues.

### Optional Features Implemented

- All optional features listed in the requirements.
- Python code runner (`run_python` action).
- Live updates of jobs and the logs in the list.
- Infinite scroll for the jobs list.
- User authentication.

### Trade-offs and Decisions

- **Manual Pi implementation:** Pi is calculated with a manual implementation (Machin's formula) rather than a library call so that the computation can be cancelled mid-run.
- **No direct caching:** With the periodic page refreshes so frequently caching isn't practical and would be more trouble than it's worth.
- **Live loading and infinite scroll multiloading:** implementing both live updates for jobs and infinite scroll means that many pages are loaded periodically to handle that behavior.

## 5. Next Steps

- **Message queue:** Introduce message queue handling to distribute jobs, allowing the system to scale horizontally across multiple runners.
- **Custom jobs:** Allow users to define custom job types built on a Python interface.
- **Web API invocation:** Expose a web API so jobs can be triggered by external calls, providing Lambda-like behavior.
- **Payments:** Integrate payments to unlock an expanded tier of the Python service.
- **Python imports and deployment:** Support third-party Python package imports and a deployment workflow for user code.
- **AWS deployment:** Deploy the full system to AWS.
- **Paid and unpaid tiers**
- **Websocket Notifications:** Use Websockets to pass through job updates to the job list view and provide notifications when your job has completed, hopefully reducing DB hits a bit.
- **Sandbox for the python call:** This is fine for local runs but the python call needs to be sandboxed or deprivileged so that there can't be any security vulnerabilities from executing from a privileged context.
- **Code syntax highlighting for the python box**
