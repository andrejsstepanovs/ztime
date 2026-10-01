package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/astepanovs/ztime/internal"
	"github.com/astepanovs/ztime/mocks"
)

func TestActivity_AddDuringWork(t *testing.T) {
	store := mocks.NewMockStore(t)
	entry := &internal.Entry{ID: 4, Tag: "work", StartedAt: time.Now().Add(-time.Hour)}
	activity := &internal.Activity{ID: 1, EntryID: 4, OccurredAt: time.Now(), Text: "Reviewed checkout alerts"}
	store.EXPECT().EntryAt("work", mock.AnythingOfType("time.Time")).Return(entry, nil)
	store.EXPECT().CreateActivity(int64(4), mock.AnythingOfType("time.Time"), "Reviewed checkout alerts").Return(activity, nil)

	out, err := execute(t, store, "task", "Reviewed", "checkout", "alerts")
	require.NoError(t, err)
	requireContains(t, out, "logged task #1")
	requireContains(t, out, "Reviewed checkout alerts")
}

func TestActivity_AfterStopAttachesToLastEntry(t *testing.T) {
	store := mocks.NewMockStore(t)
	stop := time.Now().Add(-30 * time.Minute)
	entry := &internal.Entry{ID: 4, Tag: "work", StartedAt: stop.Add(-time.Hour), StoppedAt: &stop}
	activity := &internal.Activity{ID: 2, EntryID: 4, OccurredAt: stop, Text: "Finished deployment notes"}
	store.EXPECT().EntryAt("work", mock.AnythingOfType("time.Time")).Return(nil, nil)
	store.EXPECT().LastEntryBefore("work", mock.AnythingOfType("time.Time")).Return(entry, nil)
	store.EXPECT().CreateActivity(int64(4), stop, "Finished deployment notes").Return(activity, nil)

	out, err := execute(t, store, "task", "Finished deployment notes")
	require.NoError(t, err)
	requireContains(t, out, "warning: no active work")
	requireContains(t, out, "logged task #2")
}

func TestActivity_RequiresWorkEntry(t *testing.T) {
	store := mocks.NewMockStore(t)
	store.EXPECT().EntryAt("work", mock.AnythingOfType("time.Time")).Return(nil, nil)
	store.EXPECT().LastEntryBefore("work", mock.AnythingOfType("time.Time")).Return(nil, nil)

	_, err := execute(t, store, "task", "Did some work")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no work entry")
}

func TestActivity_List(t *testing.T) {
	store := mocks.NewMockStore(t)
	at := time.Now()
	store.EXPECT().ListActivities(mock.AnythingOfType("internal.ActivityFilter")).Return([]internal.Activity{{
		ID: 1, EntryID: 4, OccurredAt: at, Text: "Reviewed PR",
	}}, nil)

	out, err := execute(t, store, "tasks")
	require.NoError(t, err)
	requireContains(t, out, "Reviewed PR")
}

func TestActivity_Undo(t *testing.T) {
	store := mocks.NewMockStore(t)
	store.EXPECT().UndoLastActivity().Return(&internal.Activity{ID: 3, Text: "Wrong note"}, nil)

	out, err := execute(t, store, "task", "undo")
	require.NoError(t, err)
	requireContains(t, out, "undid task #3")
}
