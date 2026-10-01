package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/astepanovs/ztime/internal"
	"github.com/astepanovs/ztime/mocks"
)

func TestDelete_Success(t *testing.T) {
	store := mocks.NewMockStore(t)
	store.EXPECT().Delete(int64(42)).Return(nil)

	out, err := execute(t, store, "delete", "42")
	require.NoError(t, err)
	requireContains(t, out, "deleted entry #42")
}

func TestDelete_NotFound(t *testing.T) {
	store := mocks.NewMockStore(t)
	store.EXPECT().Delete(int64(99)).Return(internal.ErrEntryNotFound)

	_, err := execute(t, store, "delete", "99")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestDelete_InvalidID(t *testing.T) {
	store := mocks.NewMockStore(t)

	_, err := execute(t, store, "delete", "abc")
	require.Error(t, err)
}
