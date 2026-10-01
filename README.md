# ztime

`ztime` is a command-line work time tracker. It records work sessions, enforces
German working-time law, and produces reports for company time-tracking tools.

## Requirements

- Go 1.21 or later
- SQLite (provided by the pure-Go `modernc.org/sqlite` driver, no CGo required)

## Install

```sh
git clone https://github.com/astepanovs/ztime
cd ztime
task build        # produces bin/ztime
```

Copy `bin/ztime` to a directory on your `PATH`.

## Database

`ztime` stores data in `~/.local/share/time/time.db`. The tool creates this
file on first run. Migrations run automatically at startup.

## Quick start

```sh
ztime start                              # start a work entry
ztime task "Investigated checkout alert" # log what you worked on
ztime stop --break --reason "lunch"      # stop and declare a qualifying break
ztime start                              # start again after the break
ztime stop                               # stop at end of day
ztime status                             # view today's timeline
```

## Commands

### start

Start a work entry.

```sh
ztime start
ztime start --at 09:00
ztime start --task "Morning monitoring"
```

| Flag | Short | Description |
|------|-------|-------------|
| `--tag` | `-t` | Entry tag. Default: `work`. |
| `--at` | `-a` | Override start time. Accepts `HH:MM` or `YYYY-MM-DDTHH:MM`. |
| `--task` | `-k` | Record an initial task at the same time as the start. |
| `--force` | `-f` | Bypass a working-time rule violation. Use only for approved exceptions. |

`ztime start` is blocked when the action would violate a working-time rule. For
example, the command is blocked when 6 continuous hours have passed without a
declared break. Use `--force` only when a manager has approved an exception.

### stop

Stop the current open work entry.

```sh
ztime stop
ztime stop --break --reason "lunch"
ztime stop --at 17:30 --reason "commute home"
```

| Flag | Short | Description |
|------|-------|-------------|
| `--tag` | `-t` | Tag of the open entry to stop. Default: `work`. |
| `--at` | `-a` | Override stop time. |
| `--break` | `-b` | Declare that the following gap is a legally qualifying rest break. |
| `--reason` | `-r` | Describe the gap that follows the stop. |

`ztime stop` is always permitted. The tool reports violations after the stop
but does not prevent recording the stop.

A gap between entries counts as a statutory break only when you pass `--break`.
A qualifying break must last at least 15 minutes. Commuting to the office is
not a break. Do not use `--break` for commutes or non-rest gaps.

### status

Show today's timeline, including tasks nested under each work entry.

```sh
ztime status
ztime status --no-tasks
```

```text
  running  [#3] work  started 12:55  (4h 32m elapsed)

today  2026-10-01
---
  #1  work      09:00 - 09:35  35m
       └─ 09:01  [task #1] Investigated checkout alert
       ·  gap    09:35 - 10:00  (25m)  // commute
  #2  work      10:00 - 12:20  2h 20m
       ·  break  12:20 - 12:55  (35m)  // lunch
  #3  work      12:55 - open   4h 32m
       └─ 14:00  [task #2] Reviewed PR
---
  work      7h 27m
```

### log

List work entries. Default filter: today.

```sh
ztime log
ztime log --date yesterday
ztime log --week last
ztime log --tag work --json
```

| Flag | Short | Description |
|------|-------|-------------|
| `--date` | `-d` | Filter by date: `today`, `yesterday`, or `YYYY-MM-DD`. |
| `--week` | `-w` | Filter by week: `this`, `last`, or `YYYY-Www`. |
| `--tag` | `-t` | Filter by tag. |
| `--json` | `-j` | Output as JSON. |

### task

Record a free-form note about what you worked on. The note attaches to the
active work entry. When no entry is active, the note attaches to the most
recent stopped entry and receives that entry's stop time.

```sh
ztime task "Investigated checkout latency"
ztime task --at 14:00 "Reviewed PR #123"
```

| Flag | Short | Description |
|------|-------|-------------|
| `--at` | `-a` | Override the activity time. |

### tasks

List task notes. Default filter: today.

```sh
ztime tasks
ztime tasks --date yesterday
ztime tasks --week this
```

### edit

Edit a work entry by its ID. The ID appears in `ztime log` and `ztime status`.

```sh
ztime edit 1 --start 09:05
ztime edit 1 --break-note "lunch"
ztime edit 1 --break
```

| Flag | Short | Description |
|------|-------|-------------|
| `--tag` | `-t` | Change the tag. |
| `--start` | `-s` | Change the start time. |
| `--stop` | `-S` | Change the stop time. |
| `--note` | `-n` | Change the entry note. |
| `--break-note` | `-b` | Change the break reason. |
| `--break` | | Set or clear the qualifying-break flag. |

### delete

Delete a work entry by its ID.

```sh
ztime delete 42
```

### balance

Show cumulative plus or minus working time. Days with no work entries are
ignored. Logged weekend and public-holiday work has a zero-hour target, so
all recorded work on those days counts as positive balance.

```sh
ztime balance
ztime balance --week this
```

```text
2026-10-01  worked 9h 0m  target 8h 0m  balance +1h 0m
2026-10-02  worked 7h 0m  target 8h 0m  balance -1h 0m
---
worked 16h 0m  target 16h 0m  balance +0m
```

| Flag | Short | Description |
|------|-------|-------------|
| `--date` | `-d` | Scope to a date. |
| `--week` | `-w` | Scope to a week. |

### report

List work periods formatted for transfer into a company time-tracking tool.
Open entries are marked as not ready.

```sh
ztime report
ztime report --date yesterday
ztime report --week this
```

```text
2026-10-01
  09:00 - 12:00  3h 0m
  12:30 - 17:30  5h 0m
  total  8h 0m
```

### validate

Check all entries for rule violations. The command exits with a non-zero code
when violations exist.

```sh
ztime validate
ztime validate --week this
```

```text
[error]   entry #3: continuous work period 6h 5m exceeds 6h without a qualifying break
[warning] entry #1: daily working time 8h 30m exceeds the regular 8h and must be balanced
```

### backfill

Start a new entry at the exact stop time of the previous entry. Use this when
you logged a stop but never left. The record shows no gap between entries.

```sh
ztime backfill
ztime backfill --note "never actually left"
```

### undo

Reverse the most recent action for a tag.

- If the last action was a start, `undo` deletes the open entry.
- If the last action was a stop, `undo` clears the stop time and reopens the entry.

```sh
ztime undo
ztime undo --tag lunch
```

### rules

Print all working-time rules with law references and examples.

```sh
ztime rules
```

The output uses `ALLOWED`, `FORBIDDEN`, and `REQUIRED` prefixes. LLM agents
can parse the output to decide which actions are permitted.

## Break classification

Every gap between work entries has a type:

| Type | Flag | Counts as legal break |
|------|------|-----------------------|
| `break` | `--break` passed to `stop` | Yes, when gap >= 15m |
| `gap` | no `--break` flag | No |

The status timeline shows the type:

```text
·  gap    09:35 - 10:00  (25m)  // commute
·  break  12:20 - 12:55  (35m)  // lunch
```

## Legal guardrails

`ztime start` is blocked when starting work would violate these rules:

- The 10-hour daily maximum is reached.
- Six continuous hours have passed without a qualifying break.
- A declared break is shorter than 15 minutes.
- Less than 11 hours have passed since the previous workday ended.
- The start time is outside 06:00-23:00 CET.
- The start time is on a Sunday or German public holiday.

`ztime stop` reports violations but always permits the stop.

Use `--force` only when a manager has approved an exception.

## Development

```sh
task          # build, lint, test
task test     # run all tests
task lint     # run golangci-lint
task mocks    # regenerate mocks
task ci       # full pipeline
```

The project uses [Task](https://taskfile.dev) as a build tool. Install it with:

```sh
brew install go-task
```

### Package layout

| Package | Purpose |
|---------|---------|
| `internal/` | `Entry`, `Activity`, `Store` interface, validation logic, balance calculation |
| `db/` | SQLite implementation of `Store`, migrations |
| `cmd/` | Cobra commands |
| `mocks/` | Mockery-generated mocks for `Store` |
| `e2e/` | End-to-end tests against a real SQLite database |

Mocks are generated with [mockery](https://vektra.github.io/mockery/). The
`Store` interface in `internal/store.go` carries the `//go:generate` directive.

### Running tests

```sh
task test          # all packages
task test:unit     # cmd package only
task test:e2e      # e2e package only
task test:coverage # coverage report
```
