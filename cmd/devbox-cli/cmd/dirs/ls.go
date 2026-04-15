package dirs

import (
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/internal/cmd/completion"
	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/types"
)

func LsCmd() *cobra.Command {
	var tag string

	c := &cobra.Command{
		Use:               "ls",
		Short:             "List collections",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completion.DirCompletions,
		Example: `  devbox-cli dirs ls
  devbox-cli dirs ls --tag nginx`,
		RunE: func(c *cobra.Command, args []string) error {
			u := internal.Server() + "/dirs"
			if tag != "" {
				u += "?tag=" + url.QueryEscape(tag)
			}
			if len(args) > 0 {
				u += "?prefix=" + url.QueryEscape(args[0])
			}
			resp, err := internal.GetJSON(u)
			if err != nil {
				return err
			}
			var dirs []types.Directory[db.File]
			if err := internal.Decode(resp, &dirs); err != nil {
				return err
			}
			if len(dirs) == 0 {
				fmt.Println("no collections found")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PREFIX\tFILE_COUNT\tTAGS")
			fmt.Fprintln(w, "─────────────────────────────\t─────────\t───────-----")
			for _, d := range dirs {
				fmt.Fprintf(w, "%s\t%d\t%s\n",
					d.Prefix,
					d.FileCount,
					internal.FmtTags(d.Tags),
				)
			}
			err = w.Flush()
			if err != nil {
				return err
			}
			fmt.Printf("\n%d directory(ies)\n", len(dirs))

			return nil
		},
	}
	c.Flags().StringVarP(&tag, "tag", "t", "", "Filter by tag")
	return c
}
