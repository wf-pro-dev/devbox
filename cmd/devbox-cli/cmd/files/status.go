package files

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	completion "github.com/wf-pro-dev/devbox/internal/cmd/completion"
)

// StatusResult matches the JSON structure returned by DriftHandler.GetFileStatus
type StatusResult struct {
	Hostname  string `json:"hostname"`
	Status    string `json:"status"`
	LocalPath string `json:"local_path"`
	Error     string `json:"error,omitempty"`
}

func StatusCmd() *cobra.Command {
	var nodes []string

	cmd := &cobra.Command{
		Use:   "status <id|path>",
		Short: "Check file drift across the Tailscale fleet",
		Long: `Compares the vault version of a file against physical files on remote nodes.
It only checks nodes where the directory is explicitly marked as 'share = true' in files.toml.`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completion.FileCompletions,
		Example: `  devbox-cli files status nginx.conf
  devbox-cli files status 550e8400-e29b-41d4-a716 --nodes vps-1,vps-2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Resolve file metadata (using existing helper in files.go)
			f, err := getFileMeta(args[0])
			if err != nil {
				return err
			}

			// 2. Build the request URL for the DriftHandler endpoint
			u := fmt.Sprintf("%s/files/%s/status", internal.Server(), url.PathEscape(f.ID))

			// Add optional node filtering if flags are provided
			if len(nodes) > 0 {
				q := url.Values{}
				nodesString := strings.Join(nodes, ",")
				q.Add("nodes", nodesString)
				u += "?" + q.Encode()
			}

			// 3. Execute request
			resp, err := internal.GetJSON(u)
			if err != nil {
				return err
			}

			var results []StatusResult
			if err := internal.Decode(resp, &results); err != nil {
				return err
			}

			// 4. Output results in a formatted table
			if len(results) == 0 {
				fmt.Println("No audited nodes found for this file path.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "HOSTNAME\tSTATUS\tLOCAL PATH")
			fmt.Fprintln(w, "--------\t------\t----------")

			for _, res := range results {
				status := res.Status
				if res.Error != "" {
					status = fmt.Sprintf("ERROR (%s)", res.Error)
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", res.Hostname, status, res.LocalPath)
			}
			w.Flush()

			return nil
		},
	}

	// Flag for manual node filtering
	cmd.Flags().StringSliceVarP(&nodes, "nodes", "n", nil, "Comma-separated list of hostnames to check")
	_ = cmd.RegisterFlagCompletionFunc("nodes", completion.PeerCompletions)

	return cmd
}
