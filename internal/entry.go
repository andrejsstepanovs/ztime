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

// Rounded returns a copy of the entry with the start time floored and the
// stop time ceiled to the nearest multiple of step. Open entries keep their
// nil stop; only their start is rounded. Rounded duration is therefore never
// shorter than the real one. Presentation only; the stored entry is untouched.
func (e Entry) Rounded(step time.Duration) Entry {
	e.StartedAt = roundDown(e.StartedAt, step)
	if e.StoppedAt != nil {
		stop := roundUp(*e.StoppedAt, step)
		e.StoppedAt = &stop
	}
	return e
}

// RoundEntries rounds every entry via Rounded and then trims overlapping
// boundaries: an entry's rounded stop never lands after the next same-tag
// entry's rounded start, so rounding cannot invent work between entries.
// Entries must be in chronological order, as returned by Store.List. Other
// tags in between (e.g. lunch) are skipped, not trimmed into.
func RoundEntries(entries []Entry, step time.Duration) []Entry {
	out := make([]Entry, len(entries))
	for i := range entries {
		out[i] = entries[i].Rounded(step)
	}
	for i := range out {
		if out[i].StoppedAt == nil {
			continue
		}
		for j := i + 1; j < len(out); j++ {
			if out[j].Tag != out[i].Tag {
				continue
			}
			if out[i].StoppedAt.After(out[j].StartedAt) {
				stop := out[j].StartedAt
				out[i].StoppedAt = &stop
			}
			break
		}
	}
	return out
}

func roundDown(t time.Time, step time.Duration) time.Time {
	stepMin := int(step / time.Minute)
	minutes := t.Hour()*60 + t.Minute()
	minutes -= minutes % stepMin
	return time.Date(t.Year(), t.Month(), t.Day(), minutes/60, minutes%60, 0, 0, t.Location())
}

func roundUp(t time.Time, step time.Duration) time.Time {
	stepMin := int(step / time.Minute)
	minutes := t.Hour()*60 + t.Minute()
	remainder := minutes % stepMin
	if remainder != 0 || t.Second() > 0 || t.Nanosecond() > 0 {
		minutes += stepMin - remainder
	}
	return time.Date(t.Year(), t.Month(), t.Day(), minutes/60, minutes%60, 0, 0, t.Location())
}

// Duration returns elapsed time. For open entries it measures against now.
func (e Entry) Duration() time.Duration {
	end := time.Now()
	if e.StoppedAt != nil {
		end = *e.StoppedAt
	}
	return end.Sub(e.StartedAt)
}
