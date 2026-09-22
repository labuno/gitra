// Command gitra is the process entry point for the gitra Git identity manager.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/zhanhd/gitra/internal/bootstrap"
	"github.com/zhanhd/gitra/internal/presentation/cli"
)

func main() {
	application, err := bootstrap.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	adapter := cli.New(application, os.Stdout, os.Stderr)
	os.Exit(adapter.Execute(context.Background(), os.Args[1:]))
}
