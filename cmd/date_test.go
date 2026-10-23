package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/astepanovs/ztime/internal"
	"github.com/astepanovs/ztime/mocks"
)

func TestParseTimeArgWithDate(t *testing.T) {
	ref := time.Date(2026, 10, 23, 13, 20, 45, 123, time.Local)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, time.Local)
	tests := []struct {
		name, clock, date string
		want              time.Time
		wantErr           string
	}{
		{name: "unchanged default", want: ref},
		{name: "clock only", clock: "18:00", want: time.Date(2026, 10, 23, 18, 0, 0, 0, time.Local)},
		{name: "explicit date", clock: "18:00", date: "2026-10-22", want: time.Date(2026, 10, 22, 18, 0, 0, 0, time.Local)},
		{name: "today", clock: "18:00", date: "today", want: today},
		{name: "yesterday", clock: "18:00", date: "yesterday", want: today.AddDate(0, 0, -1)},
		{name: "date only keeps clock", date: "2026-10-22", want: ref.AddDate(0, 0, -1)},
		{name: "full timestamp", clock: "2026-10-22T18:00:05", want: time.Date(2026, 10, 22, 18, 0, 5, 0, time.Local)},
		{name: "matching timestamp", clock: "2026-10-22T18:00", date: "2026-10-22", want: time.Date(2026, 10, 22, 18, 0, 0, 0, time.Local)},
		{name: "conflicting timestamp", clock: "2026-10-23T18:00", date: "2026-10-22", wantErr: "conflicts with --date"},
		{name: "invalid date", clock: "18:00", date: "2026-02-30", wantErr: "--date: unrecognised date"},
		{name: "invalid clock", clock: "25:00", date: "yesterday", wantErr: "unrecognised time format"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimeArgWithDate(tt.clock, tt.date, ref)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEdit_Date(t *testing.T) {
	start := time.Date(2026, 10, 22, 13, 20, 0, 0, time.Local)
	stop := time.Date(2026, 10, 23, 18, 0, 0, 0, time.Local)
	correctStop := stop.AddDate(0, 0, -1)
	newStart := start.AddDate(0, 0, -1)
	newStop := correctStop.AddDate(0, 0, -1)
	tests := []struct {
		name string
		args []string
		want internal.EditParams
	}{
		{
			name: "correct stop date only",
			args: []string{"--stop", "18:00", "-d", "2026-10-22"},
			want: internal.EditParams{StoppedAt: &correctStop},
		},
		{
			name: "move both supplied times",
			args: []string{"--start", "13:20", "--stop", "18:00", "--date", "2026-10-21"},
			want: internal.EditParams{StartedAt: &newStart, StoppedAt: &newStop},
		},
		{
			name: "existing stop day without date",
			args: []string{"--stop", "18:00"},
			want: internal.EditParams{StoppedAt: &stop},
		},
		{
			name: "change start without changing stop",
			args: []string{"--start", "13:20", "--date", "2026-10-21"},
			want: internal.EditParams{StartedAt: &newStart},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := mocks.NewMockStore(t)
			entry := &internal.Entry{ID: 35, Tag: "work", StartedAt: start, StoppedAt: &stop}
			store.EXPECT().Get(int64(35)).Return(entry, nil)
			store.EXPECT().Edit(int64(35), tt.want).Return(entry, nil)
			_, err := execute(t, store, append([]string{"edit", "35"}, tt.args...)...)
			assert.NoError(t, err)
		})
	}
}

func TestDate_InvalidArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "start invalid date", args: []string{"start", "-d", "invalid"}, want: "unrecognised date"},
		{name: "stop invalid date", args: []string{"stop", "-d", "invalid"}, want: "unrecognised date"},
		{name: "start conflicting date", args: []string{"start", "-d", "2026-10-22", "--at", "2026-10-23T09:00"}, want: "conflicts with --date"},
		{name: "stop conflicting date", args: []string{"stop", "-d", "2026-10-22", "--at", "2026-10-23T18:00"}, want: "conflicts with --date"},
		{name: "edit date only", args: []string{"edit", "35", "-d", "yesterday"}, want: "--date requires --start or --stop"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := execute(t, mocks.NewMockStore(t), tt.args...)
			assert.ErrorContains(t, err, tt.want)
		})
	}
}
