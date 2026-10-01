package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func closedEntry(id int64, tag string, start time.Time, duration time.Duration) Entry {
	stop := start.Add(duration)
	return Entry{ID: id, Tag: tag, StartedAt: start, StoppedAt: &stop}
}

func TestCalculateBalance_WeekdayPlusHours(t *testing.T) {
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.Local)
	report := CalculateBalance([]Entry{closedEntry(1, "work", start, 9*time.Hour)})

	require.Len(t, report.Days, 1)
	assert.Equal(t, time.Hour, report.Balance)
	assert.Equal(t, 8*time.Hour, report.Target)
}

func TestCalculateBalance_WeekdayMinusHours(t *testing.T) {
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.Local)
	report := CalculateBalance([]Entry{closedEntry(1, "work", start, 6*time.Hour)})
	assert.Equal(t, -2*time.Hour, report.Balance)
}

func TestCalculateBalance_IgnoresUnloggedDays(t *testing.T) {
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.Local)
	report := CalculateBalance([]Entry{closedEntry(1, "work", start, 8*time.Hour)})

	require.Len(t, report.Days, 1)
	assert.Zero(t, report.Balance)
}

func TestCalculateBalance_WeekendWorkIsEntirelyPositive(t *testing.T) {
	saturday := time.Date(2026, 10, 3, 9, 0, 0, 0, time.Local)
	report := CalculateBalance([]Entry{closedEntry(1, "work", saturday, 3*time.Hour)})

	require.Len(t, report.Days, 1)
	assert.Zero(t, report.Target)
	assert.Equal(t, 3*time.Hour, report.Balance)
}

func TestCalculateBalance_PublicHolidayWorkIsEntirelyPositive(t *testing.T) {
	christmas := time.Date(2026, 12, 25, 9, 0, 0, 0, time.Local)
	report := CalculateBalance([]Entry{closedEntry(1, "work", christmas, 2*time.Hour)})

	assert.Zero(t, report.Target)
	assert.Equal(t, 2*time.Hour, report.Balance)
}

func TestCalculateBalance_SumsMultipleEntriesAndDays(t *testing.T) {
	thursday := time.Date(2026, 10, 1, 9, 0, 0, 0, time.Local)
	friday := time.Date(2026, 10, 2, 9, 0, 0, 0, time.Local)
	saturday := time.Date(2026, 10, 3, 9, 0, 0, 0, time.Local)
	report := CalculateBalance([]Entry{
		closedEntry(1, "work", thursday, 4*time.Hour),
		closedEntry(2, "work", thursday.Add(5*time.Hour), 4*time.Hour),
		closedEntry(3, "work", friday, 7*time.Hour),
		closedEntry(4, "work", saturday, 2*time.Hour),
	})

	require.Len(t, report.Days, 3)
	assert.Equal(t, 17*time.Hour, report.Worked)
	assert.Equal(t, 16*time.Hour, report.Target)
	assert.Equal(t, time.Hour, report.Balance)
}

func TestCalculateBalance_IgnoresOtherTags(t *testing.T) {
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.Local)
	report := CalculateBalance([]Entry{closedEntry(1, "hobby", start, 5*time.Hour)})
	assert.Empty(t, report.Days)
}
