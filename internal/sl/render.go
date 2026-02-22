package sl

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func printJourneys(out io.Writer, fromLoc, toLoc location, journeys []journey) {
	useColor := supportsColor()
	displayFrom := routeNameFallback(fromLoc.Name)
	displayTo := routeNameFallback(toLoc.Name)

	routeHeader := fmt.Sprintf("%s -> %s", displayFrom, displayTo)
	fmt.Fprintf(out, "%s\n", colorize(routeHeader, ansiBold+ansiCyan, useColor))

	for i, j := range journeys {
		if i >= resultsToShow {
			break
		}
		if i > 0 {
			fmt.Fprintln(out)
		}

		dep, depOK := departureTime(j)
		arr, arrOK := arrivalTime(j)
		dur := journeyDuration(j)

		altLine := fmt.Sprintf("%d. %s -> %s", i+1, formatTime(dep, depOK), formatTime(arr, arrOK))
		detailLine := fmt.Sprintf("(%s, %d %s)", formatDuration(dur), j.Interchanges, pluralize("change", j.Interchanges))
		fmt.Fprintf(out, "%s %s\n", colorize(altLine, ansiBlue, useColor), colorize(detailLine, ansiDim, useColor))

		displayLegs := compactLegsForDisplay(j.Legs)
		var prevLegArrival time.Time
		havePrevLegArrival := false
		for legIdx, leg := range displayLegs {
			legDep, legDepOK := legDepartureTime(leg)
			legArr, legArrOK := legArrivalTime(leg)
			legDep, legDepOK, legArr, legArrOK = normalizeLegTimes(legDep, legDepOK, legArr, legArrOK, prevLegArrival, havePrevLegArrival)

			originName := lineStopName(leg.Origin.Name)
			destinationName := lineStopName(leg.Destination.Name)
			if legIdx == 0 && isAddressLike(fromLoc.Type) {
				originName = routeNameFallback(fromLoc.Name)
			}
			if legIdx == len(displayLegs)-1 && isAddressLike(toLoc.Type) {
				destinationName = routeNameFallback(toLoc.Name)
			}
			if isWalkingLeg(leg) && strings.EqualFold(originName, destinationName) {
				if legArrOK {
					prevLegArrival = legArr
					havePrevLegArrival = true
				} else if legDepOK {
					prevLegArrival = legDep
					havePrevLegArrival = true
				}
				continue
			}

			legTimes := formatLegTimeRange(legDep, legDepOK, legArr, legArrOK)
			fmt.Fprintf(out, "   %s %s %s -> %s\n", colorize(legTimes, ansiGreen, useColor), colorize(legLabel(leg), modeColor(leg), useColor), originName, destinationName)

			if legArrOK {
				prevLegArrival = legArr
				havePrevLegArrival = true
			} else if legDepOK {
				prevLegArrival = legDep
				havePrevLegArrival = true
			}
		}
	}
}

func compactLegsForDisplay(legs []leg) []leg {
	if len(legs) < 2 {
		return legs
	}

	out := make([]leg, 0, len(legs))
	for _, current := range legs {
		if len(out) == 0 {
			out = append(out, current)
			continue
		}

		lastIdx := len(out) - 1
		if isWalkingLeg(out[lastIdx]) && isWalkingLeg(current) {
			merged := out[lastIdx]
			merged.Destination = current.Destination
			out[lastIdx] = merged
			continue
		}

		out = append(out, current)
	}

	return out
}

func departureTime(j journey) (time.Time, bool) {
	if len(j.Legs) == 0 {
		return time.Time{}, false
	}
	origin := j.Legs[0].Origin
	return parseTripTime(origin.DepartureTimeEstimated, origin.DepartureTimePlanned, origin.ArrivalTimeEstimated, origin.ArrivalTimePlanned)
}

func arrivalTime(j journey) (time.Time, bool) {
	if len(j.Legs) == 0 {
		return time.Time{}, false
	}
	destination := j.Legs[len(j.Legs)-1].Destination
	return parseTripTime(destination.ArrivalTimeEstimated, destination.ArrivalTimePlanned, destination.DepartureTimeEstimated, destination.DepartureTimePlanned)
}

func legDepartureTime(l leg) (time.Time, bool) {
	return parseTripTime(l.Origin.DepartureTimeEstimated, l.Origin.DepartureTimePlanned, l.Origin.ArrivalTimeEstimated, l.Origin.ArrivalTimePlanned)
}

func legArrivalTime(l leg) (time.Time, bool) {
	return parseTripTime(l.Destination.ArrivalTimeEstimated, l.Destination.ArrivalTimePlanned, l.Destination.DepartureTimeEstimated, l.Destination.DepartureTimePlanned)
}

func parseTripTime(values ...string) (time.Time, bool) {
	for _, value := range values {
		if value == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			continue
		}
		return t.In(time.Local), true
	}

	return time.Time{}, false
}

func journeyDuration(j journey) time.Duration {
	seconds := j.TripRtDuration
	if seconds <= 0 {
		seconds = j.TripDuration
	}
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func formatDuration(duration time.Duration) string {
	if duration <= 0 {
		return "unknown duration"
	}

	duration = duration.Round(time.Minute)
	hours := duration / time.Hour
	minutes := (duration % time.Hour) / time.Minute

	if hours == 0 {
		return fmt.Sprintf("%dm", minutes)
	}

	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}

	return fmt.Sprintf("%dh%dm", hours, minutes)
}

func legLabel(leg leg) string {
	mode := strings.TrimSpace(leg.Transportation.Product.Name)
	line := strings.TrimSpace(leg.Transportation.DisassembledName)

	if line == "" {
		line = strings.TrimSpace(leg.Transportation.Number)
	}
	if line == "" {
		line = strings.TrimSpace(leg.Transportation.Name)
	}

	if mode == "" && line == "" {
		return "Unknown"
	}
	if mode == "" {
		return line
	}
	if line == "" || strings.EqualFold(mode, line) {
		return mode
	}

	return mode + " " + line
}

func formatTime(t time.Time, ok bool) string {
	if !ok {
		return "--:--"
	}
	return t.Format("15:04")
}

func formatLegTimeRange(dep time.Time, depOK bool, arr time.Time, arrOK bool) string {
	if !depOK || !arrOK {
		return fmt.Sprintf("%s -> %s", formatTime(dep, depOK), formatTime(arr, arrOK))
	}

	arrDisplay := arr
	if arr.After(dep) && dep.Format("15:04") == arr.Format("15:04") {
		arrDisplay = arr.Truncate(time.Minute).Add(time.Minute)
	}

	return fmt.Sprintf("%s -> %s", dep.Format("15:04"), arrDisplay.Format("15:04"))
}

func normalizeLegTimes(dep time.Time, depOK bool, arr time.Time, arrOK bool, prevArr time.Time, prevArrOK bool) (time.Time, bool, time.Time, bool) {
	if !depOK && prevArrOK {
		dep = prevArr
		depOK = true
	}

	if depOK && prevArrOK && dep.Before(prevArr) {
		dep = prevArr
	}

	if !arrOK && depOK {
		arr = dep
		arrOK = true
	}

	if arrOK && depOK && arr.Before(dep) {
		arr = dep
	}

	return dep, depOK, arr, arrOK
}

func isWalkingLeg(l leg) bool {
	lower := strings.ToLower(legLabel(l))
	return strings.Contains(lower, "footpath") || strings.Contains(lower, "walk")
}

func isMetroLeg(l leg) bool {
	lower := strings.ToLower(legLabel(l))
	return strings.HasPrefix(lower, "metro") || strings.Contains(lower, "tunnelbana")
}

func modeColor(l leg) string {
	lower := strings.ToLower(legLabel(l))

	switch {
	case strings.Contains(lower, "footpath") || strings.Contains(lower, "walk"):
		return ansiRed
	case strings.HasPrefix(lower, "bus"):
		return ansiYellow
	case strings.HasPrefix(lower, "metro") || strings.Contains(lower, "tunnelbana"):
		return ansiBlue
	case strings.HasPrefix(lower, "tram"):
		return ansiMagenta
	case strings.HasPrefix(lower, "train") || strings.Contains(lower, "commuter"):
		return ansiCyan
	case strings.HasPrefix(lower, "ferry") || strings.HasPrefix(lower, "ship"):
		return ansiGreen
	default:
		return ansiYellow
	}
}

func isAddressLike(locationType string) bool {
	locationType = strings.ToLower(strings.TrimSpace(locationType))
	return locationType == "singlehouse" || locationType == "address" || locationType == "street"
}

func colorize(text, colorCode string, enabled bool) string {
	if !enabled || colorCode == "" || text == "" {
		return text
	}
	return colorCode + text + ansiReset
}

func supportsColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}

	stdoutInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return (stdoutInfo.Mode() & os.ModeCharDevice) != 0
}

func pluralize(word string, amount int) string {
	if amount == 1 {
		return word
	}
	return word + "s"
}

func lineStopName(name string) string {
	parts := commaParts(name)
	if len(parts) == 0 {
		return strings.TrimSpace(name)
	}
	return parts[0]
}

func routeNameFallback(name string) string {
	parts := commaParts(name)
	if len(parts) == 0 {
		return strings.TrimSpace(name)
	}
	return parts[len(parts)-1]
}

func commaParts(name string) []string {
	rawParts := strings.Split(name, ",")
	parts := make([]string, 0, len(rawParts))
	for _, raw := range rawParts {
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
