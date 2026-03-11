package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/wf-pro-dev/devbox/cmd/devbox-cli/cmd/dirs"
	"github.com/wf-pro-dev/devbox/cmd/devbox-cli/cmd/files"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
)

var serverURL string

var rootCmd = &cobra.Command{
	Use:   "devbox-cli",
	Short: "devbox — manage files and collections on your devbox server",
	Long: `devbox-cli is the command-line interface for your self-hosted devbox server.

Files and collections are addressed by ID (UUID prefix) or by path/name.
When a name is ambiguous the CLI will tell you and ask for the full path or ID.`,
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if serverURL != "" {
			internal.SetServer(serverURL)
		}
	},
}

// Execute is called by main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "",
		"devbox server URL (overrides $DEVBOX_SERVER)")

	rootCmd.AddCommand(files.NewCmd())
	rootCmd.AddCommand(dirs.NewCmd())
	rootCmd.AddCommand(newPeersCmd())
}
