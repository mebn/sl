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

func LookupStop(client *http.Client, search string) (location, error) {
	var payload stopFinderResponse
	err := getJSON(client, stopFinderURL, map[string]string{
		"name_sf":           search,
		"any_obj_filter_sf": "46",
		"type_sf":           "any",
	}, &payload)
	if err != nil {
		return location{}, err
	}
	loc, ok := pickBestLocation(payload.Locations)
	if !ok {
		return location{}, errors.New("no matching stop found")
	}
	return loc, nil
}

func pickBestLocation(locations []location) (location, bool) {
	if len(locations) == 0 {
		return location{}, false
	}
	best := 0
	for i := 1; i < len(locations); i++ {
		if locations[i].IsBest {
			return locations[i], true
		}
		if locations[i].MatchQuality > locations[best].MatchQuality {
			best = i
		}
	}
	if locations[0].IsBest {
		return locations[0], true
	}
	return locations[best], true
}

func FetchTrips(client *http.Client, fromID, toID string, limit int) ([]journey, error) {
	if limit <= 0 {
		limit = 1
	}

	seen := map[string]struct{}{}
	collected := make([]journey, 0, limit*3)
	now := time.Now().In(time.Local)
	cursor := now
	max := max(limit*3, limit+3)

	for attempts := 0; attempts < 10 && len(collected) < max; attempts++ {
		batch, err := fetchTripsBatch(client, fromID, toID, cursor)
		if err != nil {
			if len(collected) > 0 {
				break
			}
			return nil, err
		}
		if len(batch) == 0 {
			break
		}

		latestDep := cursor
		for _, j := range batch {
			dep, ok := departureTime(j)
			if ok && dep.After(latestDep) {
				latestDep = dep
			}
			if ok && dep.Before(now.Add(-2*time.Minute)) {
				continue
			}

			key := journeyKey(j)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			collected = append(collected, j)
			if len(collected) >= max {
				break
			}
		}

		if latestDep.After(cursor) {
			cursor = latestDep.Add(time.Minute)
		} else {
			cursor = cursor.Add(10 * time.Minute)
		}
	}

	if len(collected) == 0 {
		return nil, errors.New("no journeys found")
	}

	filtered := dropTinyMetroHopDetours(collected, limit)
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func fetchTripsBatch(client *http.Client, fromID, toID string, departureFrom time.Time) ([]journey, error) {
	var payload tripsResponse
	err := getJSON(client, tripsURL, map[string]string{
		"type_origin":                "any",
		"type_destination":           "any",
		"name_origin":                fromID,
		"name_destination":           toID,
		"calc_number_of_trips":       "3",
		"language":                   "en",
		"calc_one_direction":         "true",
		"itd_trip_date_time_dep_arr": "dep",
		"itd_date":                   departureFrom.Format("20060102"),
		"itd_time":                   departureFrom.Format("1504"),
	}, &payload)
	if err != nil {
		return nil, err
	}
	if len(payload.Journeys) > 0 {
		return payload.Journeys, nil
	}

	var errs []string
	for _, msg := range payload.Messages {
		if strings.EqualFold(msg.Type, "error") && strings.TrimSpace(msg.Text) != "" {
			errs = append(errs, msg.Text)
		}
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "; "))
	}
	return nil, nil
}

func journeyKey(j journey) string {
	dep, depOK := departureTime(j)
	arr, arrOK := arrivalTime(j)
	depPart, arrPart := "--", "--"
	if depOK {
		depPart = dep.Format(time.RFC3339)
	}
	if arrOK {
		arrPart = arr.Format(time.RFC3339)
	}
	if len(j.Legs) == 0 {
		return depPart + "|" + arrPart
	}
	return depPart + "|" + arrPart + "|" + lineStopName(j.Legs[0].Origin.Name) + "|" + lineStopName(j.Legs[len(j.Legs)-1].Destination.Name)
}

func dropTinyMetroHopDetours(journeys []journey, limit int) []journey {
	if len(journeys) <= 1 || limit <= 0 {
		return journeys
	}

	clean, detour := make([]journey, 0, len(journeys)), make([]journey, 0, len(journeys))
	for _, j := range journeys {
		if hasTinyMetroHop(j) {
			detour = append(detour, j)
		} else {
			clean = append(clean, j)
		}
	}

	if len(clean) == 0 {
		return journeys
	}
	for len(clean) < limit && len(detour) > 0 {
		clean, detour = append(clean, detour[0]), detour[1:]
	}
	return clean
}

func hasTinyMetroHop(j journey) bool {
	for i := 1; i+1 < len(j.Legs); i++ {
		prev, mid, next := j.Legs[i-1], j.Legs[i], j.Legs[i+1]
		if !isWalkingLeg(prev) || !isMetroLeg(mid) || !isWalkingLeg(next) {
			continue
		}
		dep, depOK := legDepartureTime(mid)
		arr, arrOK := legArrivalTime(mid)
		if !depOK || !arrOK || (arr.After(dep) && arr.Sub(dep) <= 3*time.Minute) {
			return true
		}
	}
	return false
}

func getJSON(client *http.Client, baseURL string, query map[string]string, out any) error {
	u, err := url.Parse(baseURL)
	if err != nil {
		return err
	}
	q := u.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	resp, err := client.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("unexpected HTTP status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
