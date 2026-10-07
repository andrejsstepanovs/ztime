package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func ts(y int, mo time.Month, d, h, mi, sec int) time.Time {
	return time.Date(y, mo, d, h, mi, sec, 0, time.Local)
}

func pt(t time.Time) *time.Time { return &t }

func TestRounded_FloorsStartAndCeilsStop(t *testing.T) {
	step := 10 * time.Minute
	start := ts(2026, time.October, 6, 10, 9, 30)
	stop := ts(2026, time.October, 6, 17, 54, 3)

	got := Entry{Tag: "work", StartedAt: start, StoppedAt: &stop}.Rounded(step)

	require.Equal(t, ts(2026, time.October, 6, 10, 0, 0), got.StartedAt)
	require.Equal(t, ts(2026, time.October, 6, 18, 0, 0), *got.StoppedAt)
	require.Equal(t, 8*time.Hour, got.Duration())
}

func TestRounded_ExactBoundariesAndSeconds(t *testing.T) {
	step := 5 * time.Minute
	start := ts(2026, time.October, 6, 12, 45, 30)
	stop := ts(2026, time.October, 6, 12, 45, 0)

	got := Entry{StartedAt: start, StoppedAt: &stop}.Rounded(step)

	// Start seconds are dropped; exact stop boundary with zero seconds stays.
	require.Equal(t, ts(2026, time.October, 6, 12, 45, 0), got.StartedAt)
	require.Equal(t, ts(2026, time.October, 6, 12, 45, 0), *got.StoppedAt)
}

func TestRounded_StopCrossesMidnight(t *testing.T) {
	step := 10 * time.Minute
	start := ts(2026, time.October, 6, 23, 58, 0)
	stop := ts(2026, time.October, 6, 23, 59, 10)

	got := Entry{StartedAt: start, StoppedAt: &stop}.Rounded(step)

	require.Equal(t, ts(2026, time.October, 6, 23, 50, 0), got.StartedAt)
	require.Equal(t, ts(2026, time.October, 7, 0, 0, 0), *got.StoppedAt)
	require.Positive(t, got.Duration())
}

func TestRounded_OpenEntryKeepsNilStop(t *testing.T) {
	step := 15 * time.Minute
	start := ts(2026, time.October, 6, 9, 7, 0)

	got := Entry{StartedAt: start}.Rounded(step)

	require.Nil(t, got.StoppedAt)
	require.Equal(t, ts(2026, time.October, 6, 9, 0, 0), got.StartedAt)
	require.True(t, got.IsOpen())
}

func TestRounded_DoesNotMutateReceiverAndNeverShortens(t *testing.T) {
	step := 30 * time.Minute
	start := ts(2026, time.October, 6, 9, 17, 5)
	stop := ts(2026, time.October, 6, 17, 43, 1)
	orig := Entry{ID: 7, Tag: "work", Note: "n", StartedAt: start, StoppedAt: &stop}

	got := orig.Rounded(step)

	require.Equal(t, ts(2026, time.October, 6, 9, 17, 5), orig.StartedAt)
	require.Equal(t, ts(2026, time.October, 6, 17, 43, 1), *orig.StoppedAt)
	require.Equal(t, int64(7), got.ID)
	require.Equal(t, "work", got.Tag)
	require.Equal(t, "n", got.Note)
	require.GreaterOrEqual(t, got.Duration(), orig.Duration())
}

func TestRoundEntries_TrimsOverlappingBoundary(t *testing.T) {
	step := 10 * time.Minute
	a := Entry{ID: 1, Tag: "work",
		StartedAt: ts(2026, 10, 6, 10, 0, 0),
		StoppedAt: pt(ts(2026, 10, 6, 11, 55, 30))}
	b := Entry{ID: 2, Tag: "work",
		StartedAt: ts(2026, 10, 6, 11, 56, 0),
		StoppedAt: pt(ts(2026, 10, 6, 12, 31, 0))}

	got := RoundEntries([]Entry{a, b}, step)

	// A's ceiled stop (12:00) lands after B's floored start (11:50): trim to 11:50.
	require.Equal(t, ts(2026, 10, 6, 10, 0, 0), got[0].StartedAt)
	require.Equal(t, ts(2026, 10, 6, 11, 50, 0), *got[0].StoppedAt)
	require.Equal(t, ts(2026, 10, 6, 11, 50, 0), got[1].StartedAt)
	require.Equal(t, ts(2026, 10, 6, 12, 40, 0), *got[1].StoppedAt)
}

func TestRoundEntries_KeepsRealGaps(t *testing.T) {
	step := 10 * time.Minute
	a := Entry{Tag: "work",
		StartedAt: ts(2026, 10, 6, 10, 0, 0),
		StoppedAt: pt(ts(2026, 10, 6, 11, 0, 0))}
	b := Entry{Tag: "work",
		StartedAt: ts(2026, 10, 6, 11, 30, 0),
		StoppedAt: pt(ts(2026, 10, 6, 12, 0, 0))}

	got := RoundEntries([]Entry{a, b}, step)

	require.Equal(t, ts(2026, 10, 6, 11, 0, 0), *got[0].StoppedAt)
	require.Equal(t, ts(2026, 10, 6, 11, 30, 0), got[1].StartedAt)
}

func TestRoundEntries_TrimsOnlyTheOverlappingPair(t *testing.T) {
	step := 10 * time.Minute
	entries := []Entry{
		{ID: 1, Tag: "work",
			StartedAt: ts(2026, 10, 6, 9, 0, 0),
			StoppedAt: pt(ts(2026, 10, 6, 9, 5, 0))}, // 9:10 stop, clear of #2
		{ID: 2, Tag: "work",
			StartedAt: ts(2026, 10, 6, 9, 36, 0),
			StoppedAt: pt(ts(2026, 10, 6, 9, 58, 0))}, // 9:30..10:00, overlaps #3
		{ID: 3, Tag: "work",
			StartedAt: ts(2026, 10, 6, 9, 57, 0),
			StoppedAt: pt(ts(2026, 10, 6, 10, 30, 0))}, // 9:50..10:30
	}

	got := RoundEntries(entries, step)

	require.Equal(t, ts(2026, 10, 6, 9, 10, 0), *got[0].StoppedAt)
	require.Equal(t, ts(2026, 10, 6, 9, 50, 0), *got[1].StoppedAt)
	require.Equal(t, ts(2026, 10, 6, 10, 30, 0), *got[2].StoppedAt)
}

func TestRoundEntries_SkipsOtherTags(t *testing.T) {
	step := 10 * time.Minute
	work := Entry{Tag: "work",
		StartedAt: ts(2026, 10, 6, 10, 0, 0),
		StoppedAt: pt(ts(2026, 10, 6, 11, 56, 0))} // ceils to 12:00
	lunch := Entry{Tag: "lunch",
		StartedAt: ts(2026, 10, 6, 11, 58, 0),
		StoppedAt: pt(ts(2026, 10, 6, 12, 30, 0))}

	got := RoundEntries([]Entry{work, lunch}, step)

	// Lunch is a different tag: work keeps its ceiled stop.
	require.Equal(t, ts(2026, 10, 6, 12, 0, 0), *got[0].StoppedAt)
}

func TestRoundEntries_OpenEntryAsSuccessor(t *testing.T) {
	step := 10 * time.Minute
	closed := Entry{Tag: "work",
		StartedAt: ts(2026, 10, 6, 10, 0, 0),
		StoppedAt: pt(ts(2026, 10, 6, 11, 56, 0))}
	open := Entry{Tag: "work", StartedAt: ts(2026, 10, 6, 11, 57, 0)}

	got := RoundEntries([]Entry{closed, open}, step)

	require.Equal(t, ts(2026, 10, 6, 11, 50, 0), *got[0].StoppedAt)
	require.Nil(t, got[1].StoppedAt)
}
