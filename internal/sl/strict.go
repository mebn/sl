package sl

import (
	"strings"
	"time"
)

func applyStrictJourneyFilter(journeys []journey, limit int) []journey {
	if len(journeys) <= 1 {
		return journeys
	}

	drop := make(map[int]bool)
	for idx, j := range journeys {
		segments := tinyMetroHopSegments(j)
		for _, segment := range segments {
			if hasDirectWalkAlternative(journeys, idx, segment) {
				drop[idx] = true
				break
			}
		}
	}

	if len(drop) == 0 {
		return journeys
	}

	filtered := make([]journey, 0, len(journeys))
	fallback := make([]journey, 0, len(drop))
	for idx, j := range journeys {
		if drop[idx] {
			fallback = append(fallback, j)
			continue
		}
		filtered = append(filtered, j)
	}

	if len(filtered) < limit {
		needed := limit - len(filtered)
		if needed > len(fallback) {
			needed = len(fallback)
		}
		filtered = append(filtered, fallback[:needed]...)
	}

	return filtered
}

func tinyMetroHopSegments(j journey) []tinyHopSegment {
	if len(j.Legs) < 3 {
		return nil
	}

	segments := make([]tinyHopSegment, 0)
	for i := 1; i+1 < len(j.Legs); i++ {
		prev := j.Legs[i-1]
		mid := j.Legs[i]
		next := j.Legs[i+1]

		if !isWalkingLeg(prev) || !isMetroLeg(mid) || !isWalkingLeg(next) {
			continue
		}

		midDep, midDepOK := legDepartureTime(mid)
		midArr, midArrOK := legArrivalTime(mid)
		if midDepOK && midArrOK {
			duration := midArr.Sub(midDep)
			if duration < 0 || duration > 3*time.Minute {
				continue
			}
		}

		segmentArr, segmentArrOK := legArrivalTime(next)
		segments = append(segments, tinyHopSegment{
			From:   lineStopName(mid.Origin.Name),
			To:     lineStopName(next.Destination.Name),
			Dep:    midDep,
			Arr:    segmentArr,
			HasDep: midDepOK,
			HasArr: segmentArrOK,
		})
	}

	return segments
}

func hasDirectWalkAlternative(journeys []journey, skipIndex int, segment tinyHopSegment) bool {
	for idx, j := range journeys {
		if idx == skipIndex {
			continue
		}

		for _, leg := range j.Legs {
			if !isWalkingLeg(leg) {
				continue
			}

			from := lineStopName(leg.Origin.Name)
			to := lineStopName(leg.Destination.Name)
			if !strings.EqualFold(from, segment.From) || !strings.EqualFold(to, segment.To) {
				continue
			}

			walkDep, walkDepOK := legDepartureTime(leg)
			walkArr, walkArrOK := legArrivalTime(leg)

			if segment.HasDep && walkDepOK && walkDep.Before(segment.Dep.Add(-10*time.Minute)) {
				continue
			}
			if segment.HasArr && walkArrOK && walkArr.After(segment.Arr.Add(5*time.Minute)) {
				continue
			}

			return true
		}
	}

	return false
}
