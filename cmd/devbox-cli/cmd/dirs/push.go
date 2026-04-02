package dirs

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/types"
)

func PushCmd() *cobra.Command {
	var desc, tags string

	c := &cobra.Command{
		Use:   "push <local-dir>",
		Short: "Upload a local directory as a new collection",
		Args:  cobra.ExactArgs(1),
		Example: `  devbox-cli dirs push ./nginx --name nginx
  devbox-cli dirs push ./scripts --name scripts --desc "Deploy scripts" --tags prod,deploy`,
		RunE: func(c *cobra.Command, args []string) error {
			name, _ := c.Flags().GetString("name")
			if name == "" {
				name = filepath.Base(args[0])
			}

			files, err := internal.WalkDir(args[0])
			if err != nil {
				return fmt.Errorf("walk %s: %w", args[0], err)
			}
			if len(files) == 0 {
				return fmt.Errorf("no files found in %s", args[0])
			}

			fields := map[string]string{
				"name":        name,
				"description": desc,
				"tags":        tags,
			}

			fmt.Printf("uploading %d files...\n", len(files))
			resp, err := internal.UploadDirFiles(internal.Server()+"/dirs", fields, files)
			if err != nil {
				return err
			}

			var result types.Directory[types.File]
			if err := internal.Decode(resp, &result); err != nil {
				return err
			}
			fmt.Printf("created    %s\n", result.Prefix)
			fmt.Printf("files      %d\n", result.FileCount)
			if len(result.Tags) > 0 {
				fmt.Printf("tags       %s\n", internal.FmtTags(result.Tags))
			}
			return nil
		},
	}
	c.Flags().String("name", "", "Collection name (required, must be unique)")
	c.Flags().StringVar(&desc, "desc", "", "Description")
	c.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")
	return c
}
