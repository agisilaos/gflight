package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/agisilaos/gflight/internal/config"
	"github.com/agisilaos/gflight/internal/model"
	"github.com/agisilaos/gflight/internal/provider"
)

func newSearchFlagSet(name string) (*flag.FlagSet, *model.SearchQuery) {
	q := &model.SearchQuery{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&q.From, "from", "", "Departure airport/city code")
	fs.StringVar(&q.To, "to", "", "Arrival airport/city code")
	fs.StringVar(&q.Depart, "depart", "", "Outbound date YYYY-MM-DD")
	fs.StringVar(&q.Return, "return", "", "Return date YYYY-MM-DD")
	fs.StringVar(&q.Cabin, "cabin", "economy", "Cabin class")
	fs.IntVar(&q.Adults, "adults", 1, "Number of adults")
	fs.IntVar(&q.Children, "children", 0, "Number of children")
	fs.BoolVar(&q.Nonstop, "nonstop", false, "Nonstop only")
	fs.IntVar(&q.MaxPrice, "max-price", 0, "Maximum acceptable price")
	fs.StringVar(&q.Currency, "currency", "USD", "Currency code")
	fs.StringVar(&q.SortBy, "sort", "price", "Sort mode")
	return fs, q
}

func validateQuery(q model.SearchQuery) error {
	if q.From == "" || q.To == "" || q.Depart == "" {
		return newExitError(ExitInvalidUsage, "--from, --to, and --depart are required")
	}
	return nil
}

func (a App) resolveProvider(cfg config.Config, g globalFlags) (provider.Provider, error) {
	timeout := time.Duration(cfg.ProviderTimeoutSec) * time.Second
	if g.Timeout != "" {
		parsed, err := time.ParseDuration(g.Timeout)
		if err != nil || parsed <= 0 {
			return nil, newExitError(ExitInvalidUsage, "invalid --timeout value %q (use duration like 10s)", g.Timeout)
		}
		timeout = parsed
	}
	backoff := time.Duration(cfg.ProviderBackoffMS) * time.Millisecond

	switch strings.ToLower(cfg.Provider) {
	case "google-url", "google":
		return provider.GoogleURLProvider{}, nil
	default:
		return provider.SerpAPIProvider{
			APIKey:  cfg.SerpAPIKey,
			Timeout: timeout,
			Retries: cfg.ProviderRetries,
			Backoff: backoff,
			BaseURL: "https://serpapi.com",
		}, nil
	}
}

func (a App) cmdSearch(g globalFlags, args []string) error {
	fs, q := newSearchFlagSet("search")
	if err := fs.Parse(args); err != nil {
		return newExitError(ExitInvalidUsage, "%v", err)
	}
	if err := validateQuery(*q); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return wrapExitError(ExitGenericFailure, err)
	}
	if err := validateProviderRuntime(cfg); err != nil {
		if errors.Is(err, errProviderAuthMissing) {
			return wrapExitError(ExitAuthRequired, err)
		}
		return newExitError(ExitInvalidUsage, "%v", err)
	}
	p, err := a.resolveProvider(cfg, g)
	if err != nil {
		return err
	}
	res, err := p.Search(*q)
	if err != nil {
		if errors.Is(err, provider.ErrAuthRequired) {
			return wrapExitError(ExitAuthRequired, err)
		}
		return wrapExitError(ExitProviderFailure, err)
	}
	if g.JSON {
		return writeJSON(res)
	}
	if g.Plain {
		for _, f := range res.Flights {
			fmt.Printf("%d\t%s\t%s\t%s\t%s\n", f.Price, f.Currency, f.Airline, f.DepartTime, f.ArriveTime)
		}
		fmt.Println(res.URL)
		return nil
	}
	if len(res.Flights) == 0 {
		fmt.Printf("No priced flights returned. Open Google Flights:\n%s\n", res.URL)
		return nil
	}
	limit := len(res.Flights)
	if limit > 10 {
		limit = 10
	}
	fmt.Printf("Top %d flight options for %s -> %s on %s\n", limit, q.From, q.To, q.Depart)
	for i := 0; i < limit; i++ {
		f := res.Flights[i]
		fmt.Printf("%2d) %4d %s | %s | stops:%d | %s -> %s\n", i+1, f.Price, f.Currency, f.Airline, f.Stops, f.DepartTime, f.ArriveTime)
	}
	fmt.Printf("Google Flights: %s\n", res.URL)
	return nil
}
