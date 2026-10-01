package internal

import "time"

// Entry represents one work/lunch/break time block.
type Entry struct {
	ID        int64
	Tag       string
	StartedAt time.Time
	StoppedAt *time.Time // nil when entry is still open
	Note      string
	BreakNote string
	BreakQualifies bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsOpen returns true when the entry has no stop time.
func (e Entry) IsOpen() bool {
	return e.StoppedAt == nil
}

// Duration returns elapsed time. For open entries it measures against now.
func (e Entry) Duration() time.Duration {
	end := time.Now()
	if e.StoppedAt != nil {
		end = *e.StoppedAt
	}
	return end.Sub(e.StartedAt)
}
