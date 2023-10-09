package spectest

import (
	"encoding/json"
	"io"
	"net/http"
)

// Result provides the final result
type Result struct {
	Response       *http.Response
	unmatchedMocks []UnmatchedMock
}

// UnmatchedMocks returns any mocks that were not used, e.g. there was not a matching http Request for the mock
func (r Result) UnmatchedMocks() []UnmatchedMock {
	return r.unmatchedMocks
}

// JSON unmarshal the result response body to a valid struct
func (r Result) JSON(t interface{}) {
	data, err := io.ReadAll(r.Response.Body)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, t)
	if err != nil {
		panic(err)
	}
}
