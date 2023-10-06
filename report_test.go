package spectest

import (
	"net/http"
	"testing"
)

func TestRecorderResponseStatusRecordsFinalResponseStatus(t *testing.T) {
	status, err := NewTestRecorder().
		AddHTTPRequest(HTTPRequest{}).
		AddHTTPResponse(HTTPResponse{Value: &http.Response{StatusCode: http.StatusAccepted}}).
		AddHTTPRequest(HTTPRequest{}).
		AddHTTPResponse(HTTPResponse{Value: &http.Response{StatusCode: http.StatusBadRequest}}).
		ResponseStatus()

	assert.Equal(t, true, err == nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestRecorderResponseStatusErrorsIfNoEventsDefined(t *testing.T) {
	_, err := NewTestRecorder().
		ResponseStatus()

	assert.Equal(t, "no events are defined", err.Error())
}

func TestRecorderResponseStatusErrorsIfFinalEventNotAResponse(t *testing.T) {
	_, err := NewTestRecorder().
		AddHTTPRequest(HTTPRequest{}).
		ResponseStatus()

	assert.Equal(t, "final event should be a response type", err.Error())
}

func TestRecorderResponseStatusHandlesEventTypes(t *testing.T) {
	rec := NewTestRecorder().
		AddMessageRequest(MessageRequest{}).
		AddMessageResponse(MessageResponse{})

	status, _ := rec.ResponseStatus()
	assert.Equal(t, -1, status)
	assert.Equal(t, 2, len(rec.Events))
}

func TestRecorderAddsTitle(t *testing.T) {
	rec := NewTestRecorder().
		AddTitle("title")

	assert.Equal(t, rec.Title, "title")
}

func TestRecorderAddsSubTitle(t *testing.T) {
	rec := NewTestRecorder().
		AddSubTitle("subTitle")

	assert.Equal(t, rec.SubTitle, "subTitle")
}

func TestRecorderReset(t *testing.T) {
	rec := NewTestRecorder().
		AddTitle("title").
		AddSubTitle("subTitle").
		AddMeta(newMeta()).
		AddMessageRequest(MessageRequest{})

	rec.Reset()
	assert.Equal(t, &Recorder{}, rec)
}
