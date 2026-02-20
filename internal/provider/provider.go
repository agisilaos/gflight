package provider

import "errors"

import "github.com/agisilaos/gflight/internal/model"

var ErrAuthRequired = errors.New("provider authentication required")
var ErrRateLimited = errors.New("provider rate limited")
var ErrTransient = errors.New("provider transient failure")

type Provider interface {
	Search(query model.SearchQuery) (model.SearchResult, error)
}
