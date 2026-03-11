package dirs

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
)

func UpdateCmd() *cobra.Command {
	var message string

	c := &cobra.Command{
		Use:   "update <name|id> <local-dir>",
		Short: "Sync a local directory to an existing collection",
		Long: `Syncs a local directory to the collection on the server.
New files are added. Existing files with changed content get a new version.
Files only on the server are left untouched (use 'dirs delete' to remove them).`,
		Args: cobra.ExactArgs(2),
		Example: `  devbox-cli dirs update nginx ./nginx
  devbox-cli dirs update nginx ./nginx -m "update upstream block"`,
		RunE: func(c *cobra.Command, args []string) error {
			dir, err := getDirectory(args[0])
			if err != nil {
				return err
			}

			files, err := internal.WalkDir(args[1])
			if err != nil {
				return fmt.Errorf("walk %s: %w", args[1], err)
			}
			if len(files) == 0 {
				return fmt.Errorf("no files found in %s", args[1])
			}

			fields := map[string]string{"message": message}
			fmt.Printf("syncing %d files to %s...\n", len(files), dir.Prefix)

			u := internal.Server() + "/dirs/" + url.PathEscape(dir.Prefix)
			resp, err := internal.SyncDirFiles(u, fields, files)
			if err != nil {
				return err
			}

			var result struct {
				Updated   []string `json:"updated"`
				Added     []string `json:"added"`
				Unchanged []string `json:"unchanged"`
			}
			if err := internal.Decode(resp, &result); err != nil {
				return err
			}

			for _, p := range result.Added {
				fmt.Printf("  added    %s\n", p)
			}
			for _, p := range result.Updated {
				fmt.Printf("  updated  %s\n", p)
			}
			fmt.Printf("\n%d added, %d updated, %d unchanged\n",
				len(result.Added), len(result.Updated), len(result.Unchanged))
			return nil
		},
	}
	c.Flags().StringVarP(&message, "message", "m", "", "Version message for updated files")
	return c
}
