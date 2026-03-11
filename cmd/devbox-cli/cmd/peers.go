package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPeersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "peers",
		Short: "List online Tailscale peers",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO Phase 3: implement
			fmt.Println("peers: not yet implemented")
			return nil
		},
	}
}
