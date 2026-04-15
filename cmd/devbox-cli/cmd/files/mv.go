package files

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	completion "github.com/wf-pro-dev/devbox/internal/cmd/completion"
	"github.com/wf-pro-dev/devbox/types"
)

func MvCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "mv <id|path> <new-path>",
		Short:             "Rename or move a file to a new path",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: completion.FileCompletions,
		Example: `  devbox-cli files mv deploy.sh scripts/deploy.sh
  devbox-cli files mv abcd1234 nginx/deploy.sh`,
		RunE: func(c *cobra.Command, args []string) error {
			f, err := getFileMeta(args[0])
			if err != nil {
				return err
			}
			u := internal.Server() + "/files/" + url.PathEscape(f.ID) + "/move"
			resp, err := internal.PostJSON(u, map[string]string{"path": args[1]})
			if err != nil {
				return err
			}
			var updated types.File
			if err := internal.Decode(resp, &updated); err != nil {
				return err
			}
			fmt.Printf("moved  %s  ->  %s\n", f.Path, updated.Path)
			return nil
		},
	}
}
