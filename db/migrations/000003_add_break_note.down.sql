UPDATE sessions
SET note = COALESCE(note, break_note)
WHERE break_note IS NOT NULL;

ALTER TABLE sessions DROP COLUMN break_note;
