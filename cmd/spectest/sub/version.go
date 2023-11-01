package sub

import (
	"fmt"

	ver "github.com/go-spectest/spectest/version"
	"github.com/spf13/cobra"
)

// newVersionCmd return version command.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show " + ver.CommandName + " command version information",
		Run:   version,
	}
}

// version return spectest command version.
func version(_ *cobra.Command, _ []string) {
	fmt.Printf("%s version %s, revision %s (under MIT LICENSE)\n", ver.CommandName, ver.TagVersion, ver.Revision)
}
