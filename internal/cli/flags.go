package cli

import "flag"

// parseNamedFlags rejects leftovers before handlers can act on partially parsed input.
func parseNamedFlags(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return newExitError(ExitInvalidUsage, "%v", err)
	}
	if fs.NArg() != 0 {
		return newExitError(ExitInvalidUsage, "unexpected arguments; use named flags for %s", fs.Name())
	}
	return nil
}
