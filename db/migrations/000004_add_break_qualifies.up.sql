ALTER TABLE sessions ADD COLUMN break_qualifies INTEGER NOT NULL DEFAULT 0 CHECK (break_qualifies IN (0, 1));
