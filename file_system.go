package spectest

import (
	"os"
	"path/filepath"
)

// fileSystem interface to abstract file system operations
type fileSystem interface {
	// create creates a file at the given path
	create(name string) (*os.File, error)
	// mkdirAll creates a directory at the given path
	mkdirAll(path string, perm os.FileMode) error
}

// defaultFileSystem is the default implementation of the fileSystem interface
type defaultFileSystem struct{}

// create creates a file at the given path
func (r *defaultFileSystem) create(name string) (*os.File, error) {
	return os.Create(filepath.Clean(name))
}

// mkdirAll creates a directory at the given path
func (r *defaultFileSystem) mkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}
