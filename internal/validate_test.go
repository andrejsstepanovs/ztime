package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr(t time.Time) *time.Time { return &t }

func findIssue(issues []Issue, severity Severity, substr string) *Issue {
	for i := range issues {
		if issues[i].Severity == severity && containsStr(issues[i].Message, substr) {
			return &issues[i]
		}
	}
	return nil
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

func TestValidate_Clean(t *testing.T) {
	now := time.Now()
	entries := []Entry{
		{ID: 1, Tag: "work", StartedAt: now.Add(-6 * time.Hour), StoppedAt: ptr(now.Add(-time.Hour))},
		{ID: 2, Tag: "work", StartedAt: now.Add(-30 * time.Minute), StoppedAt: ptr(now)},
	}
	assert.Empty(t, Validate(entries))
}

func TestValidate_TooLong(t *testing.T) {
	now := time.Now()
	entries := []Entry{
		{ID: 1, Tag: "work", StartedAt: now.Add(-11 * time.Hour), StoppedAt: ptr(now)},
	}
	issues := Validate(entries)
	issue := findIssue(issues, SeverityError, "exceeds 10h")
	require.NotNil(t, issue, "expected an exceeds-10h error, got: %v", issues)
	assert.Equal(t, int64(1), issue.EntryID)
}

func TestValidate_ConsecutiveStarts(t *testing.T) {
	now := time.Now()
	entries := []Entry{
		{ID: 1, Tag: "work", StartedAt: now.Add(-3 * time.Hour)}, // open, never stopped
		{ID: 2, Tag: "work", StartedAt: now.Add(-1 * time.Hour)}, // another start
	}
	issues := Validate(entries)
	issue := findIssue(issues, SeverityError, "consecutive start")
	require.NotNil(t, issue, "expected a consecutive-start error, got: %v", issues)
	assert.Equal(t, int64(2), issue.EntryID)
}

func TestValidate_OverlappingClosedEntries(t *testing.T) {
	now := time.Now()
	firstStop := now.Add(-time.Hour)
	secondStop := now
	entries := []Entry{
		{ID: 2, Tag: "work", StartedAt: now.Add(-2 * time.Hour), StoppedAt: &secondStop},
		{ID: 1, Tag: "work", StartedAt: now.Add(-3 * time.Hour), StoppedAt: &firstStop},
	}

	issues := Validate(entries)
	issue := findIssue(issues, SeverityError, "consecutive start")
	require.NotNil(t, issue, "expected overlapping starts to be detected, got: %v", issues)
	assert.Equal(t, int64(2), issue.EntryID)
}

func TestValidate_StopBeforeStart(t *testing.T) {
	now := time.Now()
	stopped := now.Add(-time.Hour)
	entries := []Entry{{ID: 1, Tag: "work", StartedAt: now, StoppedAt: &stopped}}

	issues := Validate(entries)
	issue := findIssue(issues, SeverityError, "stop time must be after start time")
	require.NotNil(t, issue, "expected invalid stop time to be detected, got: %v", issues)
}

func TestValidate_OpenFromPreviousDay(t *testing.T) {
	// Start yesterday at 23:58 -- only 2 min long so won't trigger the >10h rule.
	yesterday := time.Now().AddDate(0, 0, -1)
	start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 58, 0, 0, time.Local)
	entries := []Entry{
		{ID: 1, Tag: "work", StartedAt: start}, // open, from yesterday
	}
	issues := Validate(entries)
	issue := findIssue(issues, SeverityWarning, "still open")
	require.NotNil(t, issue, "expected a still-open warning, got: %v", issues)
	assert.Equal(t, int64(1), issue.EntryID)
}

func TestValidate_SortsChronologically(t *testing.T) {
	// Entries inserted out of order -- ID 2 started before ID 1.
	now := time.Now()
	entries := []Entry{
		{ID: 1, Tag: "work", StartedAt: now.Add(-1 * time.Hour)}, // open, later
		{ID: 2, Tag: "work", StartedAt: now.Add(-3 * time.Hour)}, // open, earlier
	}
	issues := Validate(entries)
	// After sorting: ID2 first (open), ID1 next = consecutive start -> error on ID1
	issue := findIssue(issues, SeverityError, "consecutive start")
	require.NotNil(t, issue, "expected consecutive start error, got: %v", issues)
	assert.Equal(t, int64(1), issue.EntryID)
}

func TestValidate_DifferentTagsDoNotConflict(t *testing.T) {
	now := time.Now()
	entries := []Entry{
		{ID: 1, Tag: "work",  StartedAt: now.Add(-2 * time.Hour)},
		{ID: 2, Tag: "focus", StartedAt: now.Add(-1 * time.Hour)},
	}
	// Both open today -- different tags, no conflict
	assert.Empty(t, Validate(entries))
}

func TestValidate_MultipleIssues(t *testing.T) {
	now := time.Now()
	entries := []Entry{
		{ID: 1, Tag: "work", StartedAt: now.Add(-12 * time.Hour), StoppedAt: ptr(now.Add(-1 * time.Hour))},
		{ID: 2, Tag: "work", StartedAt: now.Add(-30 * time.Minute)},
		{ID: 3, Tag: "work", StartedAt: now.Add(-10 * time.Minute)},
	}
	issues := Validate(entries)
	assert.GreaterOrEqual(t, len(issues), 2)
}

func TestValidate_DailyWorkExceedsTenHours(t *testing.T) {
	location := time.Local
	entries := []Entry{
		{ID: 1, Tag: "work", StartedAt: time.Date(2026, 10, 1, 6, 0, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 1, 11, 30, 0, 0, location))},
		{ID: 2, Tag: "work", StartedAt: time.Date(2026, 10, 1, 12, 0, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 1, 17, 0, 0, 0, location))},
	}
	issue := findIssue(Validate(entries), SeverityError, "daily working time")
	require.NotNil(t, issue)
	assert.Contains(t, issue.Message, "10h 30m")
}

func TestValidate_BreakAfterMoreThanSixHours(t *testing.T) {
	location := time.Local
	entries := []Entry{
		{ID: 1, Tag: "work", StartedAt: time.Date(2026, 10, 1, 9, 0, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 1, 12, 0, 0, 0, location))},
		{ID: 2, Tag: "work", StartedAt: time.Date(2026, 10, 1, 12, 10, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 1, 15, 40, 0, 0, location))},
	}
	issue := findIssue(Validate(entries), SeverityError, "at least 30m")
	require.NotNil(t, issue)
	assert.Contains(t, issue.Message, "break parts must be at least 15m")
}

func TestValidate_UnclassifiedGapDoesNotCountAsBreak(t *testing.T) {
	location := time.Local
	entries := []Entry{
		{ID: 1, Tag: "work", StartedAt: time.Date(2026, 10, 1, 9, 0, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 1, 12, 30, 0, 0, location)), BreakNote: "commute"},
		{ID: 2, Tag: "work", StartedAt: time.Date(2026, 10, 1, 13, 0, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 1, 16, 30, 0, 0, location))},
	}
	issue := findIssue(Validate(entries), SeverityError, "qualifying breaks")
	require.NotNil(t, issue, "an unclassified gap must not count as a legal break")
}

func TestValidateStart_RequiresFifteenMinuteDeclaredBreak(t *testing.T) {
	location := time.Local
	stop := time.Date(2026, 10, 1, 15, 0, 0, 0, location)
	entries := []Entry{{
		ID: 1, Tag: "work", StartedAt: time.Date(2026, 10, 1, 9, 0, 0, 0, location),
		StoppedAt: &stop, BreakQualifies: true,
	}}
	issues := ValidateStart(entries, stop.Add(10*time.Minute), "work")
	issue := findIssue(issues, SeverityError, "at least 15m")
	require.NotNil(t, issue)
}

func TestValidateStart_BlocksAfterSixContinuousHoursWithoutDeclaredBreak(t *testing.T) {
	location := time.Local
	firstStop := time.Date(2026, 10, 1, 12, 0, 0, 0, location)
	secondStop := time.Date(2026, 10, 1, 15, 15, 0, 0, location)
	entries := []Entry{
		{ID: 1, Tag: "work", StartedAt: time.Date(2026, 10, 1, 9, 0, 0, 0, location), StoppedAt: &firstStop},
		{ID: 2, Tag: "work", StartedAt: time.Date(2026, 10, 1, 12, 15, 0, 0, location), StoppedAt: &secondStop},
	}
	issues := ValidateStart(entries, secondStop.Add(15*time.Minute), "work")
	issue := findIssue(issues, SeverityError, "take a qualifying break")
	require.NotNil(t, issue)
}

func TestValidateStart_AllowsDeclaredFifteenMinuteBreak(t *testing.T) {
	location := time.Local
	stop := time.Date(2026, 10, 1, 15, 0, 0, 0, location)
	entries := []Entry{{
		ID: 1, Tag: "work", StartedAt: time.Date(2026, 10, 1, 9, 0, 0, 0, location),
		StoppedAt: &stop, BreakQualifies: true,
	}}
	issues := ValidateStart(entries, stop.Add(15*time.Minute), "work")
	require.Nil(t, findIssue(issues, SeverityError, "break"))
}

func TestValidateStart_BlocksInsufficientDailyRest(t *testing.T) {
	location := time.Local
	stop := time.Date(2026, 10, 1, 22, 0, 0, 0, location)
	entries := []Entry{{ID: 1, Tag: "work", StartedAt: stop.Add(-time.Hour), StoppedAt: &stop}}
	issues := ValidateStart(entries, time.Date(2026, 10, 2, 8, 0, 0, 0, location), "work")
	require.NotNil(t, findIssue(issues, SeverityError, "at least 11h"))
}

func TestValidate_TwoFifteenMinuteBreaksAreValid(t *testing.T) {
	location := time.Local
	entries := []Entry{
		{ID: 1, Tag: "work", StartedAt: time.Date(2026, 10, 1, 9, 0, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 1, 11, 0, 0, 0, location)), BreakQualifies: true},
		{ID: 2, Tag: "work", StartedAt: time.Date(2026, 10, 1, 11, 15, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 1, 13, 15, 0, 0, location)), BreakQualifies: true},
		{ID: 3, Tag: "work", StartedAt: time.Date(2026, 10, 1, 13, 30, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 1, 16, 0, 0, 0, location))},
	}
	require.Nil(t, findIssue(Validate(entries), SeverityError, "qualifying breaks"))
}

func TestValidate_MoreThanNineHoursRequiresFortyFiveMinuteBreak(t *testing.T) {
	location := time.Local
	entries := []Entry{
		{ID: 1, Tag: "work", StartedAt: time.Date(2026, 10, 1, 7, 0, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 1, 12, 0, 0, 0, location))},
		{ID: 2, Tag: "work", StartedAt: time.Date(2026, 10, 1, 12, 30, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 1, 16, 31, 0, 0, location))},
	}
	issue := findIssue(Validate(entries), SeverityError, "at least 45m")
	require.NotNil(t, issue)
}

func TestValidate_ElevenHourRestPeriod(t *testing.T) {
	location := time.Local
	entries := []Entry{
		{ID: 1, Tag: "work", StartedAt: time.Date(2026, 10, 1, 18, 0, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 1, 22, 0, 0, 0, location))},
		{ID: 2, Tag: "work", StartedAt: time.Date(2026, 10, 2, 8, 0, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 2, 10, 0, 0, 0, location))},
	}
	issue := findIssue(Validate(entries), SeverityError, "at least 11h")
	require.NotNil(t, issue)
	assert.Equal(t, int64(2), issue.EntryID)
}

func TestValidate_FrameworkWorkingHours(t *testing.T) {
	location := time.Local
	entries := []Entry{{ID: 1, Tag: "work", StartedAt: time.Date(2026, 10, 1, 5, 59, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 1, 7, 0, 0, 0, location))}}
	issue := findIssue(Validate(entries), SeverityError, "06:00-23:00")
	require.NotNil(t, issue)
}

func TestValidate_SundayWork(t *testing.T) {
	location := time.Local
	entries := []Entry{{ID: 1, Tag: "work", StartedAt: time.Date(2026, 10, 4, 9, 0, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 4, 10, 0, 0, 0, location))}}
	issue := findIssue(Validate(entries), SeverityError, "Sunday")
	require.NotNil(t, issue)
}

func TestValidate_NationwideGermanPublicHoliday(t *testing.T) {
	location := time.Local
	entries := []Entry{{ID: 1, Tag: "work", StartedAt: time.Date(2026, 10, 3, 9, 0, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 3, 10, 0, 0, 0, location))}}
	issue := findIssue(Validate(entries), SeverityError, "public holiday")
	require.NotNil(t, issue)
}

func TestValidate_NonWorkTagSkipsWorkingTimeRules(t *testing.T) {
	location := time.Local
	entries := []Entry{{ID: 1, Tag: "hobby", StartedAt: time.Date(2026, 10, 4, 1, 0, 0, 0, location), StoppedAt: ptr(time.Date(2026, 10, 4, 5, 0, 0, 0, location))}}
	issues := Validate(entries)
	require.Nil(t, findIssue(issues, SeverityError, "Sunday"))
	require.Nil(t, findIssue(issues, SeverityError, "06:00-23:00"))
}
