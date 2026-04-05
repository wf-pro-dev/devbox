package files

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/internal/version"
	"github.com/wf-pro-dev/devbox/types"
)

func DiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <id|path> [vN] [vM | <local-file>]",
		Short: "Compare versions, local files, or remote node files vs vault",
		Args:  cobra.RangeArgs(1, 3),
		Example: `  devbox-cli files diff deploy.sh              # Vault: current vs previous
  devbox-cli files diff deploy.sh v2 v1        # Vault: v2 vs v1
  devbox-cli files diff deploy.sh ./deploy.sh  # Vault vs Local
  devbox-cli files diff deploy.sh --node vps-1 # Vault vs Remote Node`,
		RunE: func(c *cobra.Command, args []string) error {
			f, err := getFileMeta(args[0])
			if err != nil {
				return err
			}

			nodeName, _ := c.Flags().GetString("node")

			// Case 1: Vault vs. Remote Node (Drift Detection)
			if nodeName != "" {
				version := "latest"
				if len(args) >= 2 {
					version = args[1]
				}
				return diffRemote(f, version, nodeName)
			}

			// Case 2: Vault vs. Local Filesystem
			if len(args) == 2 {
				localPath := args[1]
				if _, err := os.Stat(localPath); err == nil {
					return diffLocal(f, localPath)
				}
			}

			// Case 3: Vault vs. Vault (Version Comparison)
			return diffVaultVersions(f, args)
		},
	}

	cmd.Flags().String("node", "", "Target node for remote drift comparison")
	return cmd
}

// diffLocal handles Vault vs. Local comparison using the new unified diff logic.
func diffLocal(f *types.File, localPath string) error {
	u := internal.Server() + "/files/" + url.PathEscape(f.ID) + "/diff/local"

	// Upload local file to server-side diff engine to get Unified Diff
	resp, err := internal.UploadFileUpdate(u, localPath, "")
	if err != nil {
		return fmt.Errorf("local diff failed: %w", err)
	}

	return printDiffResponse(resp)
}

// diffRemote handles Vault vs. Remote Node comparison (Layer 2 Deep Check).
func diffRemote(f *types.File, v, nodeName string) error {
	u := internal.Server() + "/files/" + url.PathEscape(f.ID) + "/diff/node"
	q := url.Values{}
	q.Set("version", version.StripV(v))
	q.Set("node", nodeName)

	resp, err := internal.GetJSON(u + "?" + q.Encode())
	if err != nil {
		return fmt.Errorf("remote node diff failed: %w", err)
	}

	return printDiffResponse(resp)
}

// diffVaultVersions handles the standard internal version comparison.
func diffVaultVersions(f *types.File, args []string) error {
	u := internal.Server() + "/files/" + url.PathEscape(f.ID) + "/diff"
	q := url.Values{}
	if len(args) >= 2 {
		q.Set("a", version.StripV(args[1]))
	}
	if len(args) == 3 {
		q.Set("b", version.StripV(args[2]))
	}

	resp, err := internal.GetJSON(u + "?" + q.Encode())
	if err != nil {
		return err
	}

	return printDiffResponse(resp)
}

// printDiffResponse handles the common output format for all diff types.
func printDiffResponse(resp *http.Response) error {
	var result struct {
		Identical bool   `json:"identical"`
		Unified   string `json:"unified"`
		LabelA    string `json:"vault_label"`
		LabelB    string `json:"node_label"`
	}

	if err := internal.Decode(resp, &result); err != nil {
		return err
	}

	if result.Identical {
		fmt.Println("No differences found (identical content).")
		return nil
	}

	fmt.Println(result.Unified)
	return nil
}
