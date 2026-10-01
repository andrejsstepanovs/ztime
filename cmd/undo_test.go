package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/astepanovs/ztime/internal"
	"github.com/astepanovs/ztime/mocks"
)

func TestUndo_UndoStop(t *testing.T) {
	store := mocks.NewMockStore(t)
	sess := &internal.Entry{ID: 3, Tag: "work", StartedAt: time.Now().Add(-2 * time.Hour)}
	result := &internal.UndoResult{Action: "reopened", Entry: sess}

	store.EXPECT().UndoLast("work").Return(result, nil)

	out, err := execute(t, store, "undo")
	require.NoError(t, err)
	requireContains(t, out, "undid stop")
	requireContains(t, out, "reopened work entry #3")
}

func TestUndo_UndoStart(t *testing.T) {
	store := mocks.NewMockStore(t)
	sess := &internal.Entry{ID: 4, Tag: "work", StartedAt: time.Now()}
	result := &internal.UndoResult{Action: "deleted", Entry: sess}

	store.EXPECT().UndoLast("work").Return(result, nil)

	out, err := execute(t, store, "undo")
	require.NoError(t, err)
	requireContains(t, out, "undid start")
	requireContains(t, out, "deleted work entry #4")
}

func TestUndo_NoSessions(t *testing.T) {
	store := mocks.NewMockStore(t)
	store.EXPECT().UndoLast("work").Return(nil, internal.ErrEntryNotFound)

	_, err := execute(t, store, "undo")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no \"work\" entries to undo")
}

func TestUndo_CustomTag(t *testing.T) {
	store := mocks.NewMockStore(t)
	sess := &internal.Entry{ID: 5, Tag: "lunch", StartedAt: time.Now()}
	result := &internal.UndoResult{Action: "deleted", Entry: sess}

	store.EXPECT().UndoLast("lunch").Return(result, nil)

	out, err := execute(t, store, "undo", "--tag", "lunch")
	require.NoError(t, err)
	requireContains(t, out, "undid start")
	requireContains(t, out, "#5")
}
