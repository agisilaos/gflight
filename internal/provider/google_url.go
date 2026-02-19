package provider

import (
	"fmt"
	"net/url"
	"time"

	"github.com/agisilaos/gflight/internal/model"
)

type GoogleURLProvider struct{}

func (p GoogleURLProvider) Search(query model.SearchQuery) (model.SearchResult, error) {
	result := model.SearchResult{
		Query:     query,
		Flights:   []model.Flight{},
		CheckedAt: time.Now().UTC(),
		URL:       buildGoogleFlightsURL(query),
	}
	return result, nil
}

func buildGoogleFlightsURL(query model.SearchQuery) string {
	values := url.Values{}
	values.Set("f", query.From)
	values.Set("t", query.To)
	values.Set("d", query.Depart)
	if query.Return != "" {
		values.Set("r", query.Return)
	}
	if query.Nonstop {
		values.Set("sc", "1")
	}
	if query.Cabin != "" {
		values.Set("c", query.Cabin)
	}
	if query.Adults > 0 {
		values.Set("ad", fmt.Sprintf("%d", query.Adults))
	}
	if query.Children > 0 {
		values.Set("ch", fmt.Sprintf("%d", query.Children))
	}
	return "https://www.google.com/travel/flights?" + values.Encode()
}
