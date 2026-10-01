package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/astepanovs/ztime/internal"
	"github.com/astepanovs/ztime/mocks"
)

func TestBackfill_Success(t *testing.T) {
	store := mocks.NewMockStore(t)
	stoppedAt := time.Now().Add(-30 * time.Minute)
	last := &internal.Entry{ID: 1, Tag: "work", StoppedAt: &stoppedAt}
	created := &internal.Entry{ID: 2, Tag: "work", StartedAt: stoppedAt}

	store.EXPECT().OpenEntry("work").Return(nil, nil)
	store.EXPECT().LastStopped("work").Return(last, nil)
	store.EXPECT().Create("work", stoppedAt, "").Return(created, nil)

	out, err := execute(t, store, "backfill")
	require.NoError(t, err)
	requireContains(t, out, "backfilled work entry #2")
	requireContains(t, out, "no gap from #1")
}

func TestBackfill_NoPreviousSession(t *testing.T) {
	store := mocks.NewMockStore(t)

	store.EXPECT().OpenEntry("work").Return(nil, nil)
	store.EXPECT().LastStopped("work").Return(nil, nil)

	_, err := execute(t, store, "backfill")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no previous")
}

func TestBackfill_AlreadyOpen(t *testing.T) {
	store := mocks.NewMockStore(t)
	open := &internal.Entry{ID: 1, Tag: "work", StartedAt: time.Now()}

	store.EXPECT().OpenEntry("work").Return(open, nil)

	_, err := execute(t, store, "backfill")
	require.Error(t, err)
	require.Contains(t, err.Error(), "already open")
}

func TestBackfill_CustomTag(t *testing.T) {
	store := mocks.NewMockStore(t)
	stoppedAt := time.Now().Add(-5 * time.Minute)
	last := &internal.Entry{ID: 2, Tag: "focus", StoppedAt: &stoppedAt}
	created := &internal.Entry{ID: 3, Tag: "focus", StartedAt: stoppedAt}

	store.EXPECT().OpenEntry("focus").Return(nil, nil)
	store.EXPECT().LastStopped("focus").Return(last, nil)
	store.EXPECT().Create("focus", mock.AnythingOfType("time.Time"), "").Return(created, nil)

	out, err := execute(t, store, "backfill", "--tag", "focus")
	require.NoError(t, err)
	requireContains(t, out, "backfilled focus entry #3")
}
