package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/agisilaos/gflight/internal/model"
)

type SerpAPIProvider struct {
	APIKey string
	Client *http.Client
}

type serpResponse struct {
	BestFlights  []serpFlight `json:"best_flights"`
	OtherFlights []serpFlight `json:"other_flights"`
	SearchMeta   struct {
		GoogleFlightsURL string `json:"google_flights_url"`
	} `json:"search_metadata"`
}

type serpFlight struct {
	Price       int    `json:"price"`
	AirlineLogo string `json:"airline_logo"`
	Flights     []struct {
		Airline      string `json:"airline"`
		FlightNumber string `json:"flight_number"`
		Departure    struct {
			Airport string `json:"airport"`
			Time    string `json:"time"`
		} `json:"departure_airport"`
		Arrival struct {
			Airport string `json:"airport"`
			Time    string `json:"time"`
		} `json:"arrival_airport"`
		Duration int `json:"duration"`
	} `json:"flights"`
	Layovers []any `json:"layovers"`
}

func (p SerpAPIProvider) Search(query model.SearchQuery) (model.SearchResult, error) {
	if p.APIKey == "" {
		return model.SearchResult{}, fmt.Errorf("%w: serpapi key missing: set GFLIGHT_SERPAPI_KEY or config.serp_api_key", ErrAuthRequired)
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	endpoint := buildSerpURL(query, p.APIKey)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return model.SearchResult{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return model.SearchResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return model.SearchResult{}, fmt.Errorf("serpapi request failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var payload serpResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return model.SearchResult{}, fmt.Errorf("decode serpapi response: %w", err)
	}
	flights := make([]model.Flight, 0, len(payload.BestFlights)+len(payload.OtherFlights))
	for _, item := range append(payload.BestFlights, payload.OtherFlights...) {
		f := mapSerpFlight(query, item)
		if query.MaxPrice > 0 && f.Price > query.MaxPrice {
			continue
		}
		flights = append(flights, f)
	}
	sort.Slice(flights, func(i, j int) bool {
		return flights[i].Price < flights[j].Price
	})
	result := model.SearchResult{
		Query:     query,
		Flights:   flights,
		CheckedAt: time.Now().UTC(),
		URL:       payload.SearchMeta.GoogleFlightsURL,
	}
	if result.URL == "" {
		result.URL = buildGoogleFlightsURL(query)
	}
	return result, nil
}

func buildSerpURL(query model.SearchQuery, apiKey string) string {
	v := url.Values{}
	v.Set("engine", "google_flights")
	v.Set("api_key", apiKey)
	v.Set("departure_id", query.From)
	v.Set("arrival_id", query.To)
	v.Set("outbound_date", query.Depart)
	if query.Return != "" {
		v.Set("return_date", query.Return)
	}
	v.Set("adults", strconv.Itoa(maxInt(query.Adults, 1)))
	v.Set("children", strconv.Itoa(maxInt(query.Children, 0)))
	if query.Cabin != "" {
		v.Set("travel_class", query.Cabin)
	}
	if query.Nonstop {
		v.Set("stops", "0")
	}
	if query.Currency != "" {
		v.Set("currency", query.Currency)
	}
	return "https://serpapi.com/search.json?" + v.Encode()
}

func mapSerpFlight(query model.SearchQuery, raw serpFlight) model.Flight {
	f := model.Flight{
		Provider: "serpapi",
		From:     query.From,
		To:       query.To,
		Price:    raw.Price,
		Currency: firstOr(query.Currency, "USD"),
		Stops:    len(raw.Layovers),
	}
	if len(raw.Flights) > 0 {
		f.Airline = raw.Flights[0].Airline
		f.FlightNumber = raw.Flights[0].FlightNumber
		f.DepartTime = raw.Flights[0].Departure.Time
		f.ArriveTime = raw.Flights[len(raw.Flights)-1].Arrival.Time
		dur := raw.Flights[0].Duration
		if dur > 0 {
			f.Duration = fmt.Sprintf("%dm", dur)
		}
	}
	if f.Airline == "" && raw.AirlineLogo != "" {
		f.Airline = raw.AirlineLogo
	}
	return f
}

func firstOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func maxInt(v, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}
