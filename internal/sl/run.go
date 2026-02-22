package sl

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func Run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseCLIArgs(args)
	if err != nil {
		printUsage(stderr)
		return failf(stderr, "%v", err)
	}

	if opts.ShowHelp {
		printUsage(stdout)
		return 0
	}

	from, to, err := resolveFromTo(opts.Positionals, opts.Reverse)
	if err != nil {
		if len(opts.Positionals) != 0 && len(opts.Positionals) != 2 {
			printUsage(stderr)
		}
		return failf(stderr, "%v", err)
	}

	if opts.SavePair {
		if err := saveConfig(appConfig{From: from, To: to}); err != nil {
			return failf(stderr, "failed to save default route: %v", err)
		}
		fmt.Fprintf(stdout, "Saved default route: %s -> %s\n\n", from, to)
	}

	client := &http.Client{Timeout: 12 * time.Second}

	fromLoc, err := lookupStop(client, from)
	if err != nil {
		return failf(stderr, "failed to resolve '%s': %v", from, err)
	}

	toLoc, err := lookupStop(client, to)
	if err != nil {
		return failf(stderr, "failed to resolve '%s': %v", to, err)
	}

	journeys, err := fetchTrips(client, fromLoc.ID, toLoc.ID, resultsToShow)
	if err != nil {
		return failf(stderr, "failed to fetch trips: %v", err)
	}

	printJourneys(stdout, fromLoc, toLoc, journeys)
	return 0
}

func failf(w io.Writer, format string, args ...any) int {
	fmt.Fprintf(w, format+"\n", args...)
	return 1
}
