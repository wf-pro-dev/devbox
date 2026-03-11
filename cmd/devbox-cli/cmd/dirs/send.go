package dirs

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
		Use:  "sendr <name|id>",
		Args: cobra.ExactArgs(1),
		Example: `  devbox-cli dirs deliver nginx --to myhost
  devbox-cli dirs deliver nginx --to host1,host2 --dest /etc/nginx
  devbox-cli dirs deliver nginx --all`,
		RunE: func(c *cobra.Command, args []string) error {
			if !all && to == "" {
				return fmt.Errorf("specify --to <host> or --all")
			}
			dir, err := getDirectory(args[0])
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

			u := internal.Server() + "/dirs/" + url.PathEscape(dir.Prefix) + "/deliver"
			resp, err := internal.PostJSON(u, body)
			if err != nil {
				return err
			}

			var result struct {
				Prefix  string             `json:"prefix"`
				Results []types.SendResult `json:"results"`
			}
			if err := internal.Decode(resp, &result); err != nil {
				return err
			}

			ok, fail := 0, 0
			for _, r := range result.Results {
				if r.Success {
					ok++
					fmt.Printf("  ok   %s -> %s\n", result.Prefix, r.Target)
				} else {
					fail++
					fmt.Printf("  fail %s -> %s: %s\n", result.Prefix, r.Target, r.Error)
				}
			}
			fmt.Printf("\n%d ok, %d failed\n", ok, fail)
			return nil
		},
	}
	c.Flags().StringVar(&to, "to", "", "Comma-separated target hostnames")
	c.Flags().StringVar(&dest, "dest", "", "Destination directory on target")
	c.Flags().BoolVar(&all, "all", false, "Deliver to all online peers")
	return c
}
