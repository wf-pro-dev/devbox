package files

import (
	"fmt"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	completion "github.com/wf-pro-dev/devbox/internal/cmd/completion"
)

func InfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "info <id|path>",
		Short:             "Show file metadata",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completion.FileCompletions,
		Example: `  devbox-cli files info deploy.sh
  devbox-cli files info abcd1234`,
		RunE: func(c *cobra.Command, args []string) error {
			f, err := getFileMeta(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("id          %s\n", f.ID)
			fmt.Printf("name        %s\n", f.FileName)
			fmt.Printf("path        %s\n", f.Path)
			fmt.Printf("version     v%d\n", f.Version)
			fmt.Printf("size        %s\n", internal.FmtSize(f.Size))
			fmt.Printf("sha256      %s\n", f.Sha256)
			fmt.Printf("language    %s\n", f.Language)
			fmt.Printf("tags        %s\n", internal.FmtTags(f.Tags))
			fmt.Printf("description %s\n", nonEmpty(f.Description))
			fmt.Printf("uploaded_by %s\n", f.UploadedBy)
			fmt.Printf("created     %s\n", internal.FmtDate(f.CreatedAt))
			fmt.Printf("updated     %s\n", internal.FmtDate(f.UpdatedAt))
			return nil
		},
	}
}

func nonEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
