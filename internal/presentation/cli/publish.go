package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type publishJSON struct {
	SchemaVersion int    `json:"schema_version"`
	Branch        string `json:"branch"`
	Remote        string `json:"remote"`
	Path          string `json:"path"`
}

// newPublishCmd exposes the one-time first upload. This is deliberately narrow:
// gitra does not proxy git (baseline §7); it only performs the first push of a
// bound repository whose remote is still empty.
func (a *App) newPublishCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "publish [path]",
		Short: "Upload a bound repository for the first time",
		Long: "Pushes the current branch to origin and sets its upstream.\n" +
			"Only available when the repository is bound and the remote is still empty;\n" +
			"afterwards use git (or your editor) as usual.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.App.Publish == nil {
				return fmt.Errorf("publish is not available in this build")
			}
			path, err := pathArg(args)
			if err != nil {
				return err
			}
			result, err := a.App.Publish.FirstPublish(cmd.Context(), path)
			if err != nil {
				return err
			}
			a.logf("published %s to %s", result.Branch, result.Remote)
			if jsonOut {
				return WriteJSON(a.Stdout, publishJSON{
					SchemaVersion: schemaVersion, Branch: result.Branch, Remote: result.Remote, Path: path,
				})
			}
			_, err = fmt.Fprintf(a.Stdout, "已上传 %s → %s（首次）\n之后在编辑器里同步即可。\n", result.Branch, result.Remote)
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON")
	return cmd
}
