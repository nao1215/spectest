package spectest

import (
	"io"
	"net/http"
	"net/http/httputil"
	"time"
)

// LogEntry represents a single log entry that is used to generate the web sequence diagram.
// It contains the header, body and timestamp of the log entry.
type LogEntry struct {
	// Header is the header of the log entry.
	// e.g.  "GET /path HTTP/1.1\r\nHost: example.com\r\n\r\n",
	Header string
	// Body is the body of the log entry.
	Body string
	// Timestamp is the timestamp of the log entry. It does not set when the log entry is created.
	Timestamp time.Time
}

// NewHTTPRequestLogEntry creates a new LogEntry from a http.Request.
func NewHTTPRequestLogEntry(req *http.Request) (LogEntry, error) {
	reqHeader, err := httputil.DumpRequest(req, false)
	if err != nil {
		return LogEntry{}, err
	}
	body, err := formatBodyContent(req.Body, func(replacementBody io.ReadCloser) {
		req.Body = replacementBody
	})
	if err != nil {
		return LogEntry{}, err
	}
	return LogEntry{Header: string(reqHeader), Body: body}, err
}

// NewHTTPResponseLogEntry creates a new LogEntry from a http.Response.
func NewHTTPResponseLogEntry(res *http.Response) (LogEntry, error) {
	resDump, err := httputil.DumpResponse(res, false)
	if err != nil {
		return LogEntry{}, err
	}
	body, err := formatBodyContent(res.Body, func(replacementBody io.ReadCloser) {
		res.Body = replacementBody
	})
	if err != nil {
		return LogEntry{}, err
	}
	return LogEntry{Header: string(resDump), Body: body}, err
}
