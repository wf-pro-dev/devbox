package files

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	completion "github.com/wf-pro-dev/devbox/internal/cmd/completion"
	"github.com/wf-pro-dev/devbox/types"
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

// getFileMetaFull fetches full metadata (used by other commands in this package).
func getFileMetaFull(idOrPath string) (*types.File, error) {
	u := internal.Server() + "/files/" + url.PathEscape(idOrPath) + "?meta=true"
	resp, err := internal.GetJSON(u)
	if err != nil {
		return nil, err
	}
	var f types.File
	if err := internal.Decode(resp, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func nonEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
