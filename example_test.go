//go:build linux || darwin

package spectest_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-spectest/spectest"
)

func TestExample(t *testing.T) {
	imageFile, err := os.Open(filepath.Clean(filepath.Join("testdata", "sample.png")))
	if err != nil {
		panic(err)
	}
	defer imageFile.Close() //nolint

	imageInfo, err := imageFile.Stat()
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(imageFile)
	if err != nil {
		panic(err)
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", fmt.Sprint(imageInfo.Size()))

		_, err = io.Copy(w, bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		CustomReportName("markdow_report").
		Report(spectest.SequenceReport(spectest.ReportFormatterConfig{
			Path: "doc",
			Kind: spectest.ReportKindMarkdown,
		})).
		Handler(handler).
		Get("/image").
		Expect(t).
		Body(string(body)).
		Header("Content-Type", "image/png").
		Header("Content-Length", fmt.Sprint(imageInfo.Size())).
		Status(http.StatusOK).
		End()
}
