CREATE TABLE IF NOT EXISTS runners (
    id            TEXT PRIMARY KEY,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_heard_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS jobs (
    id          BIGSERIAL PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at  TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    status      TEXT NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    action      TEXT NOT NULL CHECK (action IN ('calculate_pi', 'lorem_ipsum')),
    notes       TEXT NOT NULL DEFAULT '',
    logs        TEXT NOT NULL DEFAULT '',
    runner_id   TEXT REFERENCES runners (id)
);

CREATE INDEX IF NOT EXISTS jobs_pending_idx ON jobs (created_at, id) WHERE status = 'pending';
