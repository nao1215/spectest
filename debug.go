package spectest

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
)

// debug is used to enable/disable debug logging
type debug struct {
	enabled bool
}

// newDebug creates a new debug setting
func newDebug() *debug {
	return &debug{}
}

// enable will enable debug logging
func (d *debug) enable() {
	d.enabled = true
}

// isEnable returns true if debug logging is enabled
func (d *debug) isEnable() bool {
	return d.enabled
}

// dumpResponse is used to dump the response.
// If debug logging is disabled, this method will do nothing.
func (d *debug) dumpRequest(req *http.Request) {
	if !d.isEnable() {
		return
	}
	requestDump, err := httputil.DumpRequest(req, true)
	if err == nil {
		debugLog(requestDebugPrefix(), "inbound http request", string(requestDump))
	}
	// TODO: handle error
}

// dumpResponse is used to dump the response.
// If debug logging is disabled, this method will do nothing.
func (d *debug) dumpResponse(res *http.Response) {
	if !d.isEnable() {
		return
	}
	responseDump, err := httputil.DumpResponse(res, true)
	if err == nil {
		debugLog(responseDebugPrefix(), "final response", string(responseDump))
	}
	// TODO: handle error
}

// duration is used to print the duration of the test.
// If debug logging is disabled, this method will do nothing.
func (d *debug) duration(interval *Interval) {
	if !d.isEnable() {
		return
	}
	fmt.Printf("Duration: %s\n", interval.Duration())
}

// mock is used to print the request and response from the mock.
func (d *debug) mock(res *http.Response, req *http.Request) {
	if !d.isEnable() {
		return
	}

	requestDump, err := httputil.DumpRequestOut(req, true)
	if err == nil {
		debugLog(requestDebugPrefix(), "request to mock", string(requestDump))
	}

	if res != nil {
		responseDump, err := httputil.DumpResponse(res, true)
		if err == nil {
			debugLog(responseDebugPrefix(), "response from mock", string(responseDump))
		}
	} else {
		debugLog(responseDebugPrefix(), "response from mock", "")
	}
}

// debugLog is used to print debug information
func debugLog(prefix, header, msg string) {
	fmt.Printf("\n%s %s\n%s\n", prefix, header, msg)
}

// requestDebugLog is used to print debug information for the request
func requestDebugPrefix() string {
	return fmt.Sprintf("%s>", strings.Repeat("-", 10))
}

// responseDebugLog is used to print debug information for the response
func responseDebugPrefix() string {
	return fmt.Sprintf("<%s", strings.Repeat("-", 10))
}
