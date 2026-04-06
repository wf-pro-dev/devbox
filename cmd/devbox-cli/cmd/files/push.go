package files

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/types"
)

func PushCmd() *cobra.Command {
	var desc, lang, tags, collection, path string

	c := &cobra.Command{
		Use:   "push <file>",
		Short: "Upload a file",
		Args:  cobra.ExactArgs(1),
		Example: `  devbox-cli files push deploy.sh
  devbox-cli files push deploy.sh --tags nginx,prod --lang bash
  devbox-cli files push nginx/default.conf --collection nginx --path nginx/conf.d/default.conf`,
		RunE: func(c *cobra.Command, args []string) error {
			localPath := args[0]
			absolutePath, err := filepath.Abs(localPath)
			if err != nil {
				return err
			}
			fields := map[string]string{
				"description": desc,
				"language":    lang,
				"tags":        tags,
				"collection":  collection,
				"path":        path,
				"local_path":  absolutePath,
			}
			resp, err := internal.UploadFile(internal.Server()+"/files", localPath, fields)
			if err != nil {
				return err
			}
			var f types.File
			if err := internal.Decode(resp, &f); err != nil {
				return err
			}
			fmt.Printf("uploaded  %s\n", f.Path)
			fmt.Printf("id        %s\n", internal.ShortID(f.ID))
			fmt.Printf("sha256    %s\n", internal.ShortSHA(f.Sha256))
			fmt.Printf("size      %s\n", internal.FmtSize(f.Size))
			if len(f.Tags) > 0 {
				fmt.Printf("tags      %s\n", internal.FmtTags(f.Tags))
			}
			return nil
		},
	}
	c.Flags().StringVar(&desc, "desc", "", "Description")
	c.Flags().StringVar(&lang, "lang", "", "Language (auto-detected if omitted)")
	c.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")
	c.Flags().StringVar(&collection, "collection", "", "Collection name or ID")
	c.Flags().StringVar(&path, "path", "", "Logical path on server (default: filename)")
	return c
}
