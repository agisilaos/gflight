package cli

import (
	"encoding/json"
	"fmt"
	"strings"
)

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

func writePlainKV(pairs ...string) {
	if len(pairs)%2 != 0 {
		fmt.Println(strings.Join(pairs, "\t"))
		return
	}
	out := make([]string, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, fmt.Sprintf("%s=%s", pairs[i], pairs[i+1]))
	}
	fmt.Println(strings.Join(out, "\t"))
}

func writePlainTableHeader(cols ...string) {
	fmt.Println(strings.Join(cols, "\t"))
}

func writePlainTableRow(cols ...string) {
	fmt.Println(strings.Join(cols, "\t"))
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
