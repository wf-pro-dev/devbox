package files

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	completion "github.com/wf-pro-dev/devbox/internal/cmd/completion"
	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/types"
)

func PullCmd() *cobra.Command {
	var out string
	var version int

	c := &cobra.Command{
		Use:               "pull <id|path>",
		Short:             "Download a file",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completion.FileCompletions,
		Example: `  devbox-cli files pull deploy.sh
  devbox-cli files pull abcd1234
  devbox-cli files pull nginx/conf.d/default.conf --out /tmp/
  devbox-cli files pull deploy.sh --version 2`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/files/%s", internal.Server(), url.PathEscape(id)), nil)
			if err != nil {
				return err
			}
			if version > 0 {
				// Pull a specific version: get the version's sha256 from metadata,
				// then download the blob via the rollback sha. For now we stream
				// directly — the server returns the current blob; version-specific
				// pull requires fetching /versions first.
				meta, err := getFileMeta(id)
				if err != nil {
					return err
				}
				// Find the version entry
				versURL := internal.Server() + "/files/" + url.PathEscape(meta.ID) + "/versions"
				vResp, err := internal.GetJSON(versURL)
				if err != nil {
					return err
				}
				var versions []db.Version
				if err := internal.Decode(vResp, &versions); err != nil {
					return err
				}
				found := false
				for _, v := range versions {
					if v.Version == int64(version) {
						found = true
						// Download the versioned blob by rolling back temporarily is not
						// ideal — instead inform the user and fall through to current if
						// it matches.
						if v.Sha256 == meta.Sha256 {
							break // same content, proceed normally
						}
						return fmt.Errorf("version %d has different content (sha %s); rollback to that version first, then pull", version, internal.ShortSHA(v.Sha256))
					}
				}
				if !found {
					return fmt.Errorf("version %d not found", version)
				}
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("download: %w", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("server %d: %s", resp.StatusCode, body)
			}

			fileName := resp.Header.Get("X-File-Name")
			if fileName == "" {
				fileName = filepath.Base(id)
			}

			dest := fileName
			if out != "" {
				info, err := os.Stat(out)
				if err == nil && info.IsDir() {
					dest = filepath.Join(out, fileName)
				} else {
					dest = out
				}
			}

			f, err := os.Create(dest)
			if err != nil {
				return fmt.Errorf("create %s: %w", dest, err)
			}
			defer f.Close()

			contentLength, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
			if err != nil {
				return fmt.Errorf("invalid content length: %w", err)
			}

			log.Println("Downloading file", "size", contentLength)
			bar := progressbar.DefaultBytes(
				contentLength,
				"Downloading file",
			)
			n, err := io.Copy(io.MultiWriter(f, bar), resp.Body)
			if err != nil {
				return fmt.Errorf("write: %w", err)
			}

			sha := resp.Header.Get("X-File-SHA256")
			ver := resp.Header.Get("X-File-Version")
			fmt.Printf("saved     %s\n", dest)
			fmt.Printf("size      %s\n", internal.FmtSize(n))
			if sha != "" {
				fmt.Printf("sha256    %s\n", internal.ShortSHA(sha))
			}
			if ver != "" {
				v, _ := strconv.ParseInt(ver, 10, 64)
				fmt.Printf("version   v%d\n", v)
			}
			return nil
		},
	}
	c.Flags().StringVar(&out, "out", "", "Output path or directory (default: filename)")
	c.Flags().IntVar(&version, "version", 0, "Download a specific version number")
	return c
}

// getFileMeta fetches metadata for a file by id/path.
func getFileMeta(idOrPath string) (*types.File, error) {
	u := internal.Server() + "/files/" + url.PathEscape(idOrPath) + "?meta=true"
	resp, err := internal.GetJSON(u)
	if err != nil {
		return nil, err
	}
	var f types.File
	if err := internal.Decode(resp, &f); err != nil {
		return nil, err
	}
	return &f, nil
}
