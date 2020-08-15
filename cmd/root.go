package cmd

import (
	"github.com/spf13/cobra"
)

// Execute initiates this cli tool.
func Execute() error {
	root := &cobra.Command{
		Use:   "vanity",
		Short: "vanity cli tool",
	}

	root.AddCommand(ServeCmd())

	return root.Execute()
}
