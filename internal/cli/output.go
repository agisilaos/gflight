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
		fmt.Fprintln(a.output(), joinPlainFields(pairs))
		return
	}
	out := make([]string, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, fmt.Sprintf("%s=%s", escapePlainField(pairs[i]), escapePlainField(pairs[i+1])))
	}
	fmt.Fprintln(a.output(), strings.Join(out, "\t"))
}

func (a App) writePlainTableHeader(cols ...string) {
	fmt.Fprintln(a.output(), joinPlainFields(cols))
}

func (a App) writePlainTableRow(cols ...string) {
	fmt.Fprintln(a.output(), joinPlainFields(cols))
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

// Plain fields use reversible escapes so user data cannot add rows or columns.
func escapePlainField(value string) string {
	var out strings.Builder
	for _, r := range value {
		switch r {
		case '\\':
			out.WriteString(`\\`)
		case '\t':
			out.WriteString(`\t`)
		case '\r':
			out.WriteString(`\r`)
		case '\n':
			out.WriteString(`\n`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&out, `\x%02x`, r)
			} else {
				out.WriteRune(r)
			}
		}
	}
	return out.String()
}

func joinPlainFields(fields []string) string {
	escaped := make([]string, len(fields))
	for i, field := range fields {
		escaped[i] = escapePlainField(field)
	}
	return strings.Join(escaped, "\t")
}
