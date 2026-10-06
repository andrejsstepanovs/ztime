package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/astepanovs/ztime/internal"
	"github.com/astepanovs/ztime/mocks"
)

func TestReport_NoPeriods(t *testing.T) {
	store := mocks.NewMockStore(t)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(nil, nil)

	out, err := execute(t, store, "report")
	require.NoError(t, err)
	requireContains(t, out, "no work periods found")
}

func TestReport_PrintsPeriodsAndDailyTotal(t *testing.T) {
	store := mocks.NewMockStore(t)
	firstStart := time.Date(2026, 10, 1, 9, 0, 0, 0, time.Local)
	firstStop := time.Date(2026, 10, 1, 12, 0, 0, 0, time.Local)
	secondStart := time.Date(2026, 10, 1, 12, 30, 0, 0, time.Local)
	secondStop := time.Date(2026, 10, 1, 17, 30, 0, 0, time.Local)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{
		{ID: 1, Tag: "work", StartedAt: firstStart, StoppedAt: &firstStop},
		{ID: 2, Tag: "work", StartedAt: secondStart, StoppedAt: &secondStop},
	}, nil)

	out, err := execute(t, store, "report")
	require.NoError(t, err)
	requireContains(t, out, "09:00 - 12:00  3h 0m")
	requireContains(t, out, "12:30 - 17:30  5h 0m")
	requireContains(t, out, "total  8h 0m")
}

func TestReport_MarksOpenPeriod(t *testing.T) {
	store := mocks.NewMockStore(t)
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.Local)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{{
		ID: 1, Tag: "work", StartedAt: start,
	}}, nil)

	out, err := execute(t, store, "report")
	require.NoError(t, err)
	requireContains(t, out, "09:00 - open")
	requireContains(t, out, "not ready to enter")
	requireContains(t, out, "total")
	requireContains(t, out, "(running)")
}
