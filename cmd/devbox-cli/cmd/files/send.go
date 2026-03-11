package files

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/types"
)

func SendCmd() *cobra.Command {
	var to, dest string
	var all bool

	c := &cobra.Command{
		Use:   "send <id|path>",
		Short: "Deliver a file to one or more peers via Tailscale",
		Args:  cobra.ExactArgs(1),
		Example: `  devbox-cli files deliver deploy.sh --to myhost
  devbox-cli files deliver deploy.sh --to host1,host2 --dest /opt/scripts
  devbox-cli files deliver deploy.sh --all`,
		RunE: func(c *cobra.Command, args []string) error {
			if !all && to == "" {
				return fmt.Errorf("specify --to <host> or --all")
			}
			f, err := getFileMeta(args[0])
			if err != nil {
				return err
			}

			body := map[string]any{
				"broadcast": all,
				"dest_dir":  dest,
			}
			if to != "" {
				var targets []string
				for _, t := range strings.Split(to, ",") {
					if t = strings.TrimSpace(t); t != "" {
						targets = append(targets, t)
					}
				}
				body["targets"] = targets
			}

			u := internal.Server() + "/files/" + url.PathEscape(f.ID) + "/send"
			resp, err := internal.PostJSON(u, body)
			if err != nil {
				return err
			}
			var result struct {
				Results []types.SendResult `json:"results"`
			}
			if err := internal.Decode(resp, &result); err != nil {
				return err
			}
			printSendResults(f.Path, result.Results)
			return nil
		},
	}
	c.Flags().StringVar(&to, "to", "", "Comma-separated target hostnames")
	c.Flags().StringVar(&dest, "dest", "", "Destination directory on target")
	c.Flags().BoolVar(&all, "all", false, "Deliver to all online peers")
	return c
}

func printSendResults(label string, results []types.SendResult) {
	ok, fail := 0, 0
	for _, r := range results {
		if r.Success {
			ok++
			fmt.Printf("  ok   %s -> %s\n", label, r.Target)
		} else {
			fail++
			fmt.Printf("  fail %s -> %s: %s\n", label, r.Target, r.Error)
		}
	}
	fmt.Printf("\n%d ok, %d failed\n", ok, fail)
}
