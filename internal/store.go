package internal

import "time"

// LogFilter constrains which entries are returned by List.
type LogFilter struct {
	Date *time.Time // exact calendar day
	From *time.Time // range start (inclusive)
	To   *time.Time // range end (inclusive)
	Tag  string
}

// EditParams carries the optional fields that may be updated on an entry.
// Only non-nil / non-zero fields are applied.
type EditParams struct {
	Tag       *string
	StartedAt *time.Time
	StoppedAt *time.Time
	Note      *string
	BreakNote *string
	BreakQualifies *bool
}

// UndoResult describes what undo did.
type UndoResult struct {
	Action string // "deleted" (undid a start) or "reopened" (undid a stop)
	Entry  *Entry
}

// Store is the persistence interface for entries.
// All implementations must be safe for sequential use (one writer connection).
//
//go:generate mockery
type Store interface {
	// Create inserts a new entry and returns it with its assigned ID.
	Create(tag string, startedAt time.Time, note string) (*Entry, error)

	// CreateWithActivity atomically creates an entry and its initial task.
	CreateWithActivity(tag string, startedAt time.Time, text string) (*Entry, *Activity, error)

	// Stop sets stopped_at on the open entry with the given tag.
	// Returns ErrNoOpenEntry when no open entry exists for that tag.
	Stop(tag string, stoppedAt time.Time, reason string, breakQualifies bool) (*Entry, error)

	// Get returns a single entry by ID.
	Get(id int64) (*Entry, error)

	// List returns entries matching the filter, ordered by started_at ascending.
	List(f LogFilter) ([]Entry, error)

	// OpenEntry returns the currently open entry for the given tag, or nil.
	OpenEntry(tag string) (*Entry, error)

	// LastStopped returns the most recently stopped entry for the given tag, or nil.
	LastStopped(tag string) (*Entry, error)

	// UndoLast reverses the most recent action for the given tag:
	//   - If the last entry is open (last action was a start) -> deletes it.
	//   - If the last entry is stopped (last action was a stop) -> clears stopped_at, reopening it.
	// Returns ErrEntryNotFound when there are no entries for that tag.
	UndoLast(tag string) (*UndoResult, error)

	// Edit applies the non-nil fields in EditParams to the entry with the given ID.
	Edit(id int64, p EditParams) (*Entry, error)

	// Delete removes an entry by ID.
	Delete(id int64) error

	// EntryAt returns the work entry containing at, or nil.
	EntryAt(tag string, at time.Time) (*Entry, error)

	// LastEntryBefore returns the most recent closed entry ending at or before at.
	LastEntryBefore(tag string, at time.Time) (*Entry, error)

	// CreateActivity records unstructured activity text against a work entry.
	CreateActivity(entryID int64, occurredAt time.Time, text string) (*Activity, error)

	// ListActivities returns activities in chronological order.
	ListActivities(filter ActivityFilter) ([]Activity, error)

	// UndoLastActivity deletes and returns the latest-created activity.
	UndoLastActivity() (*Activity, error)
}

// Sentinel errors.
var (
	ErrNoOpenEntry  = storeError("no open entry for this tag")
	ErrEntryNotFound = storeError("entry not found")
	ErrActivityNotFound = storeError("activity not found")
)

type storeError string

func (e storeError) Error() string { return string(e) }
