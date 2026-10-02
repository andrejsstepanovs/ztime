package cmd

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/astepanovs/ztime/internal"
	"github.com/astepanovs/ztime/mocks"
)

func TestStart_Success(t *testing.T) {
	store := mocks.NewMockStore(t)
	sess := &internal.Entry{ID: 1, Tag: "work", StartedAt: time.Now()}

	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(nil, nil)
	store.EXPECT().LastStopped("work").Return(nil, nil)
	store.EXPECT().Create("work", mock.AnythingOfType("time.Time"), "").Return(sess, nil)

	out, err := execute(t, store, "start")
	require.NoError(t, err)
	requireContains(t, out, "started work entry #1")
}

func TestStart_AlreadyOpen(t *testing.T) {
	store := mocks.NewMockStore(t)
	open := &internal.Entry{ID: 1, Tag: "work", StartedAt: time.Now()}

	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{*open}, nil)

	_, err := execute(t, store, "start")
	require.Error(t, err)
	require.Contains(t, err.Error(), "working-time rules")
}

func TestStart_StoreError(t *testing.T) {
	store := mocks.NewMockStore(t)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(nil, errors.New("db error"))

	_, err := execute(t, store, "start")
	require.Error(t, err)
}

func TestStart_CustomTag(t *testing.T) {
	store := mocks.NewMockStore(t)
	sess := &internal.Entry{ID: 2, Tag: "lunch", StartedAt: time.Now()}

	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(nil, nil)
	store.EXPECT().LastStopped("lunch").Return(nil, nil)
	store.EXPECT().Create("lunch", mock.AnythingOfType("time.Time"), "").Return(sess, nil)

	out, err := execute(t, store, "start", "--tag", "lunch")
	require.NoError(t, err)
	requireContains(t, out, "started lunch entry #2")
}

func TestStart_WithTask(t *testing.T) {
	store := mocks.NewMockStore(t)
	entry := &internal.Entry{ID: 3, Tag: "work", StartedAt: time.Now()}
	activity := &internal.Activity{ID: 7, EntryID: 3, OccurredAt: entry.StartedAt, Text: "focus block"}

	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(nil, nil)
	store.EXPECT().LastStopped("work").Return(nil, nil)
	store.EXPECT().CreateWithActivity("work", mock.AnythingOfType("time.Time"), "focus block").Return(entry, activity, nil)

	out, err := execute(t, store, "start", "--task", "focus block")
	require.NoError(t, err)
	requireContains(t, out, "#3")
	requireContains(t, out, "logged task #7")
}

func TestStart_GapWarning(t *testing.T) {
	store := mocks.NewMockStore(t)
	stoppedAt := time.Now().Add(-3 * time.Hour)
	last := &internal.Entry{ID: 1, Tag: "work", StoppedAt: &stoppedAt}
	sess := &internal.Entry{ID: 2, Tag: "work", StartedAt: time.Now()}

	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(nil, nil)
	store.EXPECT().LastStopped("work").Return(last, nil)
	store.EXPECT().Create("work", mock.AnythingOfType("time.Time"), "").Return(sess, nil)

	out, err := execute(t, store, "start")
	require.NoError(t, err)
	requireContains(t, out, "warning:")
	requireContains(t, out, "gap")
}

func TestStart_NoGapWarningWhenSmallGap(t *testing.T) {
	store := mocks.NewMockStore(t)
	stoppedAt := time.Now().Add(-30 * time.Minute)
	last := &internal.Entry{ID: 1, Tag: "work", StoppedAt: &stoppedAt}
	sess := &internal.Entry{ID: 2, Tag: "work", StartedAt: time.Now()}

	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(nil, nil)
	store.EXPECT().LastStopped("work").Return(last, nil)
	store.EXPECT().Create("work", mock.AnythingOfType("time.Time"), "").Return(sess, nil)

	out, err := execute(t, store, "start")
	require.NoError(t, err)
	require.NotContains(t, out, "warning:")
}

func TestStart_BlocksConsecutiveStartUnlessForced(t *testing.T) {
	store := mocks.NewMockStore(t)
	open := internal.Entry{ID: 1, Tag: "work", StartedAt: time.Now().Add(-time.Hour)}
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{open}, nil)

	out, err := execute(t, store, "start")
	require.Error(t, err)
	requireContains(t, out, "still active")
	require.Contains(t, err.Error(), "--force")
}

func TestStart_ForceAllowsConsecutiveStart(t *testing.T) {
	store := mocks.NewMockStore(t)
	open := internal.Entry{ID: 1, Tag: "work", StartedAt: time.Now().Add(-time.Hour)}
	created := &internal.Entry{ID: 2, Tag: "work", StartedAt: time.Now()}
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{open}, nil)
	store.EXPECT().LastStopped("work").Return(nil, nil)
	store.EXPECT().Create("work", mock.AnythingOfType("time.Time"), "").Return(created, nil)

	out, err := execute(t, store, "start", "--force")
	require.NoError(t, err)
	requireContains(t, out, "validation bypassed")
	requireContains(t, out, "started work entry #2")
}

func TestStart_PrintsExistingIssuesAsWarnings(t *testing.T) {
	store := mocks.NewMockStore(t)
	now := time.Now()
	stopped := now.Add(-time.Hour)
	broken := internal.Entry{ID: 1, Tag: "other", StartedAt: now.Add(-12 * time.Hour), StoppedAt: &stopped}
	created := &internal.Entry{ID: 2, Tag: "work", StartedAt: now}
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{broken}, nil)
	store.EXPECT().LastStopped("work").Return(nil, nil)
	store.EXPECT().Create("work", mock.AnythingOfType("time.Time"), "").Return(created, nil)

	out, err := execute(t, store, "start")
	require.NoError(t, err)
	requireContains(t, out, "warning: existing data:")
	requireContains(t, out, "exceeds 10h")
}
