package completion

import (
	"fmt"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/types"
)

func FileCompletions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	resp, err := internal.GetJSON(internal.Server() + "/files")
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var files []types.File
	if err := internal.Decode(resp, &files); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	completions := make([]string, 0, len(files))
	for _, file := range files {
		completions = append(completions, fmt.Sprintf("%s\t%s", file.FileName, internal.ShortID(file.ID)))
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
