package internal

import (
	"fmt"
	"sort"
	"time"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"

	workTag               = "work"
	regularDailyWork      = 8 * time.Hour
	maximumDailyWork      = 10 * time.Hour
	maximumContinuousWork = 6 * time.Hour
	minimumDailyRest      = 11 * time.Hour
	minimumBreakPart      = 15 * time.Minute
)

type Issue struct {
	Severity Severity
	EntryID  int64
	Message  string
}

func (i Issue) String() string {
	return fmt.Sprintf("[%s] entry #%d: %s", i.Severity, i.EntryID, i.Message)
}

// Validate checks structural data integrity for every tag and German working
// time rules for entries tagged "work". Entries are always evaluated in
// chronological order, independently of their database IDs.
func Validate(entries []Entry) []Issue {
	if len(entries) == 0 {
		return nil
	}

	sorted := append([]Entry(nil), entries...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].StartedAt.Before(sorted[j].StartedAt)
	})

	issues := validateEntryIntegrity(sorted)
	issues = append(issues, validateEventOrder(sorted)...)
	issues = append(issues, validateWorkingTime(sorted)...)
	return issues
}

func validateEntryIntegrity(entries []Entry) []Issue {
	var issues []Issue
	now := time.Now()
	for _, entry := range entries {
		if entry.StoppedAt != nil && !entry.StoppedAt.After(entry.StartedAt) {
			issues = append(issues, Issue{
				Severity: SeverityError,
				EntryID:  entry.ID,
				Message:  "stop time must be after start time",
			})
		}

		if entry.Duration() > maximumDailyWork {
			issues = append(issues, Issue{
				Severity: SeverityError,
				EntryID:  entry.ID,
				Message: fmt.Sprintf(
					"entry duration %s exceeds 10h -- likely a missing stop",
					fmtValidateDuration(entry.Duration())),
			})
		}

		if entry.IsOpen() && !sameCalendarDay(entry.StartedAt, now) {
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				EntryID:  entry.ID,
				Message: fmt.Sprintf(
					"entry started %s is still open (never stopped)",
					entry.StartedAt.Format("2006-01-02 15:04")),
			})
		}
	}
	return issues
}

type entryEvent struct {
	at      time.Time
	isStart bool
	entry   Entry
}

func validateEventOrder(entries []Entry) []Issue {
	var events []entryEvent
	for _, entry := range entries {
		events = append(events, entryEvent{at: entry.StartedAt, isStart: true, entry: entry})
		if entry.StoppedAt != nil {
			events = append(events, entryEvent{at: *entry.StoppedAt, entry: entry})
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].at.Equal(events[j].at) {
			return !events[i].isStart && events[j].isStart
		}
		return events[i].at.Before(events[j].at)
	})

	active := map[string]Entry{}
	var issues []Issue
	for _, event := range events {
		if event.isStart {
			if previous, ok := active[event.entry.Tag]; ok {
				issues = append(issues, Issue{
					Severity: SeverityError,
					EntryID:  event.entry.ID,
					Message: fmt.Sprintf(
						"consecutive start for tag %q -- entry #%d had not stopped yet (started %s)",
						event.entry.Tag, previous.ID, previous.StartedAt.Format("15:04")),
				})
			}
			active[event.entry.Tag] = event.entry
			continue
		}
		if current, ok := active[event.entry.Tag]; ok && current.ID == event.entry.ID {
			delete(active, event.entry.Tag)
		}
	}
	return issues
}

type workDay struct {
	date     time.Time
	entries  []Entry
	total    time.Duration
	firstID  int64
	first    time.Time
	last     time.Time
}

func validateWorkingTime(entries []Entry) []Issue {
	days := collectWorkDays(entries)
	var issues []Issue

	for _, day := range days {
		issues = append(issues, validateWorkDay(day)...)
	}
	for i := 1; i < len(days); i++ {
		rest := days[i].first.Sub(days[i-1].last)
		if rest < minimumDailyRest {
			issues = append(issues, Issue{
				Severity: SeverityError,
				EntryID:  days[i].firstID,
				Message: fmt.Sprintf(
					"only %s rest since previous workday ended at %s; at least 11h is required",
					fmtValidateDuration(rest), days[i-1].last.Format("2006-01-02 15:04")),
			})
		}
	}
	return issues
}

func collectWorkDays(entries []Entry) []workDay {
	byDate := map[string]*workDay{}
	var keys []string

	for _, entry := range entries {
		if entry.Tag != workTag || entry.StoppedAt == nil || !entry.StoppedAt.After(entry.StartedAt) {
			continue
		}
		key := entry.StartedAt.Format("2006-01-02")
		day := byDate[key]
		if day == nil {
			date := time.Date(entry.StartedAt.Year(), entry.StartedAt.Month(), entry.StartedAt.Day(), 0, 0, 0, 0, entry.StartedAt.Location())
			day = &workDay{date: date, firstID: entry.ID, first: entry.StartedAt, last: *entry.StoppedAt}
			byDate[key] = day
			keys = append(keys, key)
		}
		day.entries = append(day.entries, entry)
		day.total += entry.Duration()
		if entry.StartedAt.Before(day.first) {
			day.first = entry.StartedAt
			day.firstID = entry.ID
		}
		if entry.StoppedAt.After(day.last) {
			day.last = *entry.StoppedAt
		}
	}

	sort.Strings(keys)
	days := make([]workDay, 0, len(keys))
	for _, key := range keys {
		day := *byDate[key]
		sort.SliceStable(day.entries, func(i, j int) bool {
			return day.entries[i].StartedAt.Before(day.entries[j].StartedAt)
		})
		days = append(days, day)
	}
	return days
}

func validateWorkDay(day workDay) []Issue {
	var issues []Issue

	if day.total > maximumDailyWork {
		issues = append(issues, Issue{
			Severity: SeverityError,
			EntryID:  day.firstID,
			Message: fmt.Sprintf("daily working time %s exceeds the legal 10h maximum", fmtValidateDuration(day.total)),
		})
	} else if day.total > regularDailyWork {
		issues = append(issues, Issue{
			Severity: SeverityWarning,
			EntryID:  day.firstID,
			Message: fmt.Sprintf("daily working time %s exceeds the regular 8h and must be balanced", fmtValidateDuration(day.total)),
		})
	}

	if day.date.Weekday() == time.Sunday {
		issues = append(issues, Issue{Severity: SeverityError, EntryID: day.firstID, Message: "work on Sunday is prohibited without an approved exception"})
	} else if day.date.Weekday() == time.Saturday {
		issues = append(issues, Issue{Severity: SeverityError, EntryID: day.firstID, Message: "work on Saturday is outside Monday-to-Friday framework hours and requires approval"})
	}
	if isNationwideGermanHoliday(day.date) {
		issues = append(issues, Issue{Severity: SeverityError, EntryID: day.firstID, Message: "work on a nationwide German public holiday is prohibited without an approved exception"})
	}

	for _, entry := range day.entries {
		if beforeFrameworkStart(entry.StartedAt) || afterFrameworkEnd(*entry.StoppedAt) {
			issues = append(issues, Issue{
				Severity: SeverityError,
				EntryID:  entry.ID,
				Message:  "working time must stay within 06:00-23:00 Central European time unless an exception is approved",
			})
		}
	}
	issues = append(issues, validateContinuousWork(day.entries)...)

	requiredBreak := requiredBreakDuration(day.total)
	if requiredBreak > 0 {
		qualifying := qualifyingBreakDuration(day.entries)
		if qualifying < requiredBreak {
			issues = append(issues, Issue{
				Severity: SeverityError,
				EntryID:  day.firstID,
				Message: fmt.Sprintf(
					"%s of qualifying breaks recorded; at least %s is required for %s of work (break parts must be at least 15m)",
					fmtValidateDuration(qualifying), fmtValidateDuration(requiredBreak), fmtValidateDuration(day.total)),
			})
		}
	}
	return issues
}

func requiredBreakDuration(work time.Duration) time.Duration {
	if work > 9*time.Hour {
		return 45 * time.Minute
	}
	if work > 6*time.Hour {
		return 30 * time.Minute
	}
	return 0
}

func qualifyingBreakDuration(entries []Entry) time.Duration {
	var total time.Duration
	for i := 1; i < len(entries); i++ {
		if entries[i-1].StoppedAt == nil {
			continue
		}
		gap := entries[i].StartedAt.Sub(*entries[i-1].StoppedAt)
		if entries[i-1].BreakQualifies && gap >= minimumBreakPart {
			total += gap
		}
	}
	return total
}

func validateContinuousWork(entries []Entry) []Issue {
	if len(entries) == 0 {
		return nil
	}
	continuous := entries[0].Duration()
	for i := 1; i < len(entries); i++ {
		previous := entries[i-1]
		gap := entries[i].StartedAt.Sub(*previous.StoppedAt)
		if previous.BreakQualifies && gap >= minimumBreakPart {
			continuous = 0
		}
		continuous += entries[i].Duration()
		if continuous > maximumContinuousWork {
			return []Issue{{
				Severity: SeverityError,
				EntryID:  entries[i].ID,
				Message: fmt.Sprintf(
					"continuous work period %s exceeds 6h without a qualifying break of at least 15m",
					fmtValidateDuration(continuous)),
			}}
		}
	}
	if continuous > maximumContinuousWork {
		return []Issue{{
			Severity: SeverityError,
			EntryID:  entries[len(entries)-1].ID,
			Message: fmt.Sprintf(
				"continuous work period %s exceeds 6h without a qualifying break of at least 15m",
				fmtValidateDuration(continuous)),
		}}
	}
	return nil
}

// ValidateStart checks whether work may legally begin at the requested time.
func ValidateStart(entries []Entry, at time.Time, tag string) []Issue {
	if tag != workTag {
		return nil
	}
	candidate := Entry{ID: -1, Tag: tag, StartedAt: at}
	var issues []Issue
	if at.Weekday() == time.Saturday || at.Weekday() == time.Sunday {
		issues = append(issues, Issue{Severity: SeverityError, EntryID: candidate.ID, Message: "work may not start on a weekend without an approved exception"})
	}
	if isNationwideGermanHoliday(at) {
		issues = append(issues, Issue{Severity: SeverityError, EntryID: candidate.ID, Message: "work may not start on a nationwide German public holiday without an approved exception"})
	}
	if beforeFrameworkStart(at) || afterFrameworkEnd(at) {
		issues = append(issues, Issue{Severity: SeverityError, EntryID: candidate.ID, Message: "work may only start within 06:00-23:00 Central European time unless an exception is approved"})
	}

	var work []Entry
	for _, entry := range entries {
		if entry.Tag == workTag && entry.StartedAt.Before(at) {
			work = append(work, entry)
		}
	}
	sort.SliceStable(work, func(i, j int) bool { return work[i].StartedAt.Before(work[j].StartedAt) })
	if len(work) == 0 {
		return issues
	}

	last := work[len(work)-1]
	if last.StoppedAt == nil || last.StoppedAt.After(at) {
		issues = append(issues, Issue{Severity: SeverityError, EntryID: candidate.ID, Message: fmt.Sprintf("work entry #%d is still active", last.ID)})
		return issues
	}
	gap := at.Sub(*last.StoppedAt)
	if !sameCalendarDay(last.StartedAt, at) {
		if gap < minimumDailyRest {
			issues = append(issues, Issue{Severity: SeverityError, EntryID: candidate.ID, Message: fmt.Sprintf("only %s rest since entry #%d stopped; at least 11h is required", fmtValidateDuration(gap), last.ID)})
		}
		return issues
	}

	var worked, continuous time.Duration
	for i, entry := range work {
		if !sameCalendarDay(entry.StartedAt, at) || entry.StoppedAt == nil || entry.StoppedAt.After(at) {
			continue
		}
		worked += entry.Duration()
		if i == 0 {
			continuous = entry.Duration()
			continue
		}
		previous := work[i-1]
		breakGap := entry.StartedAt.Sub(*previous.StoppedAt)
		if previous.BreakQualifies && breakGap >= minimumBreakPart {
			continuous = 0
		}
		continuous += entry.Duration()
	}
	if worked >= maximumDailyWork {
		issues = append(issues, Issue{Severity: SeverityError, EntryID: candidate.ID, Message: fmt.Sprintf("already worked %s today; the 10h daily maximum has been reached", fmtValidateDuration(worked))})
	}
	if !(last.BreakQualifies && gap >= minimumBreakPart) && continuous >= maximumContinuousWork {
		issues = append(issues, Issue{Severity: SeverityError, EntryID: candidate.ID, Message: fmt.Sprintf("already worked %s continuously; take a qualifying break of at least 15m before starting again", fmtValidateDuration(continuous))})
	}
	if last.BreakQualifies && gap < minimumBreakPart {
		issues = append(issues, Issue{Severity: SeverityError, EntryID: candidate.ID, Message: fmt.Sprintf("declared break is only %s; a qualifying break must last at least 15m", fmtValidateDuration(gap))})
	}
	return issues
}

func beforeFrameworkStart(t time.Time) bool {
	return t.Hour() < 6
}

func afterFrameworkEnd(t time.Time) bool {
	return t.Hour() >= 23 && (t.Hour() > 23 || t.Minute() > 0 || t.Second() > 0)
}

func isNationwideGermanHoliday(date time.Time) bool {
	year, month, day := date.Date()
	fixed := [][2]int{{1, 1}, {5, 1}, {10, 3}, {12, 25}, {12, 26}}
	for _, holiday := range fixed {
		if int(month) == holiday[0] && day == holiday[1] {
			return true
		}
	}
	easter := easterSunday(year, date.Location())
	for _, offset := range []int{-2, 1, 39, 50} {
		if sameCalendarDay(date, easter.AddDate(0, 0, offset)) {
			return true
		}
	}
	return false
}

func easterSunday(year int, location *time.Location) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, location)
}

func sameCalendarDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func fmtValidateDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}
