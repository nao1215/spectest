package spectest

import (
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDiagramBadgeCSSClass(t *testing.T) {
	tests := []struct {
		status int
		class  string
	}{
		{status: http.StatusOK, class: "badge badge-success"},
		{status: http.StatusInternalServerError, class: "badge badge-danger"},
		{status: http.StatusBadRequest, class: "badge badge-warning"},
	}
	for _, test := range tests {
		t.Run(test.class, func(t *testing.T) {
			class := badgeCSSClass(test.status)

			assert.Equal(t, test.class, class)
		})
	}
}

func TestFormatBodyContentShouldReplaceBody(t *testing.T) {
	stream := io.NopCloser(strings.NewReader("lol"))

	val, err := formatBodyContent(stream, func(replacementBody io.ReadCloser) {
		stream = replacementBody
	})
	assert.NoError(t, err)
	assert.Equal(t, "lol", val)

	valSecondRun, errSecondRun := formatBodyContent(stream, func(replacementBody io.ReadCloser) {
		stream = replacementBody
	})
	assert.NoError(t, errSecondRun)
	assert.Equal(t, "lol", valSecondRun)
}

func TestWebSequenceDiagramGeneratesDSL(t *testing.T) {
	t.Run("should generate a valid web sequence diagram", func(t *testing.T) {
		wsd := webSequenceDiagramDSL{
			meta: &Meta{},
		}
		wsd.addRequestRow("A", "B", "request1")
		wsd.addRequestRow("B", "C", "request2")
		wsd.addResponseRow("C", "B", "response1")
		wsd.addResponseRow("B", "A", "response2")

		actual := wsd.toString()

		expected := `"A"->"B": (1) request1
"B"->"C": (2) request2
"C"->>"B": (3) response1
"B"->>"A": (4) response2
`
		if expected != actual {
			t.Fatalf("expected=%s != \nactual=%s", expected, actual)
		}
	})

	t.Run("use custom consumer name and custom testing target name", func(t *testing.T) {
		wsd := webSequenceDiagramDSL{
			meta: &Meta{},
		}
		wsd.addRequestRow(ConsumerDefaultName, SystemUnderTestDefaultName, "request1")
		wsd.addRequestRow(SystemUnderTestDefaultName, "C", "request2")
		wsd.addResponseRow("C", SystemUnderTestDefaultName, "response1")
		wsd.addResponseRow(SystemUnderTestDefaultName, ConsumerDefaultName, "response2")

		actual := wsd.toString()

		expected := `"cli"->"sut": (1) request1
"sut"->"C": (2) request2
"C"->>"sut": (3) response1
"sut"->>"cli": (4) response2
`
		if expected != actual {
			t.Fatalf("expected=%s != \nactual=%s", expected, actual)
		}
	})

	t.Run("use custom consumer name and custom testing target name", func(t *testing.T) {
		wsd := webSequenceDiagramDSL{
			meta: &Meta{
				ConsumerName:      "custom-consumer",
				TestingTargetName: "custom-testing-target",
			},
		}
		wsd.addRequestRow(ConsumerDefaultName, SystemUnderTestDefaultName, "request1")
		wsd.addRequestRow(SystemUnderTestDefaultName, "C", "request2")
		wsd.addResponseRow("C", SystemUnderTestDefaultName, "response1")
		wsd.addResponseRow(SystemUnderTestDefaultName, ConsumerDefaultName, "response2")

		actual := wsd.toString()

		expected := `"custom-consumer"->"custom-testing-target": (1) request1
"custom-testing-target"->"C": (2) request2
"C"->>"custom-testing-target": (3) response1
"custom-testing-target"->>"custom-consumer": (4) response2
`
		if expected != actual {
			t.Fatalf("expected=%s != \nactual=%s", expected, actual)
		}
	})
}

func TestNewSequenceDiagramFormatterSetsDefaultPath(t *testing.T) {
	formatter := SequenceDiagram()

	assert.Equal(t, ".sequence", formatter.storagePath)
}

func TestNewSequenceDiagramFormatterOverridesPath(t *testing.T) {
	formatter := SequenceDiagram(".sequence-diagram")

	assert.Equal(t, ".sequence-diagram", formatter.storagePath)
}

func TestRecorderBuilder(t *testing.T) {
	recorder := aRecorder()

	assert.Equal(t, 4, len(recorder.Events))
	assert.Equal(t, "title", recorder.Title)
	assert.Equal(t, "subTitle", recorder.SubTitle)
	assert.Equal(t,
		&Meta{
			Path:   "/user",
			Name:   "some test",
			Host:   "example.com",
			Method: "GET",
		}, recorder.Meta)
	assert.Equal(t, "reqSource", recorder.Events[0].(HTTPRequest).Source)
	assert.Equal(t, "mesReqSource", recorder.Events[1].(MessageRequest).Source)
	assert.Equal(t, "mesResSource", recorder.Events[2].(MessageResponse).Source)
	assert.Equal(t, "resSource", recorder.Events[3].(HTTPResponse).Source)
}

func TestNewHTMLTemplateModelErrorsIfNoEventsDefined(t *testing.T) {
	recorder := NewTestRecorder()

	_, err := newHTMLTemplateModel(recorder)

	assert.Equal(t, "no events are defined", err.Error())
}

func TestNewHTMLTemplateModelSuccess(t *testing.T) {
	recorder := aRecorder()

	model, err := newHTMLTemplateModel(recorder)

	assert.True(t, err == nil)
	assert.Equal(t, 4, len(model.LogEntries))
	assert.Equal(t, "title", model.Title)
	assert.Equal(t, "subTitle", model.SubTitle)
	assert.Equal(t, template.JS(`{"host":"example.com","method":"GET","name":"some test","path":"/user"}`), model.MetaJSON)
	assert.Equal(t, http.StatusNoContent, model.StatusCode)
	assert.Equal(t, "badge badge-success", model.BadgeClass)
	assert.True(t, strings.Contains(model.WebSequenceDSL, "GET /abcdef"))
}

func aRecorder() *Recorder {
	return NewTestRecorder().
		AddTitle("title").
		AddSubTitle("subTitle").
		AddHTTPRequest(aRequest()).
		AddMessageRequest(MessageRequest{Header: "A", Body: "B", Source: "mesReqSource"}).
		AddMessageResponse(MessageResponse{Header: "C", Body: "D", Source: "mesResSource"}).
		AddHTTPResponse(aResponse()).
		AddMeta(&Meta{
			Path:   "/user",
			Name:   "some test",
			Host:   "example.com",
			Method: "GET",
		})
}

func TestNewHttpRequestLogEntry(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/path", strings.NewReader(`{"a": 12345}`))

	logEntry, err := NewHTTPRequestLogEntry(req)

	assert.True(t, err == nil)
	assert.True(t, strings.Contains(logEntry.Header, "GET /path"))
	assert.True(t, strings.Contains(logEntry.Header, "HTTP/1.1"))
	assert.JSONEq(t, logEntry.Body, `{"a": 12345}`)
}

func TestNewHttpResponseLogEntryJSON(t *testing.T) {
	response := &http.Response{
		ProtoMajor:    1,
		ProtoMinor:    1,
		StatusCode:    http.StatusOK,
		ContentLength: 21,
		Body:          io.NopCloser(strings.NewReader(`{"a": 12345}`)),
	}

	logEntry, err := NewHTTPResponseLogEntry(response)

	assert.True(t, err == nil)
	assert.True(t, strings.Contains(logEntry.Header, `HTTP/1.1 200 OK`))
	assert.True(t, strings.Contains(logEntry.Header, `Content-Length: 21`))
	assert.JSONEq(t, logEntry.Body, `{"a": 12345}`)
}

func TestNewHttpResponseLogEntryPlainText(t *testing.T) {
	response := &http.Response{
		ProtoMajor:    1,
		ProtoMinor:    1,
		StatusCode:    http.StatusOK,
		ContentLength: 21,
		Body:          io.NopCloser(strings.NewReader(`abcdef`)),
	}

	logEntry, err := NewHTTPResponseLogEntry(response)

	assert.True(t, err == nil)
	assert.True(t, strings.Contains(logEntry.Header, `HTTP/1.1 200 OK`))
	assert.True(t, strings.Contains(logEntry.Header, `Content-Length: 21`))
	assert.Equal(t, logEntry.Body, `abcdef`)
}

func aRequest() HTTPRequest {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/abcdef?name=abc", nil)
	req.Header.Set("Content-Type", "application/json")
	return HTTPRequest{Value: req, Source: "reqSource", Target: "reqTarget"}
}

func aResponse() HTTPResponse {
	return HTTPResponse{
		Value: &http.Response{
			StatusCode:    http.StatusNoContent,
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: 0,
		},
		Source: "resSource",
		Target: "resTarget",
	}
}

type FS struct {
	CapturedCreateName   string
	CapturedCreateFile   string
	CapturedMkdirAllPath string
}

func (m *FS) create(name string) (*os.File, error) {
	m.CapturedCreateName = name
	file, err := os.CreateTemp("/tmp", "spectest")
	if err != nil {
		panic(err)
	}
	m.CapturedCreateFile = file.Name()
	return file, nil
}

func (m *FS) mkdirAll(path string, _ os.FileMode) error {
	m.CapturedMkdirAllPath = path
	return nil
}
