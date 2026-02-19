package provider

import "errors"

import "github.com/agisilaos/gflight/internal/model"

var ErrAuthRequired = errors.New("provider authentication required")

type Provider interface {
	Search(query model.SearchQuery) (model.SearchResult, error)
}
