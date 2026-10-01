package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/astepanovs/ztime/internal"
)

const timeLayout = "2006-01-02T15:04:05"

// SQLiteStore implements internal.Store backed by SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewStore wraps an open *sql.DB as a Store.
func NewStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Create(tag string, startedAt time.Time, note string) (*internal.Entry, error) {
	res, err := s.db.Exec(
		`INSERT INTO sessions (tag, started_at, note) VALUES (?, ?, ?)`,
		tag, startedAt.Format(timeLayout), note,
	)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create session last id: %w", err)
	}
	return s.Get(id)
}

func (s *SQLiteStore) CreateWithActivity(tag string, startedAt time.Time, text string) (*internal.Entry, *internal.Activity, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, fmt.Errorf("begin start with task: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.Exec(
		`INSERT INTO sessions (tag, started_at, note) VALUES (?, ?, '')`,
		tag, startedAt.Format(timeLayout),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create entry with task: %w", err)
	}
	entryID, err := result.LastInsertId()
	if err != nil {
		return nil, nil, fmt.Errorf("create entry with task last id: %w", err)
	}
	result, err = tx.Exec(
		`INSERT INTO activities (entry_id, occurred_at, text) VALUES (?, ?, ?)`,
		entryID, startedAt.Format(timeLayout), text,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create initial task: %w", err)
	}
	activityID, err := result.LastInsertId()
	if err != nil {
		return nil, nil, fmt.Errorf("create initial task last id: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit start with task: %w", err)
	}

	entry, err := s.Get(entryID)
	if err != nil {
		return nil, nil, err
	}
	activity, err := s.getActivity(activityID)
	if err != nil {
		return nil, nil, err
	}
	return entry, activity, nil
}

func (s *SQLiteStore) Stop(tag string, stoppedAt time.Time, reason string, breakQualifies bool) (*internal.Entry, error) {
	open, err := s.OpenEntry(tag)
	if err != nil {
		return nil, err
	}
	if open == nil {
		return nil, internal.ErrNoOpenEntry
	}

	// A note supplied when stopping describes the break that follows.
	if reason != "" {
		_, err = s.db.Exec(
			`UPDATE sessions SET stopped_at = ?, break_note = ?, break_qualifies = ? WHERE id = ?`,
			stoppedAt.Format(timeLayout), reason, breakQualifies, open.ID,
		)
	} else {
		_, err = s.db.Exec(
			`UPDATE sessions SET stopped_at = ?, break_qualifies = ? WHERE id = ?`,
			stoppedAt.Format(timeLayout), breakQualifies, open.ID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("stop session: %w", err)
	}
	return s.Get(open.ID)
}

func (s *SQLiteStore) Get(id int64) (*internal.Entry, error) {
	row := s.db.QueryRow(
		`SELECT id, tag, started_at, stopped_at, note, break_note, break_qualifies, created_at, updated_at
		 FROM sessions WHERE id = ?`, id,
	)
	entry, err := scanEntry(row)
	if err == sql.ErrNoRows {
		return nil, internal.ErrEntryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get session %d: %w", id, err)
	}
	return entry, nil
}

func (s *SQLiteStore) List(f internal.LogFilter) ([]internal.Entry, error) {
	where, args := buildWhere(f)
	q := `SELECT id, tag, started_at, stopped_at, note, break_note, break_qualifies, created_at, updated_at
	      FROM sessions` + where + ` ORDER BY started_at ASC`

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var entries []internal.Entry
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		entries = append(entries, *entry)
	}
	return entries, rows.Err()
}

func (s *SQLiteStore) OpenEntry(tag string) (*internal.Entry, error) {
	row := s.db.QueryRow(
		`SELECT id, tag, started_at, stopped_at, note, break_note, break_qualifies, created_at, updated_at
		 FROM sessions WHERE tag = ? AND stopped_at IS NULL
		 ORDER BY started_at DESC LIMIT 1`, tag,
	)
	entry, err := scanEntry(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open entry query: %w", err)
	}
	return entry, nil
}

func (s *SQLiteStore) Edit(id int64, p internal.EditParams) (*internal.Entry, error) {
	// Build SET clause from only the non-nil fields.
	var setClauses []string
	var args []any

	if p.Tag != nil {
		setClauses = append(setClauses, "tag = ?")
		args = append(args, *p.Tag)
	}
	if p.StartedAt != nil {
		setClauses = append(setClauses, "started_at = ?")
		args = append(args, p.StartedAt.Format(timeLayout))
	}
	if p.StoppedAt != nil {
		setClauses = append(setClauses, "stopped_at = ?")
		args = append(args, p.StoppedAt.Format(timeLayout))
	}
	if p.Note != nil {
		setClauses = append(setClauses, "note = ?")
		args = append(args, *p.Note)
	}
	if p.BreakNote != nil {
		setClauses = append(setClauses, "break_note = ?")
		args = append(args, *p.BreakNote)
	}
	if p.BreakQualifies != nil {
		setClauses = append(setClauses, "break_qualifies = ?")
		args = append(args, *p.BreakQualifies)
	}
	if len(setClauses) == 0 {
		return s.Get(id)
	}

	args = append(args, id)
	q := `UPDATE sessions SET ` + strings.Join(setClauses, ", ") + ` WHERE id = ?`
	res, err := s.db.Exec(q, args...)
	if err != nil {
		return nil, fmt.Errorf("edit session %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, internal.ErrEntryNotFound
	}
	return s.Get(id)
}

func (s *SQLiteStore) Delete(id int64) error {
	res, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete session %d: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return internal.ErrEntryNotFound
	}
	return nil
}

// scanner abstracts *sql.Row and *sql.Rows so scanSession works for both.
type scanner interface {
	Scan(dest ...any) error
}

func scanEntry(s scanner) (*internal.Entry, error) {
	var (
		entry     internal.Entry
		startedAt string
		stoppedAt sql.NullString
		note      sql.NullString
		breakNote sql.NullString
		breakQualifies bool
		createdAt string
		updatedAt string
	)
	err := s.Scan(&entry.ID, &entry.Tag, &startedAt, &stoppedAt, &note, &breakNote, &breakQualifies, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	entry.StartedAt, err = time.ParseInLocation(timeLayout, startedAt, time.Local)
	if err != nil {
		return nil, fmt.Errorf("parse started_at %q: %w", startedAt, err)
	}
	if stoppedAt.Valid {
		t, err := time.ParseInLocation(timeLayout, stoppedAt.String, time.Local)
		if err != nil {
			return nil, fmt.Errorf("parse stopped_at %q: %w", stoppedAt.String, err)
		}
		entry.StoppedAt = &t
	}
	if note.Valid {
		entry.Note = note.String
	}
	if breakNote.Valid {
		entry.BreakNote = breakNote.String
	}
	entry.BreakQualifies = breakQualifies
	entry.CreatedAt, _ = time.ParseInLocation(timeLayout, createdAt, time.Local)
	entry.UpdatedAt, _ = time.ParseInLocation(timeLayout, updatedAt, time.Local)
	return &entry, nil
}

func buildWhere(f internal.LogFilter) (string, []any) {
	var clauses []string
	var args []any

	if f.Tag != "" {
		clauses = append(clauses, "tag = ?")
		args = append(args, f.Tag)
	}
	if f.Date != nil {
		day := f.Date.Format("2006-01-02")
		clauses = append(clauses, "started_at >= ? AND started_at < ?")
		args = append(args, day+"T00:00:00", day+"T23:59:59")
	} else {
		if f.From != nil {
			clauses = append(clauses, "started_at >= ?")
			args = append(args, f.From.Format(timeLayout))
		}
		if f.To != nil {
			clauses = append(clauses, "started_at <= ?")
			args = append(args, f.To.Format(timeLayout))
		}
	}

	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (s *SQLiteStore) LastStopped(tag string) (*internal.Entry, error) {
	row := s.db.QueryRow(
		`SELECT id, tag, started_at, stopped_at, note, break_note, break_qualifies, created_at, updated_at
		 FROM sessions WHERE tag = ? AND stopped_at IS NOT NULL
		 ORDER BY stopped_at DESC LIMIT 1`, tag,
	)
	entry, err := scanEntry(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("last stopped entry query: %w", err)
	}
	return entry, nil
}

func (s *SQLiteStore) UndoLast(tag string) (*internal.UndoResult, error) {
	// Find the most recent session for this tag regardless of state.
	row := s.db.QueryRow(
		`SELECT id, tag, started_at, stopped_at, note, break_note, break_qualifies, created_at, updated_at
		 FROM sessions WHERE tag = ?
		 ORDER BY started_at DESC LIMIT 1`, tag,
	)
	entry, err := scanEntry(row)
	if err == sql.ErrNoRows {
		return nil, internal.ErrEntryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("undo last entry query: %w", err)
	}

	if entry.IsOpen() {
		// Last action was a start -- delete the session.
		if err := s.Delete(entry.ID); err != nil {
			return nil, err
		}
		return &internal.UndoResult{Action: "deleted", Entry: entry}, nil
	}

	// Last action was a stop -- reopen by clearing stopped_at.
	_, err = s.db.Exec(`UPDATE sessions SET stopped_at = NULL WHERE id = ?`, entry.ID)
	if err != nil {
		return nil, fmt.Errorf("undo stop entry: %w", err)
	}
	entry.StoppedAt = nil
	return &internal.UndoResult{Action: "reopened", Entry: entry}, nil
}

func (s *SQLiteStore) EntryAt(tag string, at time.Time) (*internal.Entry, error) {
	row := s.db.QueryRow(
		`SELECT id, tag, started_at, stopped_at, note, break_note, break_qualifies, created_at, updated_at
		 FROM sessions
		 WHERE tag = ? AND started_at <= ? AND (stopped_at IS NULL OR stopped_at > ?)
		 ORDER BY started_at DESC LIMIT 1`,
		tag, at.Format(timeLayout), at.Format(timeLayout),
	)
	entry, err := scanEntry(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("entry at time query: %w", err)
	}
	return entry, nil
}

func (s *SQLiteStore) LastEntryBefore(tag string, at time.Time) (*internal.Entry, error) {
	row := s.db.QueryRow(
		`SELECT id, tag, started_at, stopped_at, note, break_note, break_qualifies, created_at, updated_at
		 FROM sessions
		 WHERE tag = ? AND stopped_at IS NOT NULL AND stopped_at <= ?
		 ORDER BY stopped_at DESC LIMIT 1`,
		tag, at.Format(timeLayout),
	)
	entry, err := scanEntry(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("last entry before time query: %w", err)
	}
	return entry, nil
}

func (s *SQLiteStore) CreateActivity(entryID int64, occurredAt time.Time, text string) (*internal.Activity, error) {
	result, err := s.db.Exec(
		`INSERT INTO activities (entry_id, occurred_at, text) VALUES (?, ?, ?)`,
		entryID, occurredAt.Format(timeLayout), text,
	)
	if err != nil {
		return nil, fmt.Errorf("create activity: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create activity last id: %w", err)
	}
	return s.getActivity(id)
}

func (s *SQLiteStore) ListActivities(filter internal.ActivityFilter) ([]internal.Activity, error) {
	where, args := buildActivityWhere(filter)
	rows, err := s.db.Query(
		`SELECT id, entry_id, occurred_at, text, created_at, updated_at
		 FROM activities`+where+` ORDER BY occurred_at ASC, id ASC`, args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list activities: %w", err)
	}
	defer rows.Close()

	var activities []internal.Activity
	for rows.Next() {
		activity, err := scanActivity(rows)
		if err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		activities = append(activities, *activity)
	}
	return activities, rows.Err()
}

func (s *SQLiteStore) UndoLastActivity() (*internal.Activity, error) {
	row := s.db.QueryRow(
		`SELECT id, entry_id, occurred_at, text, created_at, updated_at
		 FROM activities ORDER BY created_at DESC, id DESC LIMIT 1`,
	)
	activity, err := scanActivity(row)
	if err == sql.ErrNoRows {
		return nil, internal.ErrActivityNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("last activity query: %w", err)
	}
	if _, err := s.db.Exec(`DELETE FROM activities WHERE id = ?`, activity.ID); err != nil {
		return nil, fmt.Errorf("undo activity: %w", err)
	}
	return activity, nil
}

func (s *SQLiteStore) getActivity(id int64) (*internal.Activity, error) {
	row := s.db.QueryRow(
		`SELECT id, entry_id, occurred_at, text, created_at, updated_at
		 FROM activities WHERE id = ?`, id,
	)
	activity, err := scanActivity(row)
	if err == sql.ErrNoRows {
		return nil, internal.ErrActivityNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get activity %d: %w", id, err)
	}
	return activity, nil
}

func scanActivity(s scanner) (*internal.Activity, error) {
	var activity internal.Activity
	var occurredAt, createdAt, updatedAt string
	if err := s.Scan(
		&activity.ID, &activity.EntryID, &occurredAt, &activity.Text, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	var err error
	activity.OccurredAt, err = time.ParseInLocation(timeLayout, occurredAt, time.Local)
	if err != nil {
		return nil, fmt.Errorf("parse occurred_at %q: %w", occurredAt, err)
	}
	activity.CreatedAt, _ = time.ParseInLocation(timeLayout, createdAt, time.Local)
	activity.UpdatedAt, _ = time.ParseInLocation(timeLayout, updatedAt, time.Local)
	return &activity, nil
}

func buildActivityWhere(filter internal.ActivityFilter) (string, []any) {
	var clauses []string
	var args []any
	if filter.Date != nil {
		day := filter.Date.Format("2006-01-02")
		clauses = append(clauses, "occurred_at >= ? AND occurred_at < ?")
		args = append(args, day+"T00:00:00", day+"T23:59:59")
	} else {
		if filter.From != nil {
			clauses = append(clauses, "occurred_at >= ?")
			args = append(args, filter.From.Format(timeLayout))
		}
		if filter.To != nil {
			clauses = append(clauses, "occurred_at <= ?")
			args = append(args, filter.To.Format(timeLayout))
		}
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
