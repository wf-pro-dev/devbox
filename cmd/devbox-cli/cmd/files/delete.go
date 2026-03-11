package files

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
)

func DeleteCmd() *cobra.Command {
	var force bool

	c := &cobra.Command{
		Use:     "delete <id|path>",
		Aliases: []string{"rm"},
		Short:   "Delete a file",
		Args:    cobra.ExactArgs(1),
		Example: `  devbox-cli files delete deploy.sh
  devbox-cli files delete abcd1234 --force`,
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
