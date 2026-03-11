package dirs

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
)

func DeleteCmd() *cobra.Command {
	var force bool

	c := &cobra.Command{
		Use:     "delete <name|id>",
		Aliases: []string{"rm"},
		Short:   "Delete a collection and all its files",
		Args:    cobra.ExactArgs(1),
		Example: `  devbox-cli dirs delete nginx
  devbox-cli dirs delete nginx --force`,
		RunE: func(c *cobra.Command, args []string) error {
			dir, err := getDirectory(args[0])
			if err != nil {
				return err
			}

			if !force {
				fmt.Printf("delete collection %q (%d files)? [y/N] ", dir.Prefix, dir.FileCount)
				var ans string
				fmt.Scanln(&ans)
				if ans != "y" && ans != "Y" {
					fmt.Println("aborted")
					return nil
				}
			}

			u := internal.Server() + "/dirs/" + url.PathEscape(dir.Prefix)
			resp, err := internal.Del(u)
			if err != nil {
				return err
			}
			if err := internal.CheckNoContent(resp); err != nil {
				return err
			}
			fmt.Printf("deleted  %s (%d files)\n", dir.Prefix, dir.FileCount)
			return nil
		},
	}
	c.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	return c
}
