//go:build !int

package sub

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBugReport(t *testing.T) {
	t.Parallel()

	t.Run("Check bug-report --help", func(t *testing.T) {
		b := bytes.NewBufferString("")

		copyRootCmd := newRootCmd()

		copyRootCmd.SetOut(b)
		copyRootCmd.SetArgs([]string{"bug-report", "--help"})

		if err := copyRootCmd.Execute(); err != nil {
			t.Fatal(err)
		}
		gotBytes, err := io.ReadAll(b)
		if err != nil {
			t.Fatal(err)
		}
		gotBytes = bytes.ReplaceAll(gotBytes, []byte("\r\n"), []byte("\n"))

		wantBytes, err := os.ReadFile(filepath.Join("testdata", "bug_report", "bug_report.txt"))
		if err != nil {
			t.Fatal(err)
		}
		wantBytes = bytes.ReplaceAll(wantBytes, []byte("\r\n"), []byte("\n"))

		if diff := cmp.Diff(strings.TrimSpace(string(gotBytes)), strings.TrimSpace(string(wantBytes))); diff != "" {
			t.Errorf("value is mismatch (-want +got):\n%s", diff)
		}
	})
}

func Test_bugReportWithWriter(t *testing.T) {
	t.Run("Check bug-report", func(t *testing.T) {
		copyRootCmd := newRootCmd()
		copyRootCmd.SetArgs([]string{"bug-report"})

		oldMap := openFuncMap
		defer func() { openFuncMap = oldMap }()
		openFuncMap = map[string]func(string) bool{
			"darwin": func(targetURL string) bool {
				return false
			},
			"windows": func(targetURL string) bool {
				return false
			},
			"linux": func(targetURL string) bool {
				return false
			},
		}

		if err := copyRootCmd.Execute(); err != nil {
			t.Fatal(err)
		}
		// not check output
	})
}
