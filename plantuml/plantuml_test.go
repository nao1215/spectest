package plantuml

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-spectest/spectest"
)

func TestWritesTheMeta(t *testing.T) {
	recorder := aRecorder()
	capture := &writer{}

	NewFormatter(capture).Format(recorder)

	actual := bytes.NewReader([]byte(capture.captured))
	firstLine, _, err := bufio.NewReader(actual).ReadLine()
	if err != nil {
		panic(err)
	}

	if `{"host":"example.com","method":"GET","name":"some test","path":"/user"}` != string(firstLine) {
		t.Fail()
	}
}

func TestNewFormatter(t *testing.T) {
	recorder := aRecorder()
	capture := &writer{}

	NewFormatter(capture).Format(recorder)

	actual := capture.captured
	expected, _ := os.ReadFile("testdata/snapshot.txt")

	if normalize(string(expected)) != normalize(actual) {
		fmt.Printf("Expected '%s'\nReceived '%s'\n", string(expected), actual)
		t.Fail()
	}
}

type writer struct {
	captured string
}

func (p *writer) Write(data []byte) (int, error) {
	p.captured = strings.TrimSpace(string(data))
	return -1, nil
}

func normalize(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func aRecorder() *spectest.Recorder {
	return spectest.NewTestRecorder().
		AddTitle("title").
		AddSubTitle("subTitle").
		AddHTTPRequest(aRequest()).
		AddMessageRequest(spectest.MessageRequest{Header: "SQL Query", Body: "SELECT * FROM users", Source: "sut-a", Target: "a"}).
		AddMessageResponse(spectest.MessageResponse{Header: "SQL Result", Body: "Rows count: 122", Source: "a", Target: "sut-a"}).
		AddHTTPResponse(aResponse()).
		AddMeta(&spectest.Meta{
			Path:   "/user",
			Name:   "some test",
			Host:   "example.com",
			Method: http.MethodGet,
		})
}

func aRequest() spectest.HTTPRequest {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/abcdef", nil)
	req.Header.Set("Content-Type", "application/json")
	return spectest.HTTPRequest{Value: req, Source: "cli", Target: "sut-a"}
}

func aResponse() spectest.HTTPResponse {
	return spectest.HTTPResponse{
		Value: &http.Response{
			StatusCode:    http.StatusOK,
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: 0,
		},
		Source: "sut-a",
		Target: "cli",
	}
}
