package dirs

import "github.com/spf13/cobra"

// NewCmd returns the "dirs" parent command with all subcommands attached.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dirs",
		Short: "Manage file directories",
		Long: `Create, sync, tag and send directories of files.

Directories are groups of files that share a path prefix (e.g. "nginx/" owns
all files whose path starts with "nginx/"). Directory names are unique.`,
	}

	cmd.AddCommand(
		LsCmd(),
		PushCmd(),
		PullCmd(),
		UpdateCmd(),
		DeleteCmd(),
		TagCmd(),
		UntagCmd(),
		DiffCmd(),
		SendCmd(),
	)

	return cmd
}
