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
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	if opts.ShowHelp {
		printUsage(stdout)
		return 0
	}

	if len(opts.Positionals) != 0 && len(opts.Positionals) != 2 {
		printUsage(stderr)
		fmt.Fprintf(stderr, "expected either no arguments or exactly two: <from> <to>\n")
		return 1
	}

	from, to, err := resolveFromTo(opts.Positionals, opts.Reverse)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	if opts.SavePair {
		if err := saveConfig(appConfig{From: from, To: to}); err != nil {
			fmt.Fprintf(stderr, "failed to save default route: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Saved default route: %s -> %s\n\n", from, to)
	}

	client := &http.Client{Timeout: 12 * time.Second}

	fromLoc, err := lookupStop(client, from)
	if err != nil {
		fmt.Fprintf(stderr, "failed to resolve '%s': %v\n", from, err)
		return 1
	}

	toLoc, err := lookupStop(client, to)
	if err != nil {
		fmt.Fprintf(stderr, "failed to resolve '%s': %v\n", to, err)
		return 1
	}

	journeys, err := fetchTrips(client, fromLoc.ID, toLoc.ID, resultsToShow)
	if err != nil {
		fmt.Fprintf(stderr, "failed to fetch trips: %v\n", err)
		return 1
	}

	printJourneys(stdout, fromLoc, toLoc, journeys)
	return 0
}
