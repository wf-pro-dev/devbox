package files

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/internal/db"
)

func LogCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "log <id|path>",
		Aliases: []string{"history", "versions"},
		Short:   "Show version history",
		Args:    cobra.ExactArgs(1),
		Example: `  devbox-cli files log deploy.sh
  devbox-cli files log abcd1234`,
		RunE: func(c *cobra.Command, args []string) error {
			f, err := getFileMeta(args[0])
			if err != nil {
				return err
			}

			u := internal.Server() + "/files/" + url.PathEscape(f.ID) + "/versions"
			resp, err := internal.GetJSON(u)
			if err != nil {
				return err
			}
			var versions []db.Version
			if err := internal.Decode(resp, &versions); err != nil {
				return err
			}

			fmt.Printf("history for %s (current v%d)\n\n", f.Path, f.Version)

			if len(versions) == 0 {
				fmt.Println("no version history")
				return nil
			}

			t := internal.Tw()
			fmt.Fprintln(t, "VER\tSIZE\tSHA256\tDATE\tMESSAGE")
			for _, v := range versions {
				msg := v.Message
				if msg == "" {
					msg = "-"
				}
				fmt.Fprintf(t, "v%d\t%s\t%s\t%s\t%s\n",
					v.Version,
					internal.FmtSize(v.Size),
					internal.ShortSHA(v.Sha256),
					internal.FmtDate(v.CreatedAt),
					msg,
				)
			}
			return t.Flush()
		},
	}
}
