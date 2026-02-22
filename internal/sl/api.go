package sl

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func lookupStop(client *http.Client, search string) (location, error) {
	u, err := url.Parse(stopFinderURL)
	if err != nil {
		return location{}, err
	}

	q := u.Query()
	q.Set("name_sf", search)
	q.Set("any_obj_filter_sf", "46")
	q.Set("type_sf", "any")
	u.RawQuery = q.Encode()

	resp, err := client.Get(u.String())
	if err != nil {
		return location{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return location{}, fmt.Errorf("unexpected HTTP status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload stopFinderResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return location{}, err
	}

	best, ok := pickBestLocation(payload.Locations)
	if !ok {
		return location{}, errors.New("no matching stop found")
	}

	return best, nil
}

func pickBestLocation(locations []location) (location, bool) {
	if len(locations) == 0 {
		return location{}, false
	}

	for _, loc := range locations {
		if loc.IsBest {
			return loc, true
		}
	}

	bestIdx := -1
	for i, loc := range locations {
		if bestIdx == -1 || loc.MatchQuality > locations[bestIdx].MatchQuality {
			bestIdx = i
		}
	}

	if bestIdx != -1 {
		return locations[bestIdx], true
	}

	return locations[0], true
}

func fetchTrips(client *http.Client, fromID, toID string, limit int) ([]journey, error) {
	if limit <= 0 {
		limit = 1
	}

	candidateLimit := limit * 3
	if candidateLimit < limit+3 {
		candidateLimit = limit + 3
	}

	now := time.Now().In(time.Local)
	lateTolerance := 2 * time.Minute
	cursor := now
	seen := make(map[string]struct{})
	results := make([]journey, 0, candidateLimit)

	for attempts := 0; attempts < 10 && len(results) < candidateLimit; attempts++ {
		batch, err := fetchTripsBatch(client, fromID, toID, cursor)
		if err != nil {
			if len(results) > 0 {
				break
			}
			return nil, err
		}
		if len(batch) == 0 {
			break
		}

		latestDep := cursor
		for _, j := range batch {
			dep, depOK := departureTime(j)
			if depOK && dep.After(latestDep) {
				latestDep = dep
			}

			if depOK && dep.Before(now.Add(-lateTolerance)) {
				continue
			}

			key := journeyKey(j)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			results = append(results, j)
			if len(results) >= candidateLimit {
				break
			}
		}

		nextCursor := latestDep.Add(time.Minute)
		if !nextCursor.After(cursor) {
			nextCursor = cursor.Add(10 * time.Minute)
		}
		cursor = nextCursor
	}

	if len(results) == 0 {
		return nil, errors.New("no journeys found")
	}

	if strictMode {
		results = applyStrictJourneyFilter(results, limit)
	}

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func fetchTripsBatch(client *http.Client, fromID, toID string, departureFrom time.Time) ([]journey, error) {
	u, err := url.Parse(tripsURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("type_origin", "any")
	q.Set("type_destination", "any")
	q.Set("name_origin", fromID)
	q.Set("name_destination", toID)
	q.Set("calc_number_of_trips", "3")
	q.Set("language", "en")
	q.Set("calc_one_direction", "true")
	q.Set("itd_trip_date_time_dep_arr", "dep")
	q.Set("itd_date", departureFrom.Format("20060102"))
	q.Set("itd_time", departureFrom.Format("1504"))
	u.RawQuery = q.Encode()

	resp, err := client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("unexpected HTTP status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload tripsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	if len(payload.Journeys) == 0 {
		var errorTexts []string
		for _, msg := range payload.Messages {
			if strings.EqualFold(msg.Type, "error") && strings.TrimSpace(msg.Text) != "" {
				errorTexts = append(errorTexts, msg.Text)
			}
		}
		if len(errorTexts) > 0 {
			return nil, errors.New(strings.Join(errorTexts, "; "))
		}
		return nil, nil
	}

	return payload.Journeys, nil
}

func journeyKey(j journey) string {
	dep, depOK := departureTime(j)
	arr, arrOK := arrivalTime(j)
	depPart := "--"
	arrPart := "--"
	if depOK {
		depPart = dep.Format(time.RFC3339)
	}
	if arrOK {
		arrPart = arr.Format(time.RFC3339)
	}

	if len(j.Legs) == 0 {
		return depPart + "|" + arrPart
	}

	first := lineStopName(j.Legs[0].Origin.Name)
	last := lineStopName(j.Legs[len(j.Legs)-1].Destination.Name)
	return depPart + "|" + arrPart + "|" + first + "|" + last
}
