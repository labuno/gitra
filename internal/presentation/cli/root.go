// Package cli is the Presentation adapter for the command line. It only calls
// application services and never touches storage, git or ssh directly.
package cli

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/spf13/cobra"

	"github.com/zhanhd/gitra/internal/bootstrap"
)

// version is overridable at build time via -ldflags "-X .../cli.version=...".
var version = "dev"

// App carries the wired application plus the IO streams used by commands.
type App struct {
	App    *bootstrap.App
	Stdout io.Writer
	Stderr io.Writer

	verbose     bool
	showVersion bool
	journal     *log.Logger
}

// New builds the CLI adapter.
func New(application *bootstrap.App, stdout, stderr io.Writer) *App {
	return &App{App: application, Stdout: stdout, Stderr: stderr}
}

// Execute runs one command line and returns the process exit code.
func (a *App) Execute(ctx context.Context, args []string) int {
	if ctx == nil {
		ctx = context.Background()
	}
	root := a.newRootCmd()
	root.SetArgs(args)
	root.SetOut(a.Stdout)
	root.SetErr(a.Stderr)

	if _, _, err := root.Find(args); err != nil {
		fmt.Fprintln(a.Stderr, err)
		return ExitCodeFor(fmt.Errorf("%w: %v", errUsage, err))
	}
	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(a.Stderr, err)
		return ExitCodeFor(err)
	}
	return 0
}

func (a *App) newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "gitra",
		Short:         "Bind each repository to a Git account, then use plain git",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(*cobra.Command, []string) {
			a.ensureJournal()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if a.showVersion {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "gitra %s\n", version)
				return err
			}
			return cmd.Help()
		},
	}
	root.Flags().BoolVar(&a.showVersion, "version", false, "print version and exit")
	root.PersistentFlags().BoolVar(&a.verbose, "verbose", false, "print diagnostic output to stderr")
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return fmt.Errorf("%w: %v", errUsage, err)
	})
	root.AddCommand(newVersionCmd())
	root.AddCommand(a.newAccountCmd())
	root.AddCommand(a.newBindCmd())
	root.AddCommand(a.newUnbindCmd())
	root.AddCommand(a.newStatusCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the gitra version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "gitra %s\n", version)
			return err
		},
	}
}

// ensureJournal lazily binds the diagnostic logger to the current stderr.
func (a *App) ensureJournal() {
	if a.journal == nil {
		a.journal = log.New(a.Stderr, "gitra: ", 0)
	}
}

// logf writes diagnostics only with --verbose; never log secrets here.
func (a *App) logf(format string, args ...any) {
	if !a.verbose {
		return
	}
	a.ensureJournal()
	a.journal.Printf(format, args...)
}
