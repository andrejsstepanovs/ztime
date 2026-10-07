package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func ts(y int, mo time.Month, d, h, mi, sec int) time.Time {
	return time.Date(y, mo, d, h, mi, sec, 0, time.Local)
}

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
