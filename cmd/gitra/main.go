// Command gitra is the process entry point for the gitra Git identity manager.
//
// Running it without arguments inside a terminal opens the TUI; subcommands are
// available for scripts and agents.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/zhanhd/gitra/internal/bootstrap"
	"github.com/zhanhd/gitra/internal/presentation/cli"
	"github.com/zhanhd/gitra/internal/presentation/tui"
)

func main() {
	application, err := bootstrap.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	args := os.Args[1:]
	if len(args) == 0 && isTerminal(os.Stdin) && isTerminal(os.Stdout) {
		if err := tui.Run(application); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	os.Exit(cli.New(application, os.Stdout, os.Stderr).Execute(context.Background(), args))
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
