package files

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/tailkit"
)

func SendCmd() *cobra.Command {
	var to, dest string
	var all bool

	c := &cobra.Command{
		Use:   "send <id|filename>",
		Short: "Deliver a file to one or more peers via Tailscale",
		Args:  cobra.ExactArgs(1),
		Example: `  devbox-cli files send deploy.sh --to myhost
  devbox-cli files send deploy.sh --to host1,host2 --dest /opt/scripts
  devbox-cli files send deploy.sh --all`,
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

			fmt.Printf("\nSending %s (%s) to %d machines :\n", f.FileName, internal.ShortID(f.ID), len(to))
			u := internal.Server() + "/files/" + url.PathEscape(f.ID) + "/send"
			resp, err := internal.PostJSON(u, body)
			if err != nil {
				return err
			}
			var results []tailkit.SendResult
			if err := internal.Decode(resp, &results); err != nil {
				return err
			}
			printResults(results)
			return nil
		},
	}
	c.Flags().StringVar(&to, "to", "", "Comma-separated target hostnames")
	c.Flags().StringVar(&dest, "dest", "", "Destination directory on target")
	c.Flags().BoolVar(&all, "all", false, "Deliver to all online peers")
	return c
}

func printResults(results []tailkit.SendResult) {
	ok, fail := 0, 0
	fmt.Println()
	for _, result := range results {

		if result.Success {
			fmt.Printf("    - %s: ok \n", result.DestMachine)
			ok++
		} else {
			fmt.Printf("    - %s: fail, %s \n", result.DestMachine, result.Error)
			fail++
		}

	}
	fmt.Printf("\n%d ok, %d failed\n", ok, fail)
}
