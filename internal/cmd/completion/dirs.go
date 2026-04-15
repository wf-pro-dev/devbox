package completion

import (
	"net/url"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/types"
)

func DirCompletions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {

	u := internal.Server() + "/dirs"
	u += "?prefix=" + url.QueryEscape(toComplete)

	resp, err := internal.GetJSON(u)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var dirs []types.Directory[db.File]
	if err := internal.Decode(resp, &dirs); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	completions := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		completions = append(completions, dir.Prefix)
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
