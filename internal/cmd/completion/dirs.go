package completion

import (
	"strings"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/types"
)

func DirCompletions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	resp, err := internal.GetJSON(internal.Server() + "/files")
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var files []types.File
	if err := internal.Decode(resp, &files); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	seen := make(map[string]struct{})
	var completions []string

	for _, f := range files {
		if !strings.HasPrefix(f.Path, toComplete) {
			continue
		}

		// Get the portion after what the user typed
		rest := f.Path[len(toComplete):]

		// Find the next directory separator
		idx := strings.Index(rest, "/")
		if idx == -1 {
			// No slash remaining — it's a file, skip it
			continue
		}

		// MUST include toComplete so the shell can match and filter it properly.
		// Bonus: This also gracefully adds the missing slash if the user typed "monitoring/compose"
		next := toComplete + rest[:idx+1]

		if _, already := seen[next]; !already {
			seen[next] = struct{}{}
			completions = append(completions, next)
		}
	}

	return completions, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}
