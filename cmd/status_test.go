package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/astepanovs/ztime/internal"
	"github.com/astepanovs/ztime/mocks"
)

func TestStatus_NoSessions(t *testing.T) {
	store := mocks.NewMockStore(t)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(nil, nil)
	store.EXPECT().ListActivities(mock.AnythingOfType("internal.ActivityFilter")).Return(nil, nil)

	out, err := execute(t, store, "status")
	require.NoError(t, err)
	requireContains(t, out, "no work entries today")
}

func TestStatus_WithOpenSession(t *testing.T) {
	store := mocks.NewMockStore(t)
	sessions := []internal.Entry{
		{ID: 1, Tag: "work", StartedAt: time.Now().Add(-3 * time.Hour)},
	}
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(sessions, nil)
	store.EXPECT().ListActivities(mock.AnythingOfType("internal.ActivityFilter")).Return(nil, nil)

	out, err := execute(t, store, "status")
	require.NoError(t, err)
	requireContains(t, out, "running")
	requireContains(t, out, "work")
}

func TestStatus_WithClosedSessions(t *testing.T) {
	store := mocks.NewMockStore(t)
	stopped := time.Now()
	sessions := []internal.Entry{
		{
			ID: 1, Tag: "work",
			StartedAt: stopped.Add(-2 * time.Hour),
			StoppedAt: &stopped,
		},
	}
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(sessions, nil)
	store.EXPECT().ListActivities(mock.AnythingOfType("internal.ActivityFilter")).Return(nil, nil)

	out, err := execute(t, store, "status")
	require.NoError(t, err)
	requireContains(t, out, "work")
	requireContains(t, out, "2h")
}

func TestStatus_GroupsTasksUnderEntries(t *testing.T) {
	store := mocks.NewMockStore(t)
	stopped := time.Now()
	entries := []internal.Entry{{
		ID: 4, Tag: "work", StartedAt: stopped.Add(-2 * time.Hour), StoppedAt: &stopped,
	}}
	activities := []internal.Activity{{
		ID: 7, EntryID: 4, OccurredAt: stopped.Add(-time.Hour), Text: "Prepared release notes",
	}}
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(entries, nil)
	store.EXPECT().ListActivities(mock.AnythingOfType("internal.ActivityFilter")).Return(activities, nil)

	out, err := execute(t, store, "status")
	require.NoError(t, err)
	requireContains(t, out, "[task #7] Prepared release notes")
	requireContains(t, out, "└─")
}

func TestStatus_NoTasksSkipsTaskQuery(t *testing.T) {
	store := mocks.NewMockStore(t)
	stopped := time.Now()
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{{
		ID: 4, Tag: "work", StartedAt: stopped.Add(-time.Hour), StoppedAt: &stopped,
	}}, nil)

	out, err := execute(t, store, "status", "--no-tasks")
	require.NoError(t, err)
	require.NotContains(t, out, "task #")
}

func TestStatus_ShowsBreakBetweenEntries(t *testing.T) {
	store := mocks.NewMockStore(t)
	firstStop := time.Date(2026, 10, 1, 12, 20, 0, 0, time.Local)
	secondStart := time.Date(2026, 10, 1, 12, 55, 0, 0, time.Local)
	secondStop := secondStart.Add(time.Hour)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{
		{ID: 1, Tag: "work", StartedAt: firstStop.Add(-2 * time.Hour), StoppedAt: &firstStop, BreakNote: "lunch", BreakQualifies: true},
		{ID: 2, Tag: "work", StartedAt: secondStart, StoppedAt: &secondStop},
	}, nil)
	store.EXPECT().ListActivities(mock.AnythingOfType("internal.ActivityFilter")).Return(nil, nil)

	out, err := execute(t, store, "status")
	require.NoError(t, err)
	requireContains(t, out, "break  12:20 - 12:55  (35m)")
	requireContains(t, out, "// lunch")
}

func TestStatus_DoesNotShowNegativeBreakForOverlappingEntries(t *testing.T) {
	store := mocks.NewMockStore(t)
	firstStop := time.Date(2026, 10, 1, 13, 0, 0, 0, time.Local)
	secondStart := time.Date(2026, 10, 1, 12, 30, 0, 0, time.Local)
	secondStop := secondStart.Add(time.Hour)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{
		{ID: 1, Tag: "work", StartedAt: firstStop.Add(-2 * time.Hour), StoppedAt: &firstStop},
		{ID: 2, Tag: "work", StartedAt: secondStart, StoppedAt: &secondStop},
	}, nil)
	store.EXPECT().ListActivities(mock.AnythingOfType("internal.ActivityFilter")).Return(nil, nil)

	out, err := execute(t, store, "status")
	require.NoError(t, err)
	require.NotContains(t, out, "break")
}
