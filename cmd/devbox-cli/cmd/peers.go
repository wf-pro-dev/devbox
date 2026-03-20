package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/types"
)

func newPeersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "peers",
		Short: "List online Tailscale peers",
		RunE: func(cmd *cobra.Command, args []string) error {
			u := internal.Server() + "/peers"
			resp, err := internal.GetJSON(u)
			if err != nil {
				return err
			}
			var peers []types.Peer
			if err := internal.Decode(resp, &peers); err != nil {
				return err
			}
			for _, peer := range peers {
				fmt.Printf("%s\t%s\t%s\t%t\n", peer.Hostname, peer.DNSName, peer.IP, peer.Online)
			}
			return nil
		},
	}
}
