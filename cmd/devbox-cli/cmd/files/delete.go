package files

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	completion "github.com/wf-pro-dev/devbox/internal/cmd/completion"
)

func DeleteCmd() *cobra.Command {
	var force bool

	c := &cobra.Command{
		Use:               "rm <id|path>",
		Aliases:           []string{"delete"},
		Short:             "Delete a file",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completion.FileCompletions,
		Example: `  devbox-cli files rm deploy.sh
  devbox-cli files rm abcd1234 --force`,
		RunE: func(c *cobra.Command, args []string) error {
			f, err := getFileMeta(args[0])
			if err != nil {
				return err
			}

			if !force {
				fmt.Printf("delete %s (v%d, %s)? [y/N] ", f.Path, f.Version, internal.FmtSize(f.Size))
				var ans string
				fmt.Scanln(&ans)
				if ans != "y" && ans != "Y" {
					fmt.Println("aborted")
					return nil
				}
			}

			u := internal.Server() + "/files/" + url.PathEscape(f.ID)
			resp, err := internal.Del(u)
			if err != nil {
				return err
			}
			if err := internal.CheckNoContent(resp); err != nil {
				return err
			}
			fmt.Printf("deleted   %s\n", f.Path)
			return nil
		},
	}
	c.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	return c
}
