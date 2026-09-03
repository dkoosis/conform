# conform

★ every dkoosis repo runs the same SDLC because one checker says so, not because someone copied a template.

conform is the fleet's contract, checked instead of copied. A repo adopts an
improvement by bumping a pin, not by re-pasting a Makefile; a rule that would
be too noisy to hard-fail is deleted rather than warned. This file is the
destination — the ★ line above, the ordered milestones below, and a link to
every resource the project has. Hand-written; agents don't edit it unprompted.
Progress is never written here — it derives at read time from the bd DAG
joined against the epic ids below.

## Milestones

1. Three surfaces, dogfooded — in-repo files, machine-local wiring, and
   GitHub-side settings, with conform's own CI running conform → `cfm-1e1`
2. The fleet adopts it — all 13 Go repos call conform from `make check` and
   go red when the pin is removed → `sd-th5`
3. A rule change propagates as a pin bump — one deliberate change turns the
   fleet red, then green again as 13 pin-bump PRs merge → `sd-th5.22`
4. Scaffolding a new repo shares the checker's renderer, so what `init` emits
   and what the checker demands cannot drift → `sd-th5.21`

## Non-goals

- Being a template tool. Templates drift the moment they are copied; that is
  the problem this exists to replace.
- Soft-fail or warning levels. A rule earns hard-fail or it is deleted.
- Per-repo variants of the contract. A repo declares an exception by rule id
  in `conform.json`, with a reason — it does not fork the rule.
- Formatting or style opinions already owned by `.golangci.yml`.

## Resources

- `README.md` — the three surfaces and the principles behind them
- `internal/checks/` — Surface 1 rules, one file per rule, id in `checks.go`
- `internal/values/` — `conform.json`: profile and declared exceptions
- `conform.json` — this repo's own declaration, checked by its own rules
- Reference repo: [ferret](https://github.com/dkoosis/ferret)
- Fleet plan: `~/Projects/dk/Project/cc-plugins/plans/fleet-scaffolding.md`
- Work: `bd list` here (`cfm-*`); the fleet-side epics live in sdlc (`sd-th5*`)
