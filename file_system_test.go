package spectest

import (
	"os"
	"testing"
)

func Test_defaultFileSystem_create(t *testing.T) {
	t.Parallel()

	t.Run("should create a file", func(t *testing.T) {
		t.Parallel()
		if err := os.Chdir(os.TempDir()); err != nil {
			t.Fatal(err)
		}

		fs := &defaultFileSystem{}
		file, err := fs.create("test.txt")
		if err != nil {
			t.Fatalf("create() error = %v, wantErr %v", err, false)
		}
		defer file.Close() // nolint

		if _, err := os.Stat("test.txt"); os.IsNotExist(err) {
			t.Errorf("create() file does not exist")
		}
	})
}

func Test_defaultFileSystem_mkdirAll(t *testing.T) {
	t.Parallel()

	t.Run("should create a directory", func(t *testing.T) {
		t.Parallel()
		if err := os.Chdir(os.TempDir()); err != nil {
			t.Fatal(err)
		}

		fs := &defaultFileSystem{}
		if err := fs.mkdirAll("test", 0755); err != nil {
			t.Fatalf("mkdirAll() error = %v, wantErr %v", err, false)
		}

		if _, err := os.Stat("test"); os.IsNotExist(err) {
			t.Errorf("mkdirAll() directory does not exist")
		}
	})
}
