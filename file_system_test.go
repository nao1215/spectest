package spectest

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_defaultFileSystem_create(t *testing.T) {
	t.Run("should create a file", func(t *testing.T) {
		tempDir := os.TempDir()

		fs := &defaultFileSystem{}
		file, err := fs.create(filepath.Join(tempDir, "test.txt"))
		if err != nil {
			t.Fatalf("create() error = %v, wantErr %v", err, false)
		}
		defer file.Close() // nolint

		if _, err := os.Stat(filepath.Join(tempDir, "test.txt")); os.IsNotExist(err) {
			t.Errorf("create() file does not exist")
		}
	})
}

func Test_defaultFileSystem_mkdirAll(t *testing.T) {
	t.Run("should create a directory", func(t *testing.T) {
		tempDir := os.TempDir()

		fs := &defaultFileSystem{}
		if err := fs.mkdirAll(filepath.Join(tempDir, "test"), 0755); err != nil {
			t.Fatalf("mkdirAll() error = %v, wantErr %v", err, false)
		}

		if _, err := os.Stat(filepath.Join(tempDir, "test")); os.IsNotExist(err) {
			t.Errorf("mkdirAll() directory does not exist")
		}
	})
}

func Test_goldenFile_write(t *testing.T) {
	t.Run("should write data to the golden file", func(t *testing.T) {
		tempDir := os.TempDir()

		g := newGoldenFile(filepath.Join(tempDir, "test.txt"), true, &defaultFileSystem{})
		if err := g.write([]byte("test")); err != nil {
			t.Fatalf("write() error = %v, wantErr %v", err, false)
		}

		if _, err := os.Stat(filepath.Join(tempDir, "test.txt")); os.IsNotExist(err) {
			t.Errorf("write() file does not exist")
		}
	})

	t.Run("should not write data to the golden file", func(t *testing.T) {
		tempDir := os.TempDir()

		g := newGoldenFile(filepath.Join(tempDir, "test.txt"), false, &defaultFileSystem{})
		if err := g.write([]byte("test")); err == nil {
			t.Fatalf("write() error = %v, wantErr %v", err, true)
		}
	})
}
