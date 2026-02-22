package sl

import "time"

type stopFinderResponse struct {
	Locations []location      `json:"locations"`
	Messages  []systemMessage `json:"systemMessages"`
}

type location struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	IsBest       bool   `json:"isBest"`
	MatchQuality int    `json:"matchQuality"`
}

type tripsResponse struct {
	Journeys []journey       `json:"journeys"`
	Messages []systemMessage `json:"systemMessages"`
}

type journey struct {
	TripDuration   int   `json:"tripDuration"`
	TripRtDuration int   `json:"tripRtDuration"`
	Interchanges   int   `json:"interchanges"`
	Legs           []leg `json:"legs"`
}

type leg struct {
	Origin         journeyPoint   `json:"origin"`
	Destination    journeyPoint   `json:"destination"`
	Transportation transportation `json:"transportation"`
}

type journeyPoint struct {
	Name                   string `json:"name"`
	DepartureTimePlanned   string `json:"departureTimePlanned"`
	DepartureTimeEstimated string `json:"departureTimeEstimated"`
	ArrivalTimePlanned     string `json:"arrivalTimePlanned"`
	ArrivalTimeEstimated   string `json:"arrivalTimeEstimated"`
}

type transportation struct {
	Name             string  `json:"name"`
	Number           string  `json:"number"`
	DisassembledName string  `json:"disassembledName"`
	Product          product `json:"product"`
}

type product struct {
	Name string `json:"name"`
}

type systemMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type appConfig struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type cliOptions struct {
	SavePair    bool
	Reverse     bool
	ShowHelp    bool
	Positionals []string
}

type tinyHopSegment struct {
	From   string
	To     string
	Dep    time.Time
	Arr    time.Time
	HasDep bool
	HasArr bool
}
