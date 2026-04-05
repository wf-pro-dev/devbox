package dirs

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	tailkitTypes "github.com/wf-pro-dev/tailkit/types"
)

// key is the dest manchine value is the results
type SendDirResult map[string][]tailkitTypes.SendResult

func SendCmd() *cobra.Command {
	var to, dest string
	var all bool

	c := &cobra.Command{
		Use:  "send <name|id>",
		Args: cobra.ExactArgs(1),
		Example: `  devbox-cli dirs send nginx --to myhost
  devbox-cli dirs send nginx --to host1,host2 --dest /etc/nginx
  devbox-cli dirs send nginx --all`,
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

			fmt.Printf("\nSending %s to %d machines :\n", dir.Prefix, len(to))
			u := internal.Server() + "/dirs/" + url.PathEscape(dir.Prefix) + "/send"
			resp, err := internal.PostJSON(u, body)
			if err != nil {
				return err
			}

			var allResults SendDirResult
			if err := internal.Decode(resp, &allResults); err != nil {
				return err
			}

			printResults(allResults)
			return nil
		},
	}
	c.Flags().StringVar(&to, "to", "", "Comma-separated target hostnames")
	c.Flags().StringVar(&dest, "dest", "", "Destination directory on target")
	c.Flags().BoolVar(&all, "all", false, "Deliver to all online peers")
	return c
}

func printResults(allResults SendDirResult) {
	ok, fail := 0, 0
	fmt.Println()
	for destMachine, results := range allResults {
		success := true
		machineFailed := 0
		for _, result := range results {
			if !result.Success {
				success = false
				machineFailed++
			}
		}
		if success {
			fmt.Printf("    - %s: ok \n", destMachine)
			ok++
		} else {
			fmt.Printf("    - %s: fail, %d files failed \n", destMachine, machineFailed)
			fail++
		}
	}
	fmt.Printf("\n%d ok, %d failed\n", ok, fail)
}
