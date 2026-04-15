package files

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	completion "github.com/wf-pro-dev/devbox/internal/cmd/completion"
	"github.com/wf-pro-dev/devbox/types"
)

func UpdateCmd() *cobra.Command {
	var message string

	c := &cobra.Command{
		Use:   "update <id|path> <local-file>",
		Short: "Update file content (creates a new version)",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) >= 1 {
				return []string{}, cobra.ShellCompDirectiveDefault
			}
			return completion.FileCompletions(cmd, args, toComplete)
		},
		Example: `  devbox-cli files update deploy.sh ./deploy.sh
  devbox-cli files update abcd1234 ./new-deploy.sh -m "fix: correct db host"`,
		RunE: func(c *cobra.Command, args []string) error {
			f, err := getFileMeta(args[0])
			if err != nil {
				return err
			}

			u := internal.Server() + "/files/" + url.PathEscape(f.ID)
			resp, err := internal.UploadFileUpdate(u, args[1], message)
			if err != nil {
				return err
			}

			var result struct {
				Result string     `json:"result"`
				File   types.File `json:"file"`
			}
			if err := internal.Decode(resp, &result); err != nil {
				return err
			}

			switch result.Result {
			case "unchanged":
				fmt.Printf("unchanged  %s (content identical)\n", result.File.Path)
			default:
				fmt.Printf("updated    %s\n", result.File.Path)
				fmt.Printf("version    v%d\n", result.File.Version)
				fmt.Printf("sha256     %s\n", internal.ShortSHA(result.File.Sha256))
				fmt.Printf("size       %s\n", internal.FmtSize(result.File.Size))
			}
			return nil
		},
	}
	c.Flags().StringVarP(&message, "message", "m", "", "Version message")
	return c
}
