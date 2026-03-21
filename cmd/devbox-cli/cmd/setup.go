package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	tailkit "github.com/wf-pro-dev/tailkit"
)

func newSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Register this node with tailkitd",
		Long:  `Registers devbox-cli as a tool with tailkitd on this node. Run once after installation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup()
		},
	}
}

func runSetup() error {
	// exe, err := os.Executable()
	// if err != nil {
	// 	return fmt.Errorf("could not resolve binary path: %w", err)
	// }
	// // bin, err := filepath.EvalSymlinks(exe)
	// // if err != nil {
	// // 	bin = exe
	// // }

	tool := tailkit.Tool{
		Name:      "devbox",
		Version:   version,
		TsnetHost: "devbox",
	}

	fmt.Println("Registering devbox with tailkitd...")
	if err := tailkit.Install(context.Background(), tool); err != nil {
		return err
	}
	fmt.Println("Done. This node is now registered.")
	return nil
}
