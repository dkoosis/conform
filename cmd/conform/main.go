// Command conform checks a repo against the dkoosis fleet SDLC contract.
//
// Three surfaces, because the three kinds of state live in three places:
//
//	conform          in-repo files (Makefile verbs, lint core, CI shape, bd config)
//	conform --local  machine wiring CI can't see (hooksPath, hooks, dolt remote)
//	conform --fleet  GitHub-side settings (protection, labels, merge policy)
//
// Failures name file, rule, and repair command. There is no soft-fail: a rule
// too noisy to hard-fail gets deleted, not warned.
package main

import (
	"errors"
	"fmt"
	"os"
)

// errNotImplemented marks surfaces that exist in the contract but not yet in
// code (cfm-1e1.2..4).
var errNotImplemented = errors.New("not implemented yet")

// errUsage marks a bad invocation; usage has already been printed.
var errUsage = errors.New("bad usage")

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "conform:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	mode := "check"
	if len(args) > 0 {
		switch args[0] {
		case "--local", "--fleet":
			mode = args[0][2:]
		case "-h", "--help", "help":
			usage()
			return nil
		default:
			usage()
			return fmt.Errorf("%w: unknown argument %q", errUsage, args[0])
		}
	}
	return fmt.Errorf("surface %q: %w", mode, errNotImplemented)
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: conform [--local | --fleet]

  (no flag)  check in-repo files against the fleet contract
  --local    check machine-local wiring (hooksPath, hooks, dolt remote)
  --fleet    check GitHub-side settings across the fleet (gh auth)
`)
}
