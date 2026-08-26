package main

import (
	"fmt"
	"io"
	"os"

	"github.com/agisilaos/gflight/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr io.Writer) int {
	buildVersion := fmt.Sprintf("gflight %s (commit %s, built %s)", version, commit, date)
	app := cli.NewApp(buildVersion)
	if err := app.Run(args); err != nil {
		fmt.Fprintln(stderr, err)
		for _, hint := range cli.ErrorHints(err) {
			fmt.Fprintf(stderr, "next: %s\n", hint)
		}
		return cli.ExitCode(err)
	}
	return cli.ExitSuccess
}
