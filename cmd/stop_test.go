package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/astepanovs/ztime/internal"
	"github.com/astepanovs/ztime/mocks"
)

func TestStop_Success(t *testing.T) {
	store := mocks.NewMockStore(t)
	stopped := time.Now()
	sess := &internal.Entry{
		ID: 1, Tag: "work",
		StartedAt: stopped.Add(-2 * time.Hour),
		StoppedAt: &stopped,
	}

	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{{
		ID: 1, Tag: "work", StartedAt: sess.StartedAt,
	}}, nil)
	store.EXPECT().OpenEntry("work").Return(&internal.Entry{
		ID: 1, Tag: "work", StartedAt: sess.StartedAt,
	}, nil)
	store.EXPECT().Stop("work", mock.AnythingOfType("time.Time"), "", false).Return(sess, nil)

	out, err := execute(t, store, "stop")
	require.NoError(t, err)
	requireContains(t, out, "stopped work entry #1")
	requireContains(t, out, "2h")
}

func TestStop_NoOpenSession(t *testing.T) {
	store := mocks.NewMockStore(t)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(nil, nil)
	store.EXPECT().OpenEntry("work").Return(nil, nil)

	_, err := execute(t, store, "stop")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no open")
}

func TestStop_CustomTag(t *testing.T) {
	store := mocks.NewMockStore(t)
	stopped := time.Now()
	sess := &internal.Entry{
		ID: 5, Tag: "lunch",
		StartedAt: stopped.Add(-30 * time.Minute),
		StoppedAt: &stopped,
	}

	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{{
		ID: 5, Tag: "lunch", StartedAt: sess.StartedAt,
	}}, nil)
	store.EXPECT().OpenEntry("lunch").Return(&internal.Entry{
		ID: 5, Tag: "lunch", StartedAt: sess.StartedAt,
	}, nil)
	store.EXPECT().Stop("lunch", mock.AnythingOfType("time.Time"), "", false).Return(sess, nil)

	out, err := execute(t, store, "stop", "--tag", "lunch")
	require.NoError(t, err)
	requireContains(t, out, "stopped lunch entry #5")
}

func TestStop_BlocksLongEntryUnlessForced(t *testing.T) {
	store := mocks.NewMockStore(t)
	started := time.Now().Add(-11 * time.Hour)
	open := &internal.Entry{ID: 1, Tag: "work", StartedAt: started}
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{*open}, nil)
	store.EXPECT().OpenEntry("work").Return(open, nil)
	closed := &internal.Entry{ID: 1, Tag: "work", StartedAt: started}
	store.EXPECT().Stop("work", mock.AnythingOfType("time.Time"), "", false).Return(closed, nil)

	out, err := execute(t, store, "stop")
	require.NoError(t, err)
	requireContains(t, out, "exceeds 10h")
	requireContains(t, out, "stopped work entry")
}

func TestStop_ForceAllowsLongEntry(t *testing.T) {
	store := mocks.NewMockStore(t)
	started := time.Now().Add(-11 * time.Hour)
	stopped := time.Now()
	open := &internal.Entry{ID: 1, Tag: "work", StartedAt: started}
	closed := &internal.Entry{ID: 1, Tag: "work", StartedAt: started, StoppedAt: &stopped}
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{*open}, nil)
	store.EXPECT().OpenEntry("work").Return(open, nil)
	store.EXPECT().Stop("work", mock.AnythingOfType("time.Time"), "", false).Return(closed, nil)

	out, err := execute(t, store, "stop", "--force")
	require.NoError(t, err)
	requireContains(t, out, "warning:")
	requireContains(t, out, "stopped work entry #1")
}

func TestStop_ReasonIsPassedAsBreakReason(t *testing.T) {
	store := mocks.NewMockStore(t)
	started := time.Now().Add(-time.Hour)
	stopped := time.Now()
	open := &internal.Entry{ID: 1, Tag: "work", StartedAt: started}
	closed := &internal.Entry{ID: 1, Tag: "work", StartedAt: started, StoppedAt: &stopped, BreakNote: "lunch"}
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{*open}, nil)
	store.EXPECT().OpenEntry("work").Return(open, nil)
	store.EXPECT().Stop("work", mock.AnythingOfType("time.Time"), "lunch", false).Return(closed, nil)

	out, err := execute(t, store, "stop", "--reason", "lunch")
	require.NoError(t, err)
	requireContains(t, out, "stopped work entry #1")
}
