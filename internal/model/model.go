package model

import "time"

type SearchQuery struct {
	From     string
	To       string
	Depart   string
	Return   string
	Cabin    string
	Adults   int
	Children int
	Nonstop  bool
	MaxPrice int
	Currency string
	SortBy   string
}

type Flight struct {
	Provider     string `json:"provider"`
	Airline      string `json:"airline"`
	FlightNumber string `json:"flight_number,omitempty"`
	From         string `json:"from"`
	To           string `json:"to"`
	DepartTime   string `json:"depart_time,omitempty"`
	ArriveTime   string `json:"arrive_time,omitempty"`
	Duration     string `json:"duration,omitempty"`
	Stops        int    `json:"stops"`
	Price        int    `json:"price"`
	Currency     string `json:"currency"`
	DeepLink     string `json:"deep_link,omitempty"`
}

type SearchResult struct {
	Query     SearchQuery `json:"query"`
	Flights   []Flight    `json:"flights"`
	CheckedAt time.Time   `json:"checked_at"`
	URL       string      `json:"google_flights_url"`
}

type Watch struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Query           SearchQuery `json:"query"`
	Enabled         bool        `json:"enabled"`
	TargetPrice     int         `json:"target_price"`
	NotifyTerminal  bool        `json:"notify_terminal"`
	NotifyEmail     bool        `json:"notify_email"`
	EmailTo         string      `json:"email_to,omitempty"`
	LastLowestPrice int         `json:"last_lowest_price"`
	LastRunAt       time.Time   `json:"last_run_at,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type WatchStore struct {
	Watches []Watch `json:"watches"`
}

type Alert struct {
	WatchID     string    `json:"watch_id"`
	WatchName   string    `json:"watch_name"`
	TriggeredAt time.Time `json:"triggered_at"`
	Reason      string    `json:"reason"`
	LowestPrice int       `json:"lowest_price"`
	Currency    string    `json:"currency"`
	URL         string    `json:"google_flights_url"`
}
