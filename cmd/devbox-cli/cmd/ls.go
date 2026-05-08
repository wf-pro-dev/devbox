package cmd

import (
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/internal/cmd/completion"
	"github.com/wf-pro-dev/devbox/types"
)

func LsCmd() *cobra.Command {
	var dir string
	var recursive bool

	c := &cobra.Command{
		Use:   "ls",
		Short: "List files and directories",
		Example: `  devbox-cli files ls
  devbox-cli ls myapp
  devbox-cli ls myapp/config
  devbox-cli ls myapp --recursive
  devbox-cli ls nginx`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completion.DirCompletions,
		RunE: func(c *cobra.Command, args []string) error {
			// Build the URL.  A single GET /dirs/{dir} call returns both files
			// and virtual sub-directories — no extra request per sub-directory.
			if len(args) > 0 {
				dir = args[0]
			}

			var u string
			if dir == "" {
				u = fmt.Sprintf("%s/dirs", internal.Server())
			} else {
				u = fmt.Sprintf("%s/dirs/%s", internal.Server(), url.PathEscape(dir))
			}

			q := make(url.Values)

			if recursive {
				q.Set("depth", "all")
			}
			if len(q) > 0 {
				u += "?" + q.Encode()
			}

			resp, err := internal.GetJSON(u)
			if err != nil {
				return err
			}

			var listing types.DirListing
			if err := internal.Decode(resp, &listing); err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

			// Header line mirrors `ls -lh`: total count of direct children.
			fmt.Fprintf(w, "total %d\n", len(listing.Entries))

			for _, e := range listing.Entries {
				if e.IsDir {
					PrintDirEntry(w, e)
				} else if e.File != nil {
					PrintFileEntry(w, e)
				}
			}

			return w.Flush()
		},
	}

	c.Flags().BoolVarP(&recursive, "recursive", "R", false, "List all files recursively")
	c.RegisterFlagCompletionFunc("dir", completion.DirCompletions)
	return c
}

// PrintDirEntry prints a single virtual sub-directory entry.
//
// Output columns:
//
//	TYPE  NAME/  FILE_COUNT  TAGS
//
// Example:
//
//	d  config/   3
func PrintDirEntry(w *tabwriter.Writer, e types.DirEntry) {
	tags := ""
	// Directory-level tags are not stored on DirEntry directly; they would need
	// a separate fetch.  Print a placeholder dash for now.  The web UI shows
	// them in the column-view side-panel instead.
	fmt.Fprintf(w, "d\t%s/\t%d\t%s\n",
		e.Name,
		e.FileCount,
		tags,
	)
}

// PrintFileEntry prints a single file entry.
//
// Output columns (mirrors `ls -lh` extended with Devbox metadata):
//
//	TYPE  ID        VERSION  LANG    SIZE    TAGS  UPLOADED_BY   DATE        NAME
//
// Example:
//
//   - a1b2c3d4  v3  bash  12 KB  [@nginx]  laptop        2024-01-15  nginx.conf
func PrintFileEntry(w *tabwriter.Writer, e types.DirEntry) {
	f := e.File
	if f == nil {
		return
	}

	fmt.Fprintf(w, "-\t%s\tv%d\t%s\t%s\t%s\t%s\t%s\n",
		f.ID[:8],
		f.Version,
		f.Language,
		internal.FmtSize(f.Size),
		internal.Truncate(f.UploadedBy, 12),
		internal.FmtDate(f.CreatedAt),
		f.FileName,
	)
}

// joinTags formats a tag slice as "tag1 @tag2 @tag3".
func joinTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	out := tags[0]
	for _, t := range tags[1:] {
		out += " @" + t
	}
	return out
}
