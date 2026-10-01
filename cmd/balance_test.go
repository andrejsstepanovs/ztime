package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/astepanovs/ztime/internal"
	"github.com/astepanovs/ztime/mocks"
)

func TestBalance_NoEntries(t *testing.T) {
	store := mocks.NewMockStore(t)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return(nil, nil)

	out, err := execute(t, store, "balance")
	require.NoError(t, err)
	requireContains(t, out, "no work entries found")
}

func TestBalance_PrintsDailyAndTotalBalance(t *testing.T) {
	store := mocks.NewMockStore(t)
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.Local)
	stop := start.Add(9 * time.Hour)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{{
		ID: 1, Tag: "work", StartedAt: start, StoppedAt: &stop,
	}}, nil)

	out, err := execute(t, store, "balance")
	require.NoError(t, err)
	requireContains(t, out, "2026-10-01")
	requireContains(t, out, "balance +1h 0m")
}

func TestBalance_PrintsNegativeBalance(t *testing.T) {
	store := mocks.NewMockStore(t)
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.Local)
	stop := start.Add(6 * time.Hour)
	store.EXPECT().List(mock.AnythingOfType("internal.LogFilter")).Return([]internal.Entry{{
		ID: 1, Tag: "work", StartedAt: start, StoppedAt: &stop,
	}}, nil)

	out, err := execute(t, store, "balance")
	require.NoError(t, err)
	requireContains(t, out, "balance -2h 0m")
}
