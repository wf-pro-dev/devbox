package files

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	completion "github.com/wf-pro-dev/devbox/internal/cmd/completion"
)

func TagCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tag <id|path> <tag>",
		Short: "Add a tag to a file",
		Args:  cobra.ExactArgs(2),
		Example: `  devbox-cli files tag deploy.sh prod
  devbox-cli files tag abcd1234 nginx`,
		RunE: func(c *cobra.Command, args []string) error {
			f, err := getFileMeta(args[0])
			if err != nil {
				return err
			}
			u := internal.Server() + "/files/" + url.PathEscape(f.ID) + "/tags"
			resp, err := internal.PostJSON(u, map[string]any{"tags": []string{args[1]}})
			if err != nil {
				return err
			}
			var result map[string][]string
			if err := internal.Decode(resp, &result); err != nil {
				return err
			}
			fmt.Printf("tags  %s\n", internal.FmtTags(result["tags"]))
			return nil
		},
	}
}

func UntagCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "untag <id|path> <tag>",
		Short: "Remove a tag from a file",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) >= 1 {
				return []string{}, cobra.ShellCompDirectiveDefault
			}
			return completion.FileCompletions(cmd, args, toComplete)
		},
		Example: `  devbox-cli files untag deploy.sh prod
  devbox-cli files untag abcd1234 nginx`,
		RunE: func(c *cobra.Command, args []string) error {
			f, err := getFileMeta(args[0])
			if err != nil {
				return err
			}
			u := internal.Server() + "/files/" + url.PathEscape(f.ID) + "/tags/" + url.PathEscape(args[1])
			resp, err := internal.Del(u)
			if err != nil {
				return err
			}
			if err := internal.CheckNoContent(resp); err != nil {
				return err
			}
			fmt.Printf("removed tag %q from %s\n", args[1], f.Path)
			return nil
		},
	}
}
