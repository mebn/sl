package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mebn/sl/internal/sl"
)

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}

func Run(args []string, stdout, stderr io.Writer) int {
	opts, err := sl.ParseCLIArgs(args)
	if err != nil {
		sl.PrintUsage(stderr)
		return failf(stderr, "%v", err)
	}

	if opts.ShowHelp {
		sl.PrintUsage(stdout)
		return 0
	}

	if opts.Upgrade {
		return runUpgrade(stdout, stderr)
	}

	from, to, err := sl.ResolveFromTo(opts.Positionals, opts.Reverse)
	if err != nil {
		if len(opts.Positionals) != 0 && len(opts.Positionals) != 2 {
			sl.PrintUsage(stderr)
		}
		return failf(stderr, "%v", err)
	}

	if opts.SavePair {
		if err := sl.SaveRoute(from, to); err != nil {
			return failf(stderr, "failed to save default route: %v", err)
		}
		fmt.Fprintf(stdout, "Saved default route: %s -> %s\n\n", from, to)
	}

	client := &http.Client{Timeout: 12 * time.Second}

	fromLoc, err := sl.LookupStop(client, from)
	if err != nil {
		return failf(stderr, "failed to resolve '%s': %v", from, err)
	}

	toLoc, err := sl.LookupStop(client, to)
	if err != nil {
		return failf(stderr, "failed to resolve '%s': %v", to, err)
	}

	journeys, err := sl.FetchTrips(client, fromLoc.ID, toLoc.ID, sl.ResultsToShow)
	if err != nil {
		return failf(stderr, "failed to fetch trips: %v", err)
	}

	sl.PrintJourneys(stdout, fromLoc, toLoc, journeys)
	return 0
}

func runUpgrade(stdout, stderr io.Writer) int {
	cmd := exec.Command("go", "install", "github.com/mebn/sl@latest")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return failf(stderr, "upgrade failed: %v", err)
	}
	if !strings.EqualFold(os.Getenv("TERM"), "dumb") {
		fmt.Fprintln(stdout, "Upgrade complete")
	}
	return 0
}

func failf(w io.Writer, format string, args ...any) int {
	fmt.Fprintf(w, format+"\n", args...)
	return 1
}
