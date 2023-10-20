package image

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestEqualFromFile(t *testing.T) {
	t.Parallel()
	t.Run("should return nil if images are equal", func(t *testing.T) {
		t.Parallel()

		file, err := os.Open(filepath.Join("testdata", "expected.jpg"))
		if err != nil {
			t.Fatal(err)
		}

		expected := filepath.Join("testdata", "expected.jpg")
		response := &http.Response{
			Body: file,
		}
		request := &http.Request{}

		fn := EqualFromFile(expected)
		err = fn(response, request)
		if err != nil {
			t.Errorf("EqualFromFile() error = %v, wantErr %v", err, nil)
		}
	})

	t.Run("should return error if images are not equal", func(t *testing.T) {
		t.Parallel()

		file, err := os.Open(filepath.Join("testdata", "blur.jpg"))
		if err != nil {
			t.Fatal(err)
		}

		expected := filepath.Join("testdata", "expected.jpg")
		response := &http.Response{
			Body: file,
		}
		request := &http.Request{}

		fn := EqualFromFile(expected)
		err = fn(response, request)
		if err == nil {
			t.Errorf("EqualFromFile() does not return error")
		}
	})
}
