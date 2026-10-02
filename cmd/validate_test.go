package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/astepanovs/ztime/internal"
	"github.com/astepanovs/ztime/mocks"
)

func TestValidate_NoIssues(t *testing.T) {
	store := mocks.NewMockStore(t)
	now := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 10, 0, 0, 0, time.Local)
	stopped := now.Add(-1 * time.Hour)
	entries := []internal.Entry{
		{ID: 1, Tag: "work", StartedAt: stopped.Add(-1 * time.Hour), StoppedAt: &stopped},
	}
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(entries, nil)

	out, err := execute(t, store, "validate")
	require.NoError(t, err)
	requireContains(t, out, "no issues found")
}

func TestValidate_TooLong(t *testing.T) {
	store := mocks.NewMockStore(t)
	now := time.Now()
	stopped := now
	entries := []internal.Entry{
		{ID: 1, Tag: "work", StartedAt: now.Add(-11 * time.Hour), StoppedAt: &stopped},
	}
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(entries, nil)

	_, err := execute(t, store, "validate")
	require.Error(t, err)
	require.Contains(t, err.Error(), "issue(s)")
}

func TestValidate_ConsecutiveStarts(t *testing.T) {
	store := mocks.NewMockStore(t)
	now := time.Now()
	entries := []internal.Entry{
		{ID: 1, Tag: "work", StartedAt: now.Add(-3 * time.Hour)},
		{ID: 2, Tag: "work", StartedAt: now.Add(-1 * time.Hour)},
	}
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(entries, nil)

	out, err := execute(t, store, "validate")
	require.Error(t, err)
	requireContains(t, out, "consecutive start")
}
