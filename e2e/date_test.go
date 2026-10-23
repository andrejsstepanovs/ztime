package e2e

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestE2E_DateArguments(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	tests := []struct {
		name string
		date string
		day  string
	}{
		{name: "explicit date", date: "2026-10-22", day: "2026-10-22"},
		{name: "yesterday", date: "yesterday", day: yesterday},
		{name: "today", date: "today", day: now.Format("2006-01-02")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := harness(t)
			_, err := run("start", "-d", tt.date, "--at", "13:20", "--force", "--task", "afternoon work")
			if !assert.NoError(t, err) {
				return
			}
			_, err = run("stop", "--date", tt.date, "--at", "18:00")
			if !assert.NoError(t, err) {
				return
			}
			out, err := run("log", "-d", tt.date, "--json")
			assert.NoError(t, err)
			type entryTimes struct {
				StartedAt time.Time  `json:"started_at"`
				StoppedAt *time.Time `json:"stopped_at"`
			}
			var entries []entryTimes
			if !assert.NoError(t, json.Unmarshal([]byte(out), &entries)) || !assert.Len(t, entries, 1) {
				return
			}
			start, err := time.ParseInLocation("2006-01-02T15:04", tt.day+"T13:20", time.Local)
			assert.NoError(t, err)
			stop := start.Add(4*time.Hour + 40*time.Minute)
			assert.Equal(t, entryTimes{StartedAt: start, StoppedAt: &stop}, entries[0])

			// Reproduce a stop recorded on the wrong day, then correct it.
			_, err = run("edit", "1", "--stop", stop.AddDate(0, 0, 1).Format("2006-01-02T15:04"))
			assert.NoError(t, err)
			_, err = run("edit", "1", "--stop", "18:00", "-d", tt.date)
			assert.NoError(t, err)
			out, err = run("log", "-d", tt.date)
			assert.NoError(t, err)
			assert.Contains(t, out, "13:20 - 18:00  4h 40m")
		})
	}
}
