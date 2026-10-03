package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/agisilaos/gflight/internal/model"
	"github.com/agisilaos/gflight/internal/safeerror"
)

type SerpAPIProvider struct {
	APIKey  string
	Client  *http.Client
	Timeout time.Duration
	Retries int
	Backoff time.Duration
	BaseURL string
}

type serpResponse struct {
	BestFlights  []serpFlight `json:"best_flights"`
	OtherFlights []serpFlight `json:"other_flights"`
	SearchMeta   struct {
		GoogleFlightsURL string `json:"google_flights_url"`
	} `json:"search_metadata"`
}

type serpFlight struct {
	TotalDuration int    `json:"total_duration"`
	Price         int    `json:"price"`
	AirlineLogo   string `json:"airline_logo"`
	Flights       []struct {
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
		client = &http.Client{Timeout: p.resolvedTimeout()}
	}
	endpoint, err := buildSerpURL(p.baseURL(), query, p.APIKey)
	if err != nil {
		return model.SearchResult{}, err
	}

	var payload serpResponse
	if err := p.fetchWithRetry(client, endpoint, &payload); err != nil {
		return model.SearchResult{}, err
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

func (p SerpAPIProvider) fetchWithRetry(client *http.Client, endpoint string, out *serpResponse) error {
	attempts := p.resolvedRetries() + 1
	for attempt := 0; attempt < attempts; attempt++ {
		err := p.fetchOnce(client, endpoint, out)
		if err == nil {
			return nil
		}
		if !isRetryable(err) || attempt == attempts-1 {
			return err
		}
		time.Sleep(p.retryDelay(attempt))
	}
	return fmt.Errorf("%w: exhausted retries", ErrTransient)
}

func (p SerpAPIProvider) fetchOnce(client *http.Client, endpoint string, out *serpResponse) error {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return &safeerror.Error{Message: "invalid provider request URL", Cause: err}
	}
	resp, err := client.Do(req)
	if err != nil {
		if isNetworkTransient(err) {
			return fmt.Errorf("%w: %w", ErrTransient, &safeerror.Error{Message: "provider network request failed", Cause: err})
		}
		return &safeerror.Error{Message: "provider request failed", Cause: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		switch {
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return fmt.Errorf("%w: serpapi request failed: HTTP %d", ErrAuthRequired, resp.StatusCode)
		case resp.StatusCode == http.StatusTooManyRequests:
			return fmt.Errorf("%w: serpapi request failed: HTTP %d", ErrRateLimited, resp.StatusCode)
		case resp.StatusCode >= 500:
			return fmt.Errorf("%w: serpapi request failed: HTTP %d", ErrTransient, resp.StatusCode)
		default:
			return fmt.Errorf("serpapi request failed: HTTP %d", resp.StatusCode)
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return &safeerror.Error{Message: "decode serpapi response failed", Cause: err}
	}
	return nil
}

func isRetryable(err error) bool {
	return errors.Is(err, ErrTransient) || errors.Is(err, ErrRateLimited)
}

func isNetworkTransient(err error) bool {
	if errors.Is(err, io.EOF) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return false
}

func (p SerpAPIProvider) resolvedTimeout() time.Duration {
	if p.Timeout > 0 {
		return p.Timeout
	}
	return 20 * time.Second
}

func (p SerpAPIProvider) resolvedRetries() int {
	if p.Retries < 0 {
		return 0
	}
	return p.Retries
}

func (p SerpAPIProvider) resolvedBackoff() time.Duration {
	if p.Backoff > 0 {
		return p.Backoff
	}
	return 400 * time.Millisecond
}

func (p SerpAPIProvider) retryDelay(attempt int) time.Duration {
	base := p.resolvedBackoff()
	shift := attempt
	if shift > 5 {
		shift = 5
	}
	return base * time.Duration(1<<shift)
}

func (p SerpAPIProvider) baseURL() string {
	if p.BaseURL != "" {
		return strings.TrimRight(p.BaseURL, "/")
	}
	return "https://serpapi.com"
}

func buildSerpURL(baseURL string, query model.SearchQuery, apiKey string) (string, error) {
	cabin, err := SerpAPITravelClass(query.Cabin)
	if err != nil {
		return "", err
	}
	v := url.Values{}
	v.Set("engine", "google_flights")
	v.Set("api_key", apiKey)
	v.Set("departure_id", query.From)
	v.Set("arrival_id", query.To)
	v.Set("outbound_date", query.Depart)
	v.Set("type", "2")
	if query.Return != "" {
		v.Set("type", "1")
		v.Set("return_date", query.Return)
	}
	v.Set("adults", strconv.Itoa(maxInt(query.Adults, 1)))
	v.Set("children", strconv.Itoa(maxInt(query.Children, 0)))
	v.Set("travel_class", cabin)
	if query.Nonstop {
		v.Set("stops", "1")
	}
	if query.Currency != "" {
		v.Set("currency", query.Currency)
	}
	return strings.TrimRight(baseURL, "/") + "/search.json?" + v.Encode(), nil
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
	}
	if raw.TotalDuration > 0 {
		f.Duration = fmt.Sprintf("%dm", raw.TotalDuration)
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

// SerpAPITravelClass translates CLI cabin labels and legacy numeric aliases.
func SerpAPITravelClass(cabin string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(cabin)) {
	case "", "economy", "1":
		return "1", nil
	case "premium-economy", "2":
		return "2", nil
	case "business", "3":
		return "3", nil
	case "first", "4":
		return "4", nil
	default:
		return "", fmt.Errorf("invalid cabin %q (use economy, premium-economy, business, or first)", cabin)
	}
}
