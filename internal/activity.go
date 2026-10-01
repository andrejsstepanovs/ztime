package internal

import "time"

type Activity struct {
	ID         int64
	EntryID    int64
	OccurredAt time.Time
	Text       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ActivityFilter struct {
	Date *time.Time
	From *time.Time
	To   *time.Time
}
