package dirs

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/types"
)

func PullCmd() *cobra.Command {
	var out string

	c := &cobra.Command{
		Use:   "pull <name|id>",
		Short: "Download a collection, recreating the directory structure locally",
		Args:  cobra.ExactArgs(1),
		Example: `  devbox-cli dirs pull nginx
  devbox-cli dirs pull nginx --out /tmp/nginx-backup`,
		RunE: func(c *cobra.Command, args []string) error {
			// Get collection metadata + file list
			dir, err := getDirectory(args[0])
			if err != nil {
				return err
			}

			destRoot := out
			if destRoot == "" {
				destRoot = dir.Prefix
			}

			fmt.Printf("pulling %d files -> %s/\n", dir.FileCount, destRoot)
			ok, fail := 0, 0

			for _, f := range dir.Files {
				// Strip the collection prefix to get the relative path
				rel := strings.TrimPrefix(f.Path, dir.Prefix)
				destPath := filepath.Join(destRoot, filepath.FromSlash(rel))

				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					internal.Warn("mkdir %s: %v", filepath.Dir(destPath), err)
					fail++
					continue
				}

				u := internal.Server() + "/files/" + url.PathEscape(f.ID)
				resp, err := http.Get(u)
				if err != nil {
					internal.Warn("download %s: %v", f.Path, err)
					fail++
					continue
				}

				out, err := os.Create(destPath)
				if err != nil {
					resp.Body.Close()
					internal.Warn("create %s: %v", destPath, err)
					fail++
					continue
				}

				_, copyErr := io.Copy(out, resp.Body)
				resp.Body.Close()
				out.Close()

				if copyErr != nil {
					internal.Warn("write %s: %v", destPath, copyErr)
					fail++
					continue
				}

				fmt.Printf("  %s\n", rel)
				ok++
			}

			fmt.Printf("\n%d ok, %d failed\n", ok, fail)
			return nil
		},
	}
	c.Flags().StringVar(&out, "out", "", "Output directory (default: collection name)")
	return c
}

// getCollection fetches collection metadata and its file list.
func getDirectory(nameOrID string) (*types.Directory, error) {
	u := internal.Server() + "/dirs/" + url.PathEscape(nameOrID)
	resp, err := internal.GetJSON(u)
	if err != nil {
		return nil, err
	}
	var result types.Directory
	if err := internal.Decode(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
