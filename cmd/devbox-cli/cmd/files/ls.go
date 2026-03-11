package files

import (
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/types"
)

func LsCmd() *cobra.Command {
	var tag, lang, dir string

	c := &cobra.Command{
		Use:   "ls",
		Short: "List files",
		Example: `  devbox-cli files ls
  devbox-cli files ls --tag nginx
  devbox-cli files ls --lang bash
  devbox-cli files ls --dir myapp`,
		RunE: func(c *cobra.Command, args []string) error {
			u := internal.Server() + "/files"
			q := url.Values{}
			if tag != "" {
				q.Set("tag", tag)
			}
			if lang != "" {
				q.Set("lang", lang)
			}
			if dir != "" {
				q.Set("dir", dir)
			}
			if len(q) > 0 {
				u += "?" + q.Encode()
			}

			resp, err := internal.GetJSON(u)
			if err != nil {
				return err
			}
			var files []types.File
			if err := internal.Decode(resp, &files); err != nil {
				return err
			}
			if len(files) == 0 {
				fmt.Println("no files found")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tVER\tLANG\tSIZE\tTAGS\tUPLOADED BY\tCREATED")
			fmt.Fprintln(w, "──────────\t────────────────────────────\t───\t──────────\t───────\t────────────────\t────────────\t────────────────")
			for _, f := range files {
				fmt.Fprintf(w, "%s\t%s\tv%d\t%s\t%s\t%s\t%s\t%s\n",
					f.ID[:8],
					internal.Truncate(f.FileName, 28),
					f.Version,
					f.Language,
					internal.FmtSize(f.Size),
					internal.FmtTags(f.Tags),
					internal.Truncate(f.UploadedBy, 12),
					internal.FmtDate(f.CreatedAt),
				)
			}
			err = w.Flush()
			if err != nil {
				return err
			}
			fmt.Printf("\n%d file(s)\n", len(files))

			return nil
		},
	}
	c.Flags().StringVar(&tag, "tag", "", "Filter by tag")
	c.Flags().StringVar(&lang, "lang", "", "Filter by language")
	c.Flags().StringVar(&dir, "dir", "", "Filter by collection name or ID")
	return c
}
