package files

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/types"
)

func DiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <id|path> [vN] [vM | <local-file>]",
		Short: "Compare versions or local file vs stored",
		Args:  cobra.RangeArgs(1, 3),
		Example: `  devbox-cli files diff deploy.sh              # current vs previous version
  devbox-cli files diff deploy.sh v2 v1        # v2 vs v1
  devbox-cli files diff deploy.sh ./deploy.sh  # local file vs stored`,
		RunE: func(c *cobra.Command, args []string) error {
			f, err := getFileMeta(args[0])
			if err != nil {
				return err
			}

			// Case: compare local file vs stored
			if len(args) == 2 {
				localPath := args[1]
				if _, err := os.Stat(localPath); err == nil {
					return diffLocal(f, localPath)
				}
			}

			// Case: version comparison via server diff endpoint
			u := internal.Server() + "/files/" + url.PathEscape(f.ID) + "/diff"
			q := url.Values{}
			if len(args) >= 2 {
				q.Set("a", stripV(args[1]))
			}
			if len(args) == 3 {
				q.Set("b", stripV(args[2]))
			}
			if len(q) > 0 {
				u += "?" + q.Encode()
			}

			resp, err := internal.GetJSON(u)
			if err != nil {
				return err
			}
			var result struct {
				File    map[string]string `json:"file"`
				A       db.Version        `json:"a"`
				B       db.Version        `json:"b"`
				Changed struct {
					SHA256    bool  `json:"sha256"`
					SizeDelta int64 `json:"size_delta"`
				} `json:"changed"`
			}
			if err := internal.Decode(resp, &result); err != nil {
				return err
			}

			fmt.Printf("file    %s\n", result.File["path"])
			fmt.Printf("a       v%d  %s  %s\n", result.A.Version, internal.ShortSHA(result.A.Sha256), internal.FmtDate(result.A.CreatedAt))
			fmt.Printf("b       v%d  %s  %s\n", result.B.Version, internal.ShortSHA(result.B.Sha256), internal.FmtDate(result.B.CreatedAt))
			if result.Changed.SHA256 {
				delta := result.Changed.SizeDelta
				sign := "+"
				if delta < 0 {
					sign = ""
				}
				fmt.Printf("changed yes  (%s%s bytes)\n", sign, internal.FmtSize(delta))
			} else {
				fmt.Println("changed no   (identical content)")
			}
			return nil
		},
	}
}

func diffLocal(f *types.File, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", localPath, err)
	}
	defer file.Close()

	h := sha256.New()
	size, err := io.Copy(h, file)
	if err != nil {
		return err
	}
	localSHA := hex.EncodeToString(h.Sum(nil))

	fmt.Printf("file    %s\n", f.Path)
	fmt.Printf("server  v%d  %s  %s\n", f.Version, internal.ShortSHA(f.Sha256), internal.FmtSize(f.Size))
	fmt.Printf("local   %s  %s  (%s)\n", internal.ShortSHA(localSHA), internal.FmtSize(size), localPath)
	if localSHA == f.Sha256 {
		fmt.Println("changed no   (identical)")
	} else {
		delta := size - f.Size
		sign := "+"
		if delta < 0 {
			sign = ""
		}
		fmt.Printf("changed yes  (%s%s)\n", sign, internal.FmtSize(delta))
	}
	return nil
}

// stripV removes a leading "v" from a version string like "v3" -> "3".
func stripV(s string) string {
	if len(s) > 0 && (s[0] == 'v' || s[0] == 'V') {
		return s[1:]
	}
	return s
}
