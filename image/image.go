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
		return cmpImages(expected, 0.1, res)
	}
}

// EqualFromFileWithThreshold verifies that the image file in expect is the same as the image in the response body.
// The threshold is the maximum difference between the images. The value is between 0 and 1. Less more precise.
func EqualFromFileWithThreshold(expected string, threshold float64) func(*http.Response, *http.Request) error {
	return func(res *http.Response, req *http.Request) error {
		return cmpImages(expected, threshold, res)
	}
}

// cmpImages compares the image in the response body with the image in the expect file.
func cmpImages(expected string, threshold float64, res *http.Response) error {
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
	defer res.Body.Close() //nolint:errcheck

	if _, err = tempFile.Write(body); err != nil {
		return err
	}

	got, err := imaging.Open(tempFile.Name())
	if err != nil {
		return err
	}

	result := imgdiff.Diff(want, got, &imgdiff.Options{Threshold: threshold})
	if !result.Equal {
		return fmt.Errorf("image diff pixels count=%d", result.DiffPixelsCount)
	}
	return nil
}
