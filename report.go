package spectest

import (
	"errors"
	"net/http"
	"time"
)

type (
	// ReportFormatter represents the report formatter
	ReportFormatter interface {
		// Format formats the events received from the recorder
		Format(*Recorder)
	}

	// Event represents a reporting event
	Event interface {
		GetTime() time.Time
	}

	// Recorder represents all of the report data
	Recorder struct {
		// Title is the title of the report
		Title string
		// SubTitle is the subtitle of the report
		SubTitle string
		// Meta is the meta data of the report.
		Meta *Meta
		// Events is the list of events that occurred during the test
		Events []Event
	}

	// MessageRequest represents a request interaction
	MessageRequest struct {
		Source    string
		Target    string
		Header    string
		Body      string
		Timestamp time.Time
	}

	// MessageResponse represents a response interaction
	MessageResponse struct {
		Source    string
		Target    string
		Header    string
		Body      string
		Timestamp time.Time
	}

	// HTTPRequest represents an http request
	HTTPRequest struct {
		Source    string
		Target    string
		Value     *http.Request
		Timestamp time.Time
	}

	// HTTPResponse represents an http response
	HTTPResponse struct {
		Source    string
		Target    string
		Value     *http.Response
		Timestamp time.Time
	}
)

// GetTime gets the time of the HttpRequest interaction
func (r HTTPRequest) GetTime() time.Time { return r.Timestamp }

// GetTime gets the time of the HttpResponse interaction
func (r HTTPResponse) GetTime() time.Time { return r.Timestamp }

// GetTime gets the time of the MessageRequest interaction
func (r MessageRequest) GetTime() time.Time { return r.Timestamp }

// GetTime gets the time of the MessageResponse interaction
func (r MessageResponse) GetTime() time.Time { return r.Timestamp }

// NewTestRecorder creates a new TestRecorder
func NewTestRecorder() *Recorder {
	return &Recorder{}
}

// AddHTTPRequest add an http request to recorder
func (r *Recorder) AddHTTPRequest(req HTTPRequest) *Recorder {
	r.Events = append(r.Events, req)
	return r
}

// AddHTTPResponse add an HTTPResponse to the recorder
func (r *Recorder) AddHTTPResponse(req HTTPResponse) *Recorder {
	r.Events = append(r.Events, req)
	return r
}

// AddMessageRequest add a MessageRequest to the recorder
func (r *Recorder) AddMessageRequest(m MessageRequest) *Recorder {
	r.Events = append(r.Events, m)
	return r
}

// AddMessageResponse add a MessageResponse to the recorder
func (r *Recorder) AddMessageResponse(m MessageResponse) *Recorder {
	r.Events = append(r.Events, m)
	return r
}

// AddTitle add a Title to the recorder
func (r *Recorder) AddTitle(title string) *Recorder {
	r.Title = title
	return r
}

// AddSubTitle add a SubTitle to the recorder
func (r *Recorder) AddSubTitle(subTitle string) *Recorder {
	r.SubTitle = subTitle
	return r
}

// AddMeta add Meta to the recorder
func (r *Recorder) AddMeta(meta *Meta) *Recorder {
	r.Meta = meta
	return r
}

// ResponseStatus get response status of the recorder, returning an error when this wasn't possible
func (r *Recorder) ResponseStatus() (int, error) {
	if len(r.Events) == 0 {
		return -1, errors.New("no events are defined")
	}

	switch v := r.Events[len(r.Events)-1].(type) {
	case HTTPResponse:
		return v.Value.StatusCode, nil
	case MessageResponse:
		return -1, nil
	default:
		return -1, errors.New("final event should be a response type")
	}
}

// Reset resets the recorder to default starting state
func (r *Recorder) Reset() {
	r.Title = ""
	r.SubTitle = ""
	r.Events = nil
	r.Meta = nil
}
