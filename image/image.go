// Package image provides assertions for image comparison.
package image

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/go-spectest/imaging"
	"github.com/n7olkachev/imgdiff/pkg/imgdiff"
)

// EqualFromFile verifies that the image file in expect is the same as the image in the response body.
func EqualFromFile(expected string) func(*http.Response, *http.Request) error {
	return func(res *http.Response, req *http.Request) error {
		want, err := imaging.Open(expected)
		if err != nil {
			return err
		}

		tempFile, err := os.CreateTemp("", "image")
		if err != nil {
			return err
		}
		defer tempFile.Close() //nolint:errcheck

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		if _, err = tempFile.Write(body); err != nil {
			return err
		}

		got, err := imaging.Open(tempFile.Name())
		if err != nil {
			return err
		}

		result := imgdiff.Diff(want, got, &imgdiff.Options{Threshold: 0.1})
		if !result.Equal {
			return fmt.Errorf("image diff pixels count=%d", result.DiffPixelsCount)
		}
		return nil
	}
}
