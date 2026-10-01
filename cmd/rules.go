package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const rulesText = `# Working Time Rules — Zalando Germany
# Source: German Working Time Act (Arbeitszeitgesetz, ArbZG)
# These rules apply to entries tagged "work".

## Daily working time

ALLOWED   Regular daily working time is 8 hours (§ 3 ArbZG).
ALLOWED   Daily working time may be extended to a maximum of 10 hours for urgent operational reasons (§ 3 ArbZG).
REQUIRED  Extended hours (>8h) must be balanced out within a reasonable period (§ 3 ArbZG).
FORBIDDEN Daily working time must not exceed 10 hours under any circumstances (§ 3 ArbZG).

Example — valid:
  09:00 start, 12:00 stop, 12:30 start, 17:30 stop → 8h worked

Example — invalid:
  08:00 start, 19:30 stop, 30m declared break → 11h worked → FORBIDDEN


## Breaks (§ 4 ArbZG)

REQUIRED  When daily working time exceeds 6 hours: a break of at least 30 minutes.
ALLOWED   The 30-minute break may be split into two parts of at least 15 minutes each.
REQUIRED  When daily working time exceeds 9 hours: a break of at least 45 minutes.
ALLOWED   The 45-minute break may be split into parts of at least 15 minutes each.
FORBIDDEN Working more than 6 hours at a time without a qualifying break.
FORBIDDEN Counting a break that is shorter than 15 minutes toward the required break total.
FORBIDDEN Counting commuting, personal errands, or any non-rest gap as a statutory break.

A break is legally qualifying only when it is explicitly declared and lasts at least 15 minutes.
Use: ztime stop --break --reason "lunch"

Example — valid (>6h, one 30m break):
  09:00 start
  12:00 stop --break --reason "lunch"
  12:30 start
  17:00 stop
  → 7h worked, 30m declared break ✓

Example — valid (>6h, two 15m breaks):
  09:00 start
  11:00 stop --break --reason "coffee"    (15m)
  11:15 start
  13:15 stop --break --reason "walk"      (15m)
  13:30 start
  16:00 stop
  → 6h 30m worked, 30m qualifying breaks ✓

Example — invalid:
  09:00 start
  15:05 stop
  → 6h 5m continuous work with no qualifying break → FORBIDDEN

Example — invalid:
  09:00 start
  15:00 stop --break --reason "lunch"     (10m gap)
  15:10 start
  → 10m declared break < 15m minimum → FORBIDDEN

Example — invalid:
  09:00 start
  09:35 stop --reason "commute"           (no --break flag)
  10:00 start
  → Gap is not declared as a break; does not count toward required break total
  → If total work later exceeds 6h without a qualifying break → FORBIDDEN


## Rest period (§ 5 ArbZG)

REQUIRED  At least 11 uninterrupted hours must pass between the end of one workday and the start of the next.
FORBIDDEN Starting work before 11 hours have elapsed since the previous workday ended.

Example — valid:
  Day 1: stop at 18:00 → Day 2: start no earlier than 05:00

Example — invalid:
  Day 1: stop at 22:00 → Day 2: start at 08:00 → only 10h rest → FORBIDDEN


## Framework working hours (Zalando policy)

REQUIRED  Individual working time must be distributed between 06:00 and 23:00 Central European Time (CET), Monday to Friday.
ALLOWED   Exceptions for business trips or urgent operational requirements previously agreed with lead.
FORBIDDEN Starting or performing work outside 06:00–23:00 CET without prior lead approval.


## Weekends and public holidays (§ 9 ArbZG)

FORBIDDEN Work on Sundays.
FORBIDDEN Work on nationwide German public holidays:
            New Year's Day (1 Jan), Labour Day (1 May), German Unity Day (3 Oct),
            Christmas Day (25 Dec), Boxing Day (26 Dec),
            Good Friday, Easter Monday, Ascension Day, Whit Monday.
ALLOWED   Exceptions only with Works Council approval and, for Sundays and public holidays,
          also with formal authority approval (§ 13 ArbZG).
ALLOWED   Saturday work with lead and Works Council approval (Zalando policy).
ALLOWED   Work on Saturday or Sunday logged as a work entry counts entirely as positive balance
          because the daily target for those days is zero.


## Break classification (ztime-specific rule)

REQUIRED  A gap between work entries must be explicitly declared with --break to count as a legal rest break.
REQUIRED  Commuting to the regular office is not working time and must not be declared with --break (§ 2 ArbZG).
REQUIRED  Passive business travel (resting, not working) must not be declared with --break.
ALLOWED   Active business travel (actively working while travelling) counts as working time (§ 2 ArbZG).
ALLOWED   Driving a car on a business trip counts as working time (§ 2 ArbZG).

Use: ztime stop --reason "commute"          # gap, not a break
Use: ztime stop --break --reason "lunch"    # qualifying break


## Flexibility corridor (Zalando policy)

ALLOWED   Accumulating up to +25h over the regular weekly working time.
ALLOWED   Accumulating up to -10h under the regular weekly working time.
REQUIRED  Balance beyond +15h or -5h triggers a notification; review priorities with lead.
REQUIRED  Balance beyond +25h or below -10h requires lead approval to continue.
ALLOWED   Plus hours may be balanced as time-off in agreement with lead.
FORBIDDEN Letting the balance drift beyond ±corridor limits without lead alignment.


## Working time definition (§ 2 ArbZG)

ALLOWED   Working time includes: day-to-day tasks, work-related meetings, required training,
          work-related events (panel talks, fireside chats, earnings calls), active business travel.
FORBIDDEN Recording as working time: break time, commuting to the regular office,
          voluntary social events (Sunset Socials, after-work drinks), personal activities,
          passive business travel.
`

func newRulesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rules",
		Short: "Print all working-time rules and examples",
		Long:  `Rules prints the legal and policy constraints that ztime enforces, with references to the German Working Time Act (ArbZG) and Zalando policy where available. Output is intended to be both human-readable and machine-parseable by LLM agents.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(cmd.OutOrStdout(), rulesText)
			return nil
		},
	}
}
