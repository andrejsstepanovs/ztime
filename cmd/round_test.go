package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/astepanovs/ztime/internal"
	"github.com/astepanovs/ztime/mocks"
)

func roundedEntry() internal.Entry {
	start := time.Date(2026, 10, 6, 10, 9, 30, 0, time.Local)
	stop := time.Date(2026, 10, 6, 17, 54, 3, 0, time.Local)
	return internal.Entry{ID: 1, Tag: "work", StartedAt: start, StoppedAt: &stop}
}

func TestReport_RoundFlag(t *testing.T) {
	store := mocks.NewMockStore(t)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{roundedEntry()}, nil)

	out, err := execute(t, store, "report", "--round", "10")
	require.NoError(t, err)
	requireContains(t, out, "10:00 - 18:00  8h 0m")
	requireContains(t, out, "total  8h 0m")
}

func TestLog_RoundFlag(t *testing.T) {
	store := mocks.NewMockStore(t)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{roundedEntry()}, nil)

	out, err := execute(t, store, "log", "-r", "15")
	require.NoError(t, err)
	requireContains(t, out, "10:00 - 18:00  8h 0m")
	requireContains(t, out, "work      8h 0m")
}

func TestLog_RoundFlagJSON(t *testing.T) {
	store := mocks.NewMockStore(t)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{roundedEntry()}, nil)

	out, err := execute(t, store, "log", "--round", "10", "--json")
	require.NoError(t, err)
	requireContains(t, out, `"started_at": "2026-10-06T10:00:00+02:00"`)
	requireContains(t, out, `"stopped_at": "2026-10-06T18:00:00+02:00"`)
	requireContains(t, out, `"duration": "8h 0m"`)
}

func TestBalance_RoundFlag(t *testing.T) {
	store := mocks.NewMockStore(t)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{roundedEntry()}, nil)

	// Real duration is ~7h 45m (a 15m deficit); rounding to 10 grows it to 8h exactly.
	out, err := execute(t, store, "balance", "--round", "10")
	require.NoError(t, err)
	requireContains(t, out, "worked 8h 0m  target 8h 0m  balance +0m")
}

func TestLog_RoundTrimsOverlapInOutput(t *testing.T) {
	first := internal.Entry{ID: 1, Tag: "work",
		StartedAt: time.Date(2026, 10, 6, 10, 0, 0, 0, time.Local),
		StoppedAt: ptrTime(time.Date(2026, 10, 6, 11, 55, 30, 0, time.Local))}
	second := internal.Entry{ID: 2, Tag: "work",
		StartedAt: time.Date(2026, 10, 6, 11, 56, 0, 0, time.Local),
		StoppedAt: ptrTime(time.Date(2026, 10, 6, 12, 31, 0, 0, time.Local))}
	store := mocks.NewMockStore(t)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{first, second}, nil)

	out, err := execute(t, store, "log", "--round", "10")
	require.NoError(t, err)
	requireContains(t, out, "10:00 - 11:50  1h 50m")
	requireContains(t, out, "11:50 - 12:40  50m")
	requireContains(t, out, "work      2h 40m")
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestRound_NegativeValueErrors(t *testing.T) {
	store := mocks.NewMockStore(t)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(nil, nil)

	_, err := execute(t, store, "log", "--round", "-5")
	require.ErrorContains(t, err, "--round must be a positive number of minutes")
}
