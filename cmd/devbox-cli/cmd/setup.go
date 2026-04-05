package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wf-pro-dev/tailkit"
	tailkitTypes "github.com/wf-pro-dev/tailkit/types"
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

	tool := tailkitTypes.Tool{
		Name:      "devbox",
		Version:   VERSION,
		TsnetHost: "devbox",
	}

	fmt.Println("Registering devbox with tailkitd...")
	if err := tailkit.Install(context.Background(), tool); err != nil {
		return err
	}
	fmt.Println("Done. This node is now registered.")
	return nil
}
