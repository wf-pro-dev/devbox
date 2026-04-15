package completion

import (
	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	tailkitTypes "github.com/wf-pro-dev/tailkit/types"
)

func PeerCompletions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	u := internal.Server() + "/peers"
	resp, err := internal.GetJSON(u)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var peers []tailkitTypes.Peer
	if err := internal.Decode(resp, &peers); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	completions := make([]string, 0, len(peers))
	for _, peer := range peers {
		completions = append(completions, peer.Status.HostName)
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
