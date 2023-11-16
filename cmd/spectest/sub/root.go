// Package sub is spectest sub-commands.
package sub

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Execute run process.
func Execute() int {
	rootCmd := newRootCmd()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err.Error())
		return 1
	}
	return 0
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spectest",
		Short: "spectest is a tool for unit test.",
		Long: `🦁 The spectest command provides utility for unit testing, not only API test.
🦁 It provides features for all developers writing unit tests in Golang.
`,
	}
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newBugReportCmd())
	cmd.AddCommand(newIndexCmd())
	return cmd
}
