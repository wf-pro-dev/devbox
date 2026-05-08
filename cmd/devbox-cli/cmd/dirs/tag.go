package dirs

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/internal/cmd/completion"
)

func TagCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tag <name> <tag>",
		Short: "Add a tag to a collection",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) >= 1 {
				return []string{}, cobra.ShellCompDirectiveDefault
			}
			return completion.DirCompletions(cmd, args, toComplete)
		},
		Example: `  devbox-cli dirs tag nginx prod
  devbox-cli dirs tag abcd1234 infra`,
		RunE: func(c *cobra.Command, args []string) error {
			dir, err := getDirectory(args[0])
			if err != nil {
				return err
			}
			u := internal.Server() + "/dirs/" + url.PathEscape(dir.Prefix) + "/tags"
			resp, err := internal.PostJSON(u, map[string]any{"tags": []string{args[1]}})
			if err != nil {
				return err
			}
			var result map[string][]string
			if err := internal.Decode(resp, &result); err != nil {
				return err
			}
			fmt.Printf("tags  %s\n", internal.FmtTags(dir.Tags))
			return nil
		},
	}
}

func UntagCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "untag <name|id> <tag>",
		Short: "Remove a tag from a collection",
		Args:  cobra.ExactArgs(2),
		Example: `  devbox-cli dirs untag nginx prod
  devbox-cli dirs untag abcd1234 infra`,
		RunE: func(c *cobra.Command, args []string) error {
			dir, err := getDirectory(args[0])
			if err != nil {
				return err
			}
			u := internal.Server() + "/dirs/" + url.PathEscape(dir.Prefix) + "/tags/" + url.PathEscape(args[1])
			resp, err := internal.Del(u)
			if err != nil {
				return err
			}
			if err := internal.CheckNoContent(resp); err != nil {
				return err
			}
			fmt.Printf("removed tag %q from %s\n", args[1], dir.Prefix)
			return nil
		},
	}
}
