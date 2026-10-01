CREATE TABLE sessions (
    id          INTEGER PRIMARY KEY,
    tag         TEXT    NOT NULL DEFAULT 'work',
    started_at  TEXT    NOT NULL,
    stopped_at  TEXT,
    note        TEXT,
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%S', 'now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%S', 'now'))
);

CREATE INDEX idx_sessions_started_at ON sessions (started_at);

CREATE TRIGGER sessions_updated_at
AFTER UPDATE ON sessions
FOR EACH ROW
BEGIN
    UPDATE sessions SET updated_at = strftime('%Y-%m-%dT%H:%M:%S', 'now')
    WHERE id = OLD.id;
END;
