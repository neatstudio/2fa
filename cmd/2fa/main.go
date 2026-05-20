package main

import (
	"fmt"
	"os"

	"github.com/gouki/tools/2fa/internal/cli"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Fprintf(os.Stdout, "2fa %s (%s, %s)\n", version, commit, date)
		return
	}
	if err := cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
