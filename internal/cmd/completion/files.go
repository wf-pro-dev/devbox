package completion

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/internal/db"
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

func VersionCompletions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	resp, err := internal.GetJSON(fmt.Sprintf("%s/files/%s/versions", internal.Server(), args[0]))
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var versions []db.Version
	if err := internal.Decode(resp, &versions); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	completions := make([]string, 0, len(versions))
	for _, version := range versions {
		completions = append(completions, fmt.Sprintf("v%d", version.Version))
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func LocalFileCompletions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// List local files
	files, err := os.ReadDir(".")
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	completions := make([]string, 0, len(files))
	for _, file := range files {
		fileName := file.Name()
		if strings.HasPrefix(fileName, toComplete) {
			completions = append(completions, fileName)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
