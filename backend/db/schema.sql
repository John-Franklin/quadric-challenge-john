CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    is_admin      BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE users DROP COLUMN IF EXISTS tier;

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
    status      TEXT NOT NULL DEFAULT 'pending',
    action      TEXT NOT NULL,
    notes       TEXT NOT NULL DEFAULT '',
    code        TEXT NOT NULL DEFAULT '',
    logs        TEXT NOT NULL DEFAULT '',
    runner_id   TEXT REFERENCES runners (id)
);

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS code TEXT NOT NULL DEFAULT '';
-- Nullable so jobs created before accounts existed survive; only admins can see those.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users (id);
-- Per-action size parameters; NULL for other actions and for jobs created before they existed.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS digits INTEGER CHECK (digits > 0);
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS words INTEGER CHECK (words > 0);

-- Re-created on every startup so databases made with an older action/status list pick up new values.
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_action_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_action_check
    CHECK (action IN ('calculate_pi', 'lorem_ipsum', 'run_python'));

ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_status_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_status_check
    CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled'));

CREATE INDEX IF NOT EXISTS jobs_pending_idx ON jobs (created_at, id) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS jobs_user_idx ON jobs (user_id, id);
