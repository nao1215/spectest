package sub

import (
	"bytes"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"

	ver "github.com/nao1215/spectest/version"
	"github.com/spf13/cobra"
)

// newBugReportCmd return bug-report command.
func newBugReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "bug-report",
		Short:   "Submit a bug report at GitHub",
		Long:    "bug-report opens the default browser to start a bug report which will include useful system information.",
		Example: "   spectest bug-report",
		RunE:    bugReport,
	}
}

var openFuncMap = map[string]func(string) bool{
	"darwin": func(targetURL string) bool {
		return exec.Command("open", targetURL).Start() == nil
	},
	"windows": func(targetURL string) bool {
		return exec.Command("rundll32.exe", "url,OpenURL", targetURL).Start() == nil
	},
	"linux": func(targetURL string) bool {
		return exec.Command("xdg-open", targetURL).Start() == nil
	},
}

// bugReport opens the default browser to start a bug report which will include useful system information.
func bugReport(_ *cobra.Command, _ []string) error {
	openBrowser, ok := openFuncMap[runtime.GOOS]
	if !ok {
		openBrowser = func(s string) bool { return false }
	}
	return bugReportWithWriter(openBrowser)
}

func bugReportWithWriter(openBrowser func(string) bool) error {
	const (
		description = `## Description (About the problem)
A clear description of the bug encountered.

`
		toReproduce = `## Steps to reproduce
Steps to reproduce the bug.

`
		expectedBehavior = `## Expected behavior
Expected behavior.

`
		additionalDetails = `## Additional details**
Any other useful data to share.
`
	)

	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("## spectest version\ntag version=%s\nrevision=%s\n\n", ver.TagVersion, ver.Revision))
	buf.WriteString(description)
	buf.WriteString(toReproduce)
	buf.WriteString(expectedBehavior)
	buf.WriteString(additionalDetails)
	body := buf.String()

	url := "https://github.com/go-spectest/spectest/issues/new?title=[Bug Report] Title&body=" + url.QueryEscape(body)

	if !openBrowser(url) {
		fmt.Print("Please file a new issue at https://github.com/go-spectest/spectest/issues/new using this template:\n\n")
		fmt.Print(body)
	}
	return nil
}
