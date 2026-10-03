package cli

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (a App) writeMaybeJSON(g globalFlags, v any) error {
	if g.JSON {
		return a.writeJSON(v)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(a.output(), string(b))
	return nil
}

func (a App) writeJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(a.output(), string(b))
	return nil
}

func (a App) writePlainKV(pairs ...string) {
	if len(pairs)%2 != 0 {
		fmt.Fprintln(a.output(), strings.Join(pairs, "\t"))
		return
	}
	out := make([]string, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, fmt.Sprintf("%s=%s", pairs[i], pairs[i+1]))
	}
	fmt.Fprintln(a.output(), strings.Join(out, "\t"))
}

func (a App) writePlainTableHeader(cols ...string) {
	fmt.Fprintln(a.output(), strings.Join(cols, "\t"))
}

func (a App) writePlainTableRow(cols ...string) {
	fmt.Fprintln(a.output(), strings.Join(cols, "\t"))
}

func firstOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func boolToPlain(v any) string {
	b, ok := v.(bool)
	if !ok {
		return "false"
	}
	if b {
		return "true"
	}
	return "false"
}
