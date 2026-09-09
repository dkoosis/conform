// Command conform checks a repo against the dkoosis fleet SDLC contract.
//
// Three surfaces, because the three kinds of state live in three places:
//
//	conform          in-repo files (Makefile verbs, lint core, CI shape, bd config)
//	conform --local  machine wiring CI can't see (hooksPath, hooks, dolt remote)
//	conform --fleet  GitHub-side settings (protection, labels, merge policy)
//
// One subcommand writes rather than reports:
//
//	conform sandbox sync   rewrite .sandbox/lib from the canonical copy
//
// Failures name file, rule, and repair command. There is no soft-fail: a rule
// too noisy to hard-fail gets deleted, not warned.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/dkoosis/conform/internal/checks"
	"github.com/dkoosis/conform/internal/sandbox"
)

// errUsage marks a bad invocation; usage has already been printed.
var errUsage = errors.New("bad usage")

// errFindings marks a completed run that found contract violations — the
// hard-fail exit, distinct from conform itself breaking.
var errFindings = errors.New("contract violations")

func main() {
	if err := run(os.Args[1:]); err != nil {
		if !errors.Is(err, errFindings) {
			fmt.Fprintln(os.Stderr, "conform:", err)
		}
		os.Exit(1)
	}
}

func run(args []string) error {
	mode := "check"
	if len(args) > 0 {
		switch args[0] {
		case "--local", "--fleet":
			mode = args[0][2:]
		case "sandbox":
			return runSandbox(args[1:])
		case "-h", "--help", "help":
			usage()
			return nil
		default:
			usage()
			return fmt.Errorf("%w: unknown argument %q", errUsage, args[0])
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	var findings []checks.Finding
	switch mode {
	case "local":
		findings = checks.RunLocal(context.Background(), dir)
	case "fleet":
		findings, err = checks.RunFleet(context.Background())
		if err != nil {
			return err
		}
	default:
		findings = checks.Run(dir)
	}
	if len(findings) == 0 {
		fmt.Println("conform: ok")
		return nil
	}
	for _, f := range findings {
		fmt.Println(f)
	}
	fmt.Printf("conform: %d finding(s)\n", len(findings))
	return errFindings
}

// runSandbox implements the one write conform performs: rewriting a repo's
// .sandbox/lib from the canonical copy. Checking reports drift; this repairs
// it, so the repair command a finding prints is a command that exists.
func runSandbox(args []string) error {
	if len(args) != 1 || args[0] != "sync" {
		usage()
		return fmt.Errorf("%w: conform sandbox takes exactly one subcommand, sync", errUsage)
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	changed, err := sandbox.Sync(dir)
	if err != nil {
		return err
	}
	if len(changed) == 0 {
		fmt.Println("conform: sandbox library already canonical")
		return nil
	}
	for _, name := range changed {
		fmt.Println("conform: synced", sandbox.LibDir+"/"+name)
	}
	fmt.Printf("conform: %d file(s) synced\n", len(changed))
	return nil
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: conform [--local | --fleet]

  (no flag)  check in-repo files against the fleet contract
  --local    check machine-local wiring (hooksPath, hooks, dolt remote)
  --fleet    check GitHub-side settings across the fleet (gh auth)

  sandbox sync   rewrite .sandbox/lib from the canonical copy conform ships
`)
}
