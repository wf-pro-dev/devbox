package dirs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
)

func DiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <name|id> <local-dir>",
		Short: "Compare a local directory against the server collection",
		Args:  cobra.ExactArgs(2),
		Example: `  devbox-cli dirs diff nginx ./nginx
  devbox-cli dirs diff abcd1234 ./nginx`,
		RunE: func(c *cobra.Command, args []string) error {
			dir, err := getDirectory(args[0])
			if err != nil {
				return err
			}

			// Hash all local files
			localFiles, err := internal.WalkDir(args[1])
			if err != nil {
				return fmt.Errorf("walk %s: %w", args[1], err)
			}

			type localEntry struct {
				Path   string `json:"path"`
				SHA256 string `json:"sha256"`
			}
			var manifest []localEntry
			for _, lf := range localFiles {
				sha, err := hashFile(lf.LocalPath)
				if err != nil {
					internal.Warn("hash %s: %v", lf.LocalPath, err)
					continue
				}
				manifest = append(manifest, localEntry{Path: lf.RelPath, SHA256: sha})
			}

			u := internal.Server() + "/dirs/" + url.PathEscape(dir.Prefix) + "/diff"
			resp, err := internal.PostJSON(u, manifest)
			if err != nil {
				return err
			}

			var result struct {
				Collection string   `json:"collection"`
				Changed    []string `json:"changed"`
				Added      []string `json:"added"`
				Removed    []string `json:"removed"`
			}
			if err := internal.Decode(resp, &result); err != nil {
				return err
			}

			if len(result.Changed)+len(result.Added)+len(result.Removed) == 0 {
				fmt.Printf("%s is up to date\n", dir.Prefix)
				return nil
			}

			for _, p := range result.Added {
				fmt.Printf("  + %s  (new local file)\n", p)
			}
			for _, p := range result.Changed {
				fmt.Printf("  ~ %s  (content differs)\n", p)
			}
			for _, p := range result.Removed {
				fmt.Printf("  - %s  (only on server)\n", p)
			}

			fmt.Printf("\n%d changed, %d to add, %d only on server\n",
				len(result.Changed), len(result.Added), len(result.Removed))

			// Suggest next step
			if len(result.Changed)+len(result.Added) > 0 {
				fmt.Printf("\nrun:  devbox-cli dirs update %s %s\n",
					dir.Prefix, strings.TrimRight(args[1], "/"))
			}
			return nil
		},
	}
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
