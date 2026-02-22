package sl

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func printJourneys(out io.Writer, fromLoc, toLoc location, journeys []journey) {
	color := supportsColor()
	fmt.Fprintf(out, "%s\n", colorize(routeNameFallback(fromLoc.Name)+" -> "+routeNameFallback(toLoc.Name), ansiBold+ansiCyan, color))

	for i, j := range journeys {
		if i >= resultsToShow {
			break
		}
		if i > 0 {
			fmt.Fprintln(out)
		}

		dep, depOK := departureTime(j)
		arr, arrOK := arrivalTime(j)
		changesWord := "changes"
		if j.Interchanges == 1 {
			changesWord = "change"
		}
		fmt.Fprintf(out, "%s %s\n",
			colorize(fmt.Sprintf("%d. %s -> %s", i+1, formatTime(dep, depOK), formatTime(arr, arrOK)), ansiBlue, color),
			colorize(fmt.Sprintf("(%s, %d %s)", formatDuration(journeyDuration(j)), j.Interchanges, changesWord), ansiDim, color),
		)

		legs := compactWalkLegs(j.Legs)
		var prevEnd time.Time
		havePrevEnd := false
		for idx, l := range legs {
			depText, arrText, legEnd, haveLegEnd := legTimeWindow(l, prevEnd, havePrevEnd)

			from := lineStopName(l.Origin.Name)
			to := lineStopName(l.Destination.Name)
			if idx == 0 && isAddressType(fromLoc.Type) {
				from = routeNameFallback(fromLoc.Name)
			}
			if idx == len(legs)-1 && isAddressType(toLoc.Type) {
				to = routeNameFallback(toLoc.Name)
			}
			if isWalkingLeg(l) && strings.EqualFold(from, to) {
				if haveLegEnd {
					prevEnd, havePrevEnd = legEnd, true
				}
				continue
			}

			fmt.Fprintf(out, "   %s %s %s -> %s\n",
				colorize(depText+" -> "+arrText, ansiGreen, color),
				colorize(legLabel(l), modeColor(l), color),
				from,
				to,
			)
			if haveLegEnd {
				prevEnd, havePrevEnd = legEnd, true
			}
		}
	}
}

func compactWalkLegs(legs []leg) []leg {
	if len(legs) < 2 {
		return legs
	}
	out := make([]leg, 0, len(legs))
	for _, l := range legs {
		if len(out) > 0 && isWalkingLeg(out[len(out)-1]) && isWalkingLeg(l) {
			merged := out[len(out)-1]
			merged.Destination = l.Destination
			out[len(out)-1] = merged
			continue
		}
		out = append(out, l)
	}
	return out
}

func departureTime(j journey) (time.Time, bool) {
	if len(j.Legs) == 0 {
		return time.Time{}, false
	}
	return parseTripTime(j.Legs[0].Origin.DepartureTimeEstimated, j.Legs[0].Origin.DepartureTimePlanned, j.Legs[0].Origin.ArrivalTimeEstimated, j.Legs[0].Origin.ArrivalTimePlanned)
}

func arrivalTime(j journey) (time.Time, bool) {
	if len(j.Legs) == 0 {
		return time.Time{}, false
	}
	last := j.Legs[len(j.Legs)-1].Destination
	return parseTripTime(last.ArrivalTimeEstimated, last.ArrivalTimePlanned, last.DepartureTimeEstimated, last.DepartureTimePlanned)
}

func legDepartureTime(l leg) (time.Time, bool) {
	return parseTripTime(l.Origin.DepartureTimeEstimated, l.Origin.DepartureTimePlanned, l.Origin.ArrivalTimeEstimated, l.Origin.ArrivalTimePlanned)
}

func legArrivalTime(l leg) (time.Time, bool) {
	return parseTripTime(l.Destination.ArrivalTimeEstimated, l.Destination.ArrivalTimePlanned, l.Destination.DepartureTimeEstimated, l.Destination.DepartureTimePlanned)
}

func legTimeWindow(l leg, prevEnd time.Time, havePrev bool) (string, string, time.Time, bool) {
	dep, depOK := legDepartureTime(l)
	arr, arrOK := legArrivalTime(l)

	if !depOK && havePrev {
		dep, depOK = prevEnd, true
	}
	if depOK && havePrev && dep.Before(prevEnd) {
		dep = prevEnd
	}
	if !arrOK && depOK {
		arr, arrOK = dep, true
	}
	if depOK && arrOK && arr.Before(dep) {
		arr = dep
	}
	if depOK && arrOK && arr.After(dep) && dep.Format("15:04") == arr.Format("15:04") {
		arr = arr.Truncate(time.Minute).Add(time.Minute)
	}

	end, endOK := arr, arrOK
	if !endOK {
		end, endOK = dep, depOK
	}
	return formatTime(dep, depOK), formatTime(arr, arrOK), end, endOK
}

func parseTripTime(values ...string) (time.Time, bool) {
	for _, v := range values {
		if v == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			return t.In(time.Local), true
		}
	}
	return time.Time{}, false
}

func journeyDuration(j journey) time.Duration {
	if j.TripRtDuration > 0 {
		return time.Duration(j.TripRtDuration) * time.Second
	}
	if j.TripDuration > 0 {
		return time.Duration(j.TripDuration) * time.Second
	}
	return 0
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "unknown duration"
	}
	d = d.Round(time.Minute)
	h, m := d/time.Hour, (d%time.Hour)/time.Minute
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh%dm", h, m)
}

func legLabel(l leg) string {
	mode := strings.TrimSpace(l.Transportation.Product.Name)
	line := strings.TrimSpace(l.Transportation.DisassembledName)
	if line == "" {
		line = strings.TrimSpace(l.Transportation.Number)
	}
	if line == "" {
		line = strings.TrimSpace(l.Transportation.Name)
	}
	if mode == "" {
		if line == "" {
			return "Unknown"
		}
		return line
	}
	if line == "" || strings.EqualFold(mode, line) {
		return mode
	}
	return mode + " " + line
}

func isWalkingLeg(l leg) bool {
	x := strings.ToLower(legLabel(l))
	return strings.Contains(x, "footpath") || strings.Contains(x, "walk")
}

func isMetroLeg(l leg) bool {
	x := strings.ToLower(legLabel(l))
	return strings.HasPrefix(x, "metro") || strings.Contains(x, "tunnelbana")
}

func modeColor(l leg) string {
	x := strings.ToLower(legLabel(l))
	switch {
	case strings.Contains(x, "footpath") || strings.Contains(x, "walk"):
		return ansiRed
	case strings.HasPrefix(x, "bus"):
		return ansiYellow
	case strings.HasPrefix(x, "metro") || strings.Contains(x, "tunnelbana"):
		return ansiBlue
	case strings.HasPrefix(x, "tram"):
		return ansiMagenta
	case strings.HasPrefix(x, "train") || strings.Contains(x, "commuter"):
		return ansiCyan
	case strings.HasPrefix(x, "ferry") || strings.HasPrefix(x, "ship"):
		return ansiGreen
	default:
		return ansiYellow
	}
}

func formatTime(t time.Time, ok bool) string {
	if !ok {
		return "--:--"
	}
	return t.Format("15:04")
}

func colorize(s, c string, on bool) string {
	if !on || s == "" || c == "" {
		return s
	}
	return c + s + ansiReset
}

func supportsColor() bool {
	if os.Getenv("NO_COLOR") != "" || strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	st, err := os.Stdout.Stat()
	return err == nil && (st.Mode()&os.ModeCharDevice) != 0
}

func isAddressType(t string) bool {
	t = strings.ToLower(strings.TrimSpace(t))
	return t == "singlehouse" || t == "address" || t == "street"
}

func lineStopName(name string) string {
	head, _, ok := strings.Cut(name, ",")
	if ok {
		return strings.TrimSpace(head)
	}
	return strings.TrimSpace(name)
}

func routeNameFallback(name string) string {
	parts := strings.Split(name, ",")
	return strings.TrimSpace(parts[len(parts)-1])
}
