package files

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/types"
)

func RollbackCmd() *cobra.Command {
	var force bool

	c := &cobra.Command{
		Use:   "rollback <id|path> <version>",
		Short: "Restore a file to a previous version",
		Args:  cobra.ExactArgs(2),
		Example: `  devbox-cli files rollback deploy.sh 2
  devbox-cli files rollback deploy.sh v2 --force`,
		RunE: func(c *cobra.Command, args []string) error {
			f, err := getFileMeta(args[0])
			if err != nil {
				return err
			}

			vStr := stripV(args[1])
			v, err := strconv.ParseInt(vStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid version %q: must be a number", args[1])
			}

			if !force {
				fmt.Printf("rollback %s from v%d to v%d? [y/N] ", f.Path, f.Version, v)
				var ans string
				fmt.Scanln(&ans)
				if ans != "y" && ans != "Y" {
					fmt.Println("aborted")
					return nil
				}
			}

			u := fmt.Sprintf("%s/files/%s/versions/%d/rollback",
				internal.Server(), url.PathEscape(f.ID), v)
			resp, err := internal.PostJSON(u, nil)
			if err != nil {
				return err
			}
			var updated types.File
			if err := internal.Decode(resp, &updated); err != nil {
				return err
			}
			fmt.Printf("rolled back  %s\n", updated.Path)
			fmt.Printf("now          v%d  (%s)\n", updated.Version, internal.ShortSHA(updated.Sha256))
			return nil
		},
	}
	c.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	return c
}
