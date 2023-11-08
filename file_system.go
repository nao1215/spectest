package spectest

import (
	"errors"
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

// goldenFile is a file that is used to store the golden data.
type goldenFile struct {
	// path is the path to the golden file
	path string
	// update is a flag that indicates if the golden file should be updated
	update bool
	// fs is the file system used to create the golden file
	fs fileSystem
}

// newGoldenFile creates a new golden file
func newGoldenFile(path string, update bool, fs fileSystem) *goldenFile {
	return &goldenFile{
		path:   path,
		update: update,
		fs:     fs,
	}
}

// read reads the golden file.
func (g *goldenFile) read() ([]byte, error) {
	return os.ReadFile(filepath.Clean(g.path))
}

// write writes the given data to the golden file.
func (g *goldenFile) write(data []byte) error {
	if !g.update {
		return errors.New("golden file update is disabled")
	}

	dir := filepath.Dir(g.path)
	if err := g.fs.mkdirAll(filepath.Clean(dir), os.ModePerm); err != nil {
		return err
	}

	f, err := g.fs.create(g.path)
	if err != nil {
		return err
	}
	defer f.Close() // nolint

	if _, err := f.Write(data); err != nil {
		return err
	}
	return nil
}
