package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/agisilaos/gflight/internal/config"
	"github.com/agisilaos/gflight/internal/model"
	"github.com/agisilaos/gflight/internal/notify"
	"github.com/agisilaos/gflight/internal/provider"
	"github.com/agisilaos/gflight/internal/watcher"
)

type App struct {
	Version string
}

type globalFlags struct {
	JSON     bool
	Plain    bool
	Quiet    bool
	Verbose  bool
	NoInput  bool
	NoColor  bool
	StateDir string
	Help     bool
	Version  bool
}

func NewApp(version string) App {
	return App{Version: version}
}

func (a App) Run(args []string) error {
	g, rest, err := parseGlobal(args)
	if err != nil {
		return err
	}
	if g.Help {
		return a.help(nil)
	}
	if g.Version {
		fmt.Println(a.Version)
		return nil
	}
	if len(rest) == 0 {
		return a.help(nil)
	}
	cmd := rest[0]
	argv := rest[1:]

	switch cmd {
	case "help", "-h", "--help":
		return a.help(nil)
	case "--version", "version":
		fmt.Println(a.Version)
		return nil
	case "search":
		return a.cmdSearch(g, argv)
	case "watch":
		return a.cmdWatch(g, argv)
	case "notify":
		return a.cmdNotify(g, argv)
	case "auth":
		return a.cmdAuth(g, argv)
	case "config":
		return a.cmdConfig(g, argv)
	default:
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usageText())
	}
}

func parseGlobal(args []string) (globalFlags, []string, error) {
	var g globalFlags
	for len(args) > 0 {
		a := args[0]
		switch a {
		case "-h", "--help":
			g.Help = true
			args = args[1:]
		case "--version":
			g.Version = true
			args = args[1:]
		case "--json":
			g.JSON = true
			args = args[1:]
		case "--plain":
			g.Plain = true
			args = args[1:]
		case "-q", "--quiet":
			g.Quiet = true
			args = args[1:]
		case "-v", "--verbose":
			g.Verbose = true
			args = args[1:]
		case "--no-input":
			g.NoInput = true
			args = args[1:]
		case "--no-color":
			g.NoColor = true
			args = args[1:]
		case "--state-dir":
			if len(args) < 2 {
				return g, nil, fmt.Errorf("--state-dir requires a value")
			}
			g.StateDir = args[1]
			args = args[2:]
		default:
			if strings.HasPrefix(a, "-") {
				return g, nil, fmt.Errorf("unknown global flag %q", a)
			}
			return g, args, nil
		}
	}
	return g, args, nil
}

func (a App) help(_ []string) error {
	fmt.Print(usageText())
	return nil
}

func usageText() string {
	return `gflight - Search Google Flights and run price alerts

USAGE:
  gflight [global flags] <command> [args]

COMMANDS:
  search             One-shot flight search
  watch create       Create a watch
  watch list         List watches
  watch enable       Enable a watch
  watch disable      Disable a watch
  watch delete       Delete a watch
  watch run          Execute watches and emit notifications
  watch test         Simulate a watch alert
  notify test        Test notification channels
  auth login         Store API key interactively
  auth status        Show auth/config status
  config get/set     Read/write config values

GLOBAL FLAGS:
  --json             JSON output
  --plain            Stable plain output
  -q, --quiet        Suppress non-essential text
  -v, --verbose      Extra diagnostics to stderr
  --no-input         Disable prompts
  --state-dir PATH   Override state directory
  --version          Print version
  -h, --help         Show help
`
}

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
		return errors.New("--from, --to, and --depart are required")
	}
	return nil
}

func (a App) resolveProvider(cfg config.Config) provider.Provider {
	switch strings.ToLower(cfg.Provider) {
	case "google-url", "google":
		return provider.GoogleURLProvider{}
	default:
		return provider.SerpAPIProvider{APIKey: cfg.SerpAPIKey}
	}
}

func (a App) cmdSearch(g globalFlags, args []string) error {
	fs, q := newSearchFlagSet("search")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validateQuery(*q); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	res, err := a.resolveProvider(cfg).Search(*q)
	if err != nil {
		return err
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

func (a App) watcherStore(stateOverride string) (watcher.Store, error) {
	dir, err := config.StateDir(stateOverride)
	if err != nil {
		return watcher.Store{}, err
	}
	return watcher.Store{Path: filepath.Join(dir, "watches.json")}, nil
}

func (a App) cmdWatch(g globalFlags, args []string) error {
	if len(args) == 0 {
		return errors.New("watch requires subcommand: create|list|enable|disable|delete|run|test")
	}
	sub := args[0]
	argv := args[1:]
	switch sub {
	case "create":
		return a.cmdWatchCreate(g, argv)
	case "list":
		return a.cmdWatchList(g, argv)
	case "enable":
		return a.cmdWatchSetEnabled(g, argv, true)
	case "disable":
		return a.cmdWatchSetEnabled(g, argv, false)
	case "delete":
		return a.cmdWatchDelete(g, argv)
	case "run":
		return a.cmdWatchRun(g, argv)
	case "test":
		return a.cmdWatchTest(g, argv)
	default:
		return fmt.Errorf("unknown watch subcommand %q", sub)
	}
}

func (a App) cmdWatchCreate(g globalFlags, args []string) error {
	fs, q := newSearchFlagSet("watch create")
	name := fs.String("name", "", "Watch name")
	target := fs.Int("target-price", 0, "Alert when price <= target")
	notifyTerminal := fs.Bool("notify-terminal", true, "Send terminal notifications")
	notifyEmail := fs.Bool("notify-email", false, "Send email notifications")
	emailTo := fs.String("email-to", "", "Email recipient")
	dryRun := fs.Bool("dry-run", false, "Preview watch without saving")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validateQuery(*q); err != nil {
		return err
	}
	if *name == "" {
		*name = fmt.Sprintf("%s-%s-%s", q.From, q.To, q.Depart)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if *emailTo == "" {
		*emailTo = cfg.DefaultNotifyEmail
	}
	w := model.Watch{
		ID:             fmt.Sprintf("w_%d", time.Now().UnixNano()),
		Name:           *name,
		Query:          *q,
		Enabled:        true,
		TargetPrice:    *target,
		NotifyTerminal: *notifyTerminal,
		NotifyEmail:    *notifyEmail,
		EmailTo:        *emailTo,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if *dryRun {
		return writeMaybeJSON(g, w)
	}
	store, err := a.watcherStore(g.StateDir)
	if err != nil {
		return err
	}
	ws, err := store.Load()
	if err != nil {
		return err
	}
	ws.Watches = append(ws.Watches, w)
	if err := store.Save(ws); err != nil {
		return err
	}
	return writeMaybeJSON(g, w)
}

func (a App) cmdWatchList(g globalFlags, args []string) error {
	fs := flag.NewFlagSet("watch list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := a.watcherStore(g.StateDir)
	if err != nil {
		return err
	}
	ws, err := store.Load()
	if err != nil {
		return err
	}
	sort.Slice(ws.Watches, func(i, j int) bool { return ws.Watches[i].CreatedAt.After(ws.Watches[j].CreatedAt) })
	if g.JSON {
		return writeJSON(ws.Watches)
	}
	if len(ws.Watches) == 0 {
		fmt.Println("No watches configured")
		return nil
	}
	for _, w := range ws.Watches {
		fmt.Printf("%s\t%s\t%s->%s\t%s\ttarget=%d\n", w.ID, w.Name, w.Query.From, w.Query.To, w.Query.Depart, w.TargetPrice)
	}
	return nil
}

func (a App) cmdWatchSetEnabled(g globalFlags, args []string, enabled bool) error {
	fs := flag.NewFlagSet("watch set-enabled", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	id := fs.String("id", "", "Watch ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *id == "" {
		return errors.New("--id is required")
	}
	store, err := a.watcherStore(g.StateDir)
	if err != nil {
		return err
	}
	ws, err := store.Load()
	if err != nil {
		return err
	}
	for i := range ws.Watches {
		if ws.Watches[i].ID != *id {
			continue
		}
		ws.Watches[i].Enabled = enabled
		ws.Watches[i].UpdatedAt = time.Now().UTC()
		if err := store.Save(ws); err != nil {
			return err
		}
		return writeMaybeJSON(g, ws.Watches[i])
	}
	return fmt.Errorf("watch not found: %s", *id)
}

func (a App) cmdWatchDelete(g globalFlags, args []string) error {
	fs := flag.NewFlagSet("watch delete", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	id := fs.String("id", "", "Watch ID")
	force := fs.Bool("force", false, "Delete without confirmation")
	confirm := fs.String("confirm", "", "Confirmation token (watch ID)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *id == "" {
		return errors.New("--id is required")
	}
	if !*force && *confirm != *id {
		return errors.New("destructive action: pass --force or --confirm with the watch ID")
	}
	if g.NoInput && !*force {
		return errors.New("--no-input requires --force for watch delete")
	}
	store, err := a.watcherStore(g.StateDir)
	if err != nil {
		return err
	}
	ws, err := store.Load()
	if err != nil {
		return err
	}
	filtered := make([]model.Watch, 0, len(ws.Watches))
	found := false
	for _, w := range ws.Watches {
		if w.ID == *id {
			found = true
			continue
		}
		filtered = append(filtered, w)
	}
	if !found {
		return fmt.Errorf("watch not found: %s", *id)
	}
	ws.Watches = filtered
	if err := store.Save(ws); err != nil {
		return err
	}
	return writeMaybeJSON(g, map[string]any{"deleted": *id})
}

func (a App) cmdWatchRun(g globalFlags, args []string) error {
	fs := flag.NewFlagSet("watch run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	watchID := fs.String("id", "", "Watch ID")
	runAll := fs.Bool("all", false, "Run all watches")
	once := fs.Bool("once", true, "Single pass")
	if err := fs.Parse(args); err != nil {
		return err
	}
	_ = once
	store, err := a.watcherStore(g.StateDir)
	if err != nil {
		return err
	}
	ws, err := store.Load()
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	p := a.resolveProvider(cfg)
	n := notify.Notifier{Config: cfg}
	alerts := []model.Alert{}

	for i := range ws.Watches {
		w := &ws.Watches[i]
		if !w.Enabled {
			continue
		}
		if *watchID != "" && w.ID != *watchID {
			continue
		}
		if *watchID == "" && !*runAll {
			continue
		}
		res, err := p.Search(w.Query)
		if err != nil {
			if g.Verbose {
				fmt.Fprintf(os.Stderr, "watch %s failed: %v\n", w.ID, err)
			}
			continue
		}
		lowest := 0
		currency := "USD"
		if len(res.Flights) > 0 {
			lowest = res.Flights[0].Price
			currency = res.Flights[0].Currency
		}
		reason := ""
		if w.TargetPrice > 0 && lowest > 0 && lowest <= w.TargetPrice {
			reason = fmt.Sprintf("price reached target <= %d", w.TargetPrice)
		} else if w.LastLowestPrice > 0 && lowest > 0 && lowest < w.LastLowestPrice {
			reason = fmt.Sprintf("price dropped from %d to %d", w.LastLowestPrice, lowest)
		}
		w.LastRunAt = time.Now().UTC()
		if lowest > 0 {
			w.LastLowestPrice = lowest
		}
		w.UpdatedAt = time.Now().UTC()
		if reason != "" {
			alert := model.Alert{
				WatchID:     w.ID,
				WatchName:   w.Name,
				TriggeredAt: time.Now().UTC(),
				Reason:      reason,
				LowestPrice: lowest,
				Currency:    currency,
				URL:         res.URL,
			}
			alerts = append(alerts, alert)
			if w.NotifyTerminal {
				n.SendTerminal(alert)
			}
			if w.NotifyEmail {
				_ = n.SendEmail(w.EmailTo, alert)
			}
		}
	}
	if err := store.Save(ws); err != nil {
		return err
	}
	if g.JSON {
		return writeJSON(alerts)
	}
	if len(alerts) == 0 {
		fmt.Println("No alerts triggered")
		return nil
	}
	fmt.Printf("Triggered %d alert(s)\n", len(alerts))
	return nil
}

func (a App) cmdWatchTest(g globalFlags, args []string) error {
	fs := flag.NewFlagSet("watch test", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	id := fs.String("id", "", "Watch ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *id == "" {
		return errors.New("--id is required")
	}
	store, err := a.watcherStore(g.StateDir)
	if err != nil {
		return err
	}
	ws, err := store.Load()
	if err != nil {
		return err
	}
	for _, w := range ws.Watches {
		if w.ID != *id {
			continue
		}
		alert := model.Alert{
			WatchID:     w.ID,
			WatchName:   w.Name,
			TriggeredAt: time.Now().UTC(),
			Reason:      "manual test",
			LowestPrice: w.TargetPrice,
			Currency:    firstOr(w.Query.Currency, "USD"),
			URL:         "https://www.google.com/travel/flights",
		}
		cfg, _ := config.Load()
		n := notify.Notifier{Config: cfg}
		if w.NotifyTerminal {
			n.SendTerminal(alert)
		}
		if w.NotifyEmail {
			_ = n.SendEmail(w.EmailTo, alert)
		}
		return writeMaybeJSON(g, alert)
	}
	return fmt.Errorf("watch not found: %s", *id)
}

func (a App) cmdNotify(g globalFlags, args []string) error {
	if len(args) == 0 || args[0] != "test" {
		return errors.New("notify supports only: notify test")
	}
	fs := flag.NewFlagSet("notify test", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	channel := fs.String("channel", "terminal", "terminal|email")
	to := fs.String("to", "", "Email recipient")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	n := notify.Notifier{Config: cfg}
	alert := model.Alert{
		WatchID:     "test",
		WatchName:   "test-notification",
		TriggeredAt: time.Now().UTC(),
		Reason:      "notification test",
		LowestPrice: 499,
		Currency:    "USD",
		URL:         "https://www.google.com/travel/flights",
	}
	switch *channel {
	case "terminal":
		n.SendTerminal(alert)
		return writeMaybeJSON(g, map[string]any{"ok": true, "channel": "terminal"})
	case "email":
		recipient := *to
		if recipient == "" {
			recipient = cfg.DefaultNotifyEmail
		}
		if err := n.SendEmail(recipient, alert); err != nil {
			return err
		}
		return writeMaybeJSON(g, map[string]any{"ok": true, "channel": "email", "to": recipient})
	default:
		return errors.New("--channel must be terminal or email")
	}
}

func (a App) cmdAuth(g globalFlags, args []string) error {
	if len(args) == 0 {
		return errors.New("auth requires subcommand: login|status")
	}
	switch args[0] {
	case "status":
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		status := map[string]any{
			"provider":        cfg.Provider,
			"serpapi_key":     cfg.SerpAPIKey != "",
			"smtp_configured": cfg.SMTPHost != "" && cfg.SMTPUsername != "" && cfg.SMTPPassword != "" && cfg.SMTPSender != "",
		}
		return writeMaybeJSON(g, status)
	case "login":
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		apiKey := fs.String("serpapi-key", "", "SerpAPI key")
		providerName := fs.String("provider", "", "Provider: serpapi|google-url")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *providerName != "" {
			cfg.Provider = *providerName
		}
		if *apiKey != "" {
			cfg.SerpAPIKey = *apiKey
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		return writeMaybeJSON(g, map[string]any{"ok": true, "provider": cfg.Provider})
	default:
		return fmt.Errorf("unknown auth subcommand %q", args[0])
	}
}

func (a App) cmdConfig(g globalFlags, args []string) error {
	if len(args) < 2 {
		return errors.New("usage: gflight config get <key> | gflight config set <key> <value>")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	switch args[0] {
	case "get":
		if len(args) != 2 {
			return errors.New("usage: gflight config get <key>")
		}
		val, ok := configGet(cfg, args[1])
		if !ok {
			return fmt.Errorf("unknown key %q", args[1])
		}
		if g.JSON {
			return writeJSON(map[string]string{"key": args[1], "value": val})
		}
		fmt.Println(val)
		return nil
	case "set":
		if len(args) != 3 {
			return errors.New("usage: gflight config set <key> <value>")
		}
		if err := configSet(&cfg, args[1], args[2]); err != nil {
			return err
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		return writeMaybeJSON(g, map[string]string{"ok": "true", "key": args[1]})
	default:
		return fmt.Errorf("unknown config action %q", args[0])
	}
}

func configGet(cfg config.Config, key string) (string, bool) {
	switch key {
	case "provider":
		return cfg.Provider, true
	case "serp_api_key":
		if cfg.SerpAPIKey == "" {
			return "", true
		}
		return "***", true
	case "smtp_host":
		return cfg.SMTPHost, true
	case "smtp_user":
		return cfg.SMTPUsername, true
	case "smtp_sender":
		return cfg.SMTPSender, true
	case "notify_email":
		return cfg.DefaultNotifyEmail, true
	default:
		return "", false
	}
}

func configSet(cfg *config.Config, key, value string) error {
	switch key {
	case "provider":
		cfg.Provider = value
	case "serp_api_key":
		cfg.SerpAPIKey = value
	case "smtp_host":
		cfg.SMTPHost = value
	case "smtp_port":
		var p int
		_, err := fmt.Sscanf(value, "%d", &p)
		if err != nil || p <= 0 {
			return errors.New("smtp_port must be positive integer")
		}
		cfg.SMTPPort = p
	case "smtp_user":
		cfg.SMTPUsername = value
	case "smtp_pass":
		cfg.SMTPPassword = value
	case "smtp_sender":
		cfg.SMTPSender = value
	case "notify_email":
		cfg.DefaultNotifyEmail = value
	default:
		return fmt.Errorf("unknown key %q", key)
	}
	return nil
}

func writeMaybeJSON(g globalFlags, v any) error {
	if g.JSON {
		return writeJSON(v)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func writeJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func firstOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
