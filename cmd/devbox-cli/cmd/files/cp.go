package files

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	completion "github.com/wf-pro-dev/devbox/internal/cmd/completion"
	"github.com/wf-pro-dev/devbox/internal/db"
)

func CpCmd() *cobra.Command {
	var collection string

	c := &cobra.Command{
		Use:   "cp <id|path> <new-path>",
		Short: "Copy a file to a new path (shares blob, no disk copy)",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) >= 1 {
				return []string{}, cobra.ShellCompDirectiveDefault
			}
			return completion.FileCompletions(cmd, args, toComplete)
		},
		Example: `  devbox-cli files cp deploy.sh deploy-backup.sh
  devbox-cli files cp abcd1234 nginx/deploy.sh --collection nginx`,
		RunE: func(c *cobra.Command, args []string) error {
			f, err := getFileMeta(args[0])
			if err != nil {
				return err
			}
			u := internal.Server() + "/files/" + url.PathEscape(f.ID) + "/copy"
			body := map[string]string{"path": args[1]}
			if collection != "" {
				body["collection"] = collection
			}
			resp, err := internal.PostJSON(u, body)
			if err != nil {
				return err
			}
			var newFile db.File
			if err := internal.Decode(resp, &newFile); err != nil {
				return err
			}
			fmt.Printf("copied  %s  ->  %s\n", f.Path, newFile.Path)
			fmt.Printf("id      %s\n", internal.ShortID(newFile.ID))
			return nil
		},
	}
	c.Flags().StringVar(&collection, "collection", "", "Target collection name or ID")
	return c
}
