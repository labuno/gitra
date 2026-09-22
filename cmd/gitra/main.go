// Command gitra is the process entry point for the gitra Git identity manager.
//
// The V1.0 core is a library-first milestone: this entry point only exposes
// build wiring (version reporting). CLI commands and the TUI arrive in later
// plans (see docs/plans/gitra-v1-core-plan.md).
package main

import (
	"flag"
	"fmt"
	"os"
)

// version is overridable at build time via:
//
//	go build -ldflags "-X main.version=1.2.3" ./cmd/gitra
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Fprintf(os.Stdout, "gitra %s\n", version)
		return
	}

	// Not an error: the command surface is intentionally absent in the core
	// milestone. Exit code 2 matches the baseline CLI invalid-input contract.
	fmt.Fprintln(os.Stderr, "gitra: command surface not available yet (V1.0 core skeleton)")
	os.Exit(2)
}
