package files

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/types"
)

func EditCmd() *cobra.Command {
	var desc, lang, path string

	c := &cobra.Command{
		Use:   "edit <id|path>",
		Short: "Edit file metadata (description, language, path)",
		Args:  cobra.ExactArgs(1),
		Example: `  devbox-cli files edit deploy.sh --desc "Production deploy script"
  devbox-cli files edit abcd1234 --lang bash
  devbox-cli files edit old/path.sh --path new/path.sh`,

		RunE: func(c *cobra.Command, args []string) error {

			if !c.Flags().Changed("desc") && !c.Flags().Changed("lang") && !c.Flags().Changed("path") {
				return fmt.Errorf("specify at least one of --desc, --lang, --path")
			}

			// Resolve to real ID first so PATCH hits /files/{uuid}
			f, err := getFileMeta(args[0])
			if err != nil {
				return err
			}

			body := map[string]string{}
			if c.Flags().Changed("desc") {
				body["description"] = desc
			}
			if c.Flags().Changed("lang") {
				body["language"] = lang
			}
			if c.Flags().Changed("path") {
				body["path"] = path
			}

			u := internal.Server() + "/files/" + url.PathEscape(f.ID)
			resp, err := internal.PatchJSON(u, body)
			if err != nil {
				return err
			}
			var updated types.File
			if err := internal.Decode(resp, &updated); err != nil {
				return err
			}
			fmt.Printf("updated   %s\n", updated.Path)
			fmt.Printf("language  %s\n", updated.Language)
			fmt.Printf("desc      %s\n", nonEmpty(updated.Description))
			return nil
		},
	}
	c.Flags().StringVar(&desc, "desc", "", "New description")
	c.Flags().StringVar(&lang, "lang", "", "New language")
	c.Flags().StringVar(&path, "path", "", "New path (rename/move)")
	return c
}
