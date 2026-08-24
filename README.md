# `conform`

> **One sentence:** checks a dkoosis fleet repo against the shared SDLC contract, so drift is a finding instead of a five-repo grep.

`conform` **checks a repo's Makefile verbs, lint core, CI shape, hooks, and bd config against one fleet contract** for **dkoosis repo maintainers**, without **hand-copying templates that silently drift the moment one repo edits its own copy**.

## See it work

```console
$ conform
conform.json: values-file: values file missing — conform needs the repo's profile (tool|lib) and declared exceptions
    fix: create conform.json (or docs/conform.json): {"profile": "tool", "exceptions": []}
conform: 1 finding(s)

$ echo '{"profile": "tool", "exceptions": []}' > conform.json

$ conform
conform: ok
```

Every finding names the file, the rule, and the fix — never a bare pass/fail.

## Install

```console
go install github.com/dkoosis/conform/cmd/conform@latest
```

A repo can also pin it as a Go tool dependency (`go get -tool
github.com/dkoosis/conform/cmd/conform`) and run it as `go tool conform` —
the fleet's own repos do this, so `make check` fails the build the same way
CI will.

## Common workflows

### Check the in-repo contract

```console
$ conform
conform: ok
```

Runs on every `make check`: Makefile's four-verb contract, `.golangci.yml`'s
lint floor, CI calling `make check`, bd config, hooks shape, PR template,
`.sandbox/lib` against the canonical copy. Hard-fail only — a rule too noisy
to hard-fail gets deleted, not downgraded to a warning.

### Check machine-local wiring

```console
$ conform --local
```

Sees what CI can't: `core.hooksPath`, whether the hooks are actually
executable, the dolt remote. Run at session start, not in CI.

### Check GitHub-side settings across the fleet

```console
$ conform --fleet
```

Branch protection, labels, merge policy, PR template — swept across every
repo in the fleet roster, not just the one you're standing in.

### Declare a reasoned exception

```console
$ cat conform.json
{
  "profile": "tool",
  "exceptions": [
    {"rule": "no-git-ops", "reason": "loto never performs git operations by design"}
  ]
}
```

An exception is the one sanctioned way to suppress a finding — every entry
carries a rule id and a reason; an unreasoned exception is a hard error, not
a warning.

### Resync the sandbox library

```console
$ conform sandbox sync
conform: synced .sandbox/lib/lib-setup.sh
conform: 1 file(s) synced
```

Rewrites `.sandbox/lib` from the canonical copy conform ships. The one
command that writes instead of reports.

## How it works

| surface | sees | runs |
|---|---|---|
| `conform` | in-repo files: Makefile verbs, `.golangci.yml` core, single lint pin, CI-calls-make-check, bd config, hooks, PR template, sandbox lib | every `make check`, hard-fail |
| `conform --local` | machine wiring CI can't see: `core.hooksPath`, hooks executable, dolt remote | `make doctor`, session start |
| `conform --fleet` | GitHub: branch protection, labels, merge policy, PR template | promulgation + pin-bump sweeps |

### The important rule

**The fleet contract is authoritative; every repo's copy is checked against it, never hand-copied from it.** Improvements to the contract propagate as a pin bump plus a `conform` run — not as a template someone forgot to re-paste into five repos.

## Configuration

Every repo declares its `conform.json` at its root, or at `docs/conform.json`
for a repo that keeps every doc but `README.md` under `docs/` — conform
tries both locations, root first:

```json
{
  "profile": "tool",
  "exceptions": []
}
```

`profile` is `tool` (the full SDLC surface, deploy included, beads required)
or `lib` (the same contract minus the deploy verb, beads optional) — `lib`
is not an exception, it's a different profile.

## Development

```console
git clone https://github.com/dkoosis/conform.git
cd conform
make check
```

Dogfooded: this repo's own CI runs `conform` on itself, and `make check`
includes `selfcheck` for exactly that reason.
