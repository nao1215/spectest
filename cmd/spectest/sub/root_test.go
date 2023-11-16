// Package sub is spectest sub-commands.
package sub

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_Execute(t *testing.T) {
	t.Run("generate index", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skip test on Windows. we test feature that generate index file in markdown package")
		}

		os.Args = []string{"spectest", "index", "testdata", "-t", "SPECTEST_TEST", "-d", "This file is used by unit test"}
		want, err := os.ReadFile(filepath.Join("expected", "index.md"))
		if err != nil {
			t.Fatal(err)
		}

		exitCode := Execute()
		if exitCode != 0 {
			t.Errorf("Execute() = %v, want %v", exitCode, 0)
		}
		got, err := os.ReadFile(filepath.Join("testdata", "index.md"))
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(string(got), string(want)); diff != "" {
			t.Errorf("Execute() mismatch (-want +got):\n%s", diff)
		}
	})
}
