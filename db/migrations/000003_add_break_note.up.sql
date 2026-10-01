ALTER TABLE sessions ADD COLUMN break_note TEXT;

-- Before this column existed, `stop --note` stored the break reason in note.
-- Preserve those existing reasons and free note for entry-specific context.
UPDATE sessions
SET break_note = note,
    note = NULL
WHERE stopped_at IS NOT NULL
  AND note IS NOT NULL
  AND trim(note) <> '';
