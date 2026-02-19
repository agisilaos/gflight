package provider

import "github.com/agisilaos/gflight/internal/model"

type Provider interface {
	Search(query model.SearchQuery) (model.SearchResult, error)
}
