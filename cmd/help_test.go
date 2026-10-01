package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/astepanovs/ztime/mocks"
)

func TestTaskHelpShowsArguments(t *testing.T) {
	store := mocks.NewMockStore(t)
	out, err := execute(t, store, "task", "--help")
	require.NoError(t, err)
	requireContains(t, out, "Arguments:")
	requireContains(t, out, "<text>")
	requireContains(t, out, "free-form description")
}

func TestEditHelpShowsArguments(t *testing.T) {
	store := mocks.NewMockStore(t)
	out, err := execute(t, store, "edit", "--help")
	require.NoError(t, err)
	requireContains(t, out, "Arguments:")
	requireContains(t, out, "<id>")
	requireContains(t, out, "numeric entry ID")
}

func TestCommandsWithoutArgumentsDoNotShowArgumentsSection(t *testing.T) {
	store := mocks.NewMockStore(t)
	out, err := execute(t, store, "status", "--help")
	require.NoError(t, err)
	require.NotContains(t, out, "Arguments:")
}
