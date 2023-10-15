package spectest

import (
	"net/http"
	"time"
)

// capture is used to capture the inbound request and final response.
type capture struct {
	// inboundRequest is the inbound http request
	inboundRequest *http.Request
	// finalResponse is the final http response
	finalResponse *http.Response
	// mockInteractions is the list of mock interactions
	mockInteractions []*mockInteraction
}

// newCapture creates a new capture
func newCapture() *capture {
	return &capture{
		mockInteractions: []*mockInteraction{},
	}
}

// appendObserver appends the observer to the list of observers
func (c *capture) appendObserver(observers []Observe) []Observe {
	return append(observers, func(finalRes *http.Response, inboundReq *http.Request, a *SpecTest) {
		c.finalResponse = copyHTTPResponse(finalRes)
		defer func() {
			if err := c.finalResponse.Body.Close(); err != nil {
				panic(err) // FIXME: handle error
			}
		}()
		c.inboundRequest = copyHTTPRequest(inboundReq)
	})
}

// appendMockObservers appends the mock observer to the list of observers
func (c *capture) appendMockObservers(mocksObservers []Observe) []Observe {
	return append(mocksObservers, func(mockRes *http.Response, mockReq *http.Request, a *SpecTest) {
		c.mockInteractions = append(c.mockInteractions, &mockInteraction{
			request:   copyHTTPRequest(mockReq),
			response:  copyHTTPResponse(mockRes),
			timestamp: time.Now().UTC(),
		})
	})
}
