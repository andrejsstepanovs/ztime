package internal

import (
	"sort"
	"time"
)

const standardWorkDay = 8 * time.Hour

type DailyBalance struct {
	Date    time.Time
	Worked  time.Duration
	Target  time.Duration
	Balance time.Duration
}

type BalanceReport struct {
	Days    []DailyBalance
	Worked  time.Duration
	Target  time.Duration
	Balance time.Duration
}

// CalculateBalance compares recorded work against an eight-hour target on
// logged ordinary weekdays. Unlogged days are ignored. Logged weekends and
// nationwide German public holidays have a zero target, so their work counts
// entirely as positive balance.
func CalculateBalance(entries []Entry) BalanceReport {
	workedByDate := map[string]time.Duration{}
	dateByKey := map[string]time.Time{}

	for _, entry := range entries {
		if entry.Tag != workTag || entry.Duration() <= 0 {
			continue
		}
		key := entry.StartedAt.Format("2006-01-02")
		workedByDate[key] += entry.Duration()
		dateByKey[key] = time.Date(
			entry.StartedAt.Year(), entry.StartedAt.Month(), entry.StartedAt.Day(),
			0, 0, 0, 0, entry.StartedAt.Location(),
		)
	}

	keys := make([]string, 0, len(workedByDate))
	for key := range workedByDate {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	report := BalanceReport{}
	for _, key := range keys {
		date := dateByKey[key]
		worked := workedByDate[key]
		target := targetForLoggedDay(date)
		day := DailyBalance{
			Date:    date,
			Worked:  worked,
			Target:  target,
			Balance: worked - target,
		}
		report.Days = append(report.Days, day)
		report.Worked += worked
		report.Target += target
		report.Balance += day.Balance
	}
	return report
}

func targetForLoggedDay(date time.Time) time.Duration {
	if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday || isNationwideGermanHoliday(date) {
		return 0
	}
	return standardWorkDay
}
