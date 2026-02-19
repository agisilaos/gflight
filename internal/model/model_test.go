package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSearchQueryJSONTags(t *testing.T) {
	q := SearchQuery{From: "SFO", To: "ATH", Depart: "2026-06-10", SortBy: "price", Cabin: "economy", Adults: 1, Currency: "USD"}
	b, err := json.Marshal(q)
	if err != nil {
		t.Fatalf("marshal query: %v", err)
	}
	s := string(b)
	if strings.Contains(s, "\"From\"") || strings.Contains(s, "\"To\"") || strings.Contains(s, "\"Depart\"") || strings.Contains(s, "\"SortBy\"") {
		t.Fatalf("expected snake_case json keys, got: %s", s)
	}
	if !strings.Contains(s, "\"from\"") || !strings.Contains(s, "\"to\"") || !strings.Contains(s, "\"depart\"") || !strings.Contains(s, "\"sort_by\"") {
		t.Fatalf("expected normalized keys in json: %s", s)
	}
}
