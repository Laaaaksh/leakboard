CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token      TEXT PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE connections (
    id            BIGSERIAL PRIMARY KEY,
    name          TEXT NOT NULL,
    github_org    TEXT NOT NULL,
    access_token  TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE repos (
    id                  BIGSERIAL PRIMARY KEY,
    connection_id       BIGINT REFERENCES connections(id) ON DELETE SET NULL,
    name                TEXT NOT NULL,
    clone_url           TEXT NOT NULL,
    default_branch      TEXT NOT NULL DEFAULT 'main',
    mirror_path         TEXT NOT NULL,
    scanned_ref_tips    JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_scanned_at     TIMESTAMPTZ,
    last_scan_error     TEXT NOT NULL DEFAULT '',
    scan_interval_secs  INTEGER NOT NULL DEFAULT 300,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (clone_url)
);

CREATE TABLE scan_runs (
    id                BIGSERIAL PRIMARY KEY,
    repo_id           BIGINT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    started_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at       TIMESTAMPTZ,
    status            TEXT NOT NULL DEFAULT 'running',
    new_findings      INTEGER NOT NULL DEFAULT 0,
    error             TEXT
);

CREATE TABLE findings (
    id             BIGSERIAL PRIMARY KEY,
    repo_id        BIGINT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    fingerprint    TEXT NOT NULL,
    rule_id        TEXT NOT NULL,
    description    TEXT NOT NULL DEFAULT '',
    file_path      TEXT NOT NULL,
    start_line     INTEGER NOT NULL,
    end_line       INTEGER NOT NULL,
    commit_sha     TEXT NOT NULL,
    commit_author  TEXT NOT NULL DEFAULT '',
    commit_email   TEXT NOT NULL DEFAULT '',
    commit_date    TIMESTAMPTZ,
    secret         TEXT NOT NULL,
    match          TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'new',
    first_seen_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at    TIMESTAMPTZ,
    UNIQUE (repo_id, fingerprint)
);

CREATE INDEX findings_repo_status_idx ON findings (repo_id, status);
CREATE INDEX findings_status_idx ON findings (status);

CREATE TABLE allowlist_entries (
    id           BIGSERIAL PRIMARY KEY,
    rule_id      TEXT,
    path_pattern TEXT,
    regex        TEXT,
    fingerprint  TEXT,
    reason       TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE webhooks (
    id         BIGSERIAL PRIMARY KEY,
    kind       TEXT NOT NULL,
    target_url TEXT NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
