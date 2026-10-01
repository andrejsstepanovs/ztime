package db

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/astepanovs/ztime/internal"
)

func TestListHandlesNullableNotes(t *testing.T) {
	conn, err := OpenAt(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	_, err = conn.Exec(`
		INSERT INTO sessions (tag, started_at, stopped_at, note, break_note)
		VALUES ('work', '2026-10-01T09:00:00', '2026-10-01T10:00:00', NULL, NULL)
	`)
	require.NoError(t, err)

	entries, err := NewStore(conn).List(internal.LogFilter{})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Empty(t, entries[0].Note)
	require.Empty(t, entries[0].BreakNote)
}
