package cli

import (
	"encoding/json"
	"fmt"
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

func firstOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
