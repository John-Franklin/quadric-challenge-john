# Runner

Basic job runner that polls the backend for jobs and executes them.

## Supported Actions

- `calculate_pi` - Calculates PI to the job's `digits` decimal places using Machin's formula
- `lorem_ipsum` - Generates the job's `words` count of Lorem Ipsum text, one paragraph per log line
- `run_python` - Runs the job's `code` with `python3` (must be on `PATH`), streaming stdout/stderr
  into the job logs. Output lines are prefixed with `| ` so the backend never mistakes script
  output for a terminal `ERROR: ` / ` completed` line. Scripts are killed (with any child
  processes) on cancellation or after 5 minutes. **The code is not sandboxed** and runs with the
  runner's permissions, so only run this runner where every job author is trusted.

## Running

```bash
cd runner
go run main.go
```

Or build and run:

```bash
cd runner
go build -o runner
./runner
```

## How it works

1. Registers itself with the backend
2. Polls `/api/jobs/acquire` every 5 seconds
3. Executes acquired jobs
4. Streams logs back to `/api/jobs/{id}/logs`
5. Handles graceful shutdown on SIGINT/SIGTERM
