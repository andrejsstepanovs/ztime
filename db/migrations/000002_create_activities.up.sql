CREATE TABLE activities (
    id          INTEGER PRIMARY KEY,
    entry_id    INTEGER NOT NULL,
    occurred_at TEXT    NOT NULL,
    text        TEXT    NOT NULL CHECK (length(trim(text)) > 0),
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%S', 'now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%S', 'now')),
    FOREIGN KEY (entry_id) REFERENCES sessions (id) ON DELETE CASCADE
);

CREATE INDEX idx_activities_occurred_at ON activities (occurred_at);
CREATE INDEX idx_activities_entry_id ON activities (entry_id);

CREATE TRIGGER activities_updated_at
AFTER UPDATE ON activities
FOR EACH ROW
BEGIN
    UPDATE activities SET updated_at = strftime('%Y-%m-%dT%H:%M:%S', 'now')
    WHERE id = OLD.id;
END;
