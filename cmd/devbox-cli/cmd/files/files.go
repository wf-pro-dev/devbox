package files

import "github.com/spf13/cobra"

// NewCmd returns the "files" parent command with all subcommands attached.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files",
		Short: "Manage individual files",
		Long: `Upload, download, tag, version and deliver individual files.

Files are addressed by UUID (or prefix), full path, or unique filename.
If a filename matches more than one file the CLI will ask you to use the path or ID.`,
	}

	cmd.AddCommand(
		LsCmd(),
		PushCmd(),
		PullCmd(),
		InfoCmd(),
		EditCmd(),
		UpdateCmd(),
		DeleteCmd(),
		TagCmd(),
		UntagCmd(),
		MvCmd(),
		CpCmd(),
		LogCmd(),
		DiffCmd(),
		RollbackCmd(),
		SendCmd(),
	)

	return cmd
}
