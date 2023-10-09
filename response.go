package spectest

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"os"
	"time"
)

// Response is the user defined expected response from the application under test
type Response struct {
	status            int
	body              string
	headers           map[string][]string
	headersPresent    []string
	headersNotPresent []string
	cookies           []*Cookie
	cookiesPresent    []string
	cookiesNotPresent []string
	apiTest           *SpecTest
	assert            []Assert
}

// Body is the expected response body
func (r *Response) Body(b string) *Response {
	r.body = b
	return r
}

// Bodyf is the expected response body that supports a formatter
func (r *Response) Bodyf(format string, args ...interface{}) *Response {
	r.body = fmt.Sprintf(format, args...)
	return r
}

// BodyFromFile reads the given file and uses the content as the expected response body
func (r *Response) BodyFromFile(f string) *Response {
	b, err := os.ReadFile(f)
	if err != nil {
		r.apiTest.t.Fatal(err)
	}
	r.body = string(b)
	return r
}

// Cookies is the expected response cookies
func (r *Response) Cookies(cookies ...*Cookie) *Response {
	r.cookies = append(r.cookies, cookies...)
	return r
}

// Cookie is used to match on an individual cookie name/value pair in the expected response cookies
func (r *Response) Cookie(name, value string) *Response {
	r.cookies = append(r.cookies, NewCookie(name).Value(value))
	return r
}

// CookiePresent is used to assert that a cookie is present in the response,
// regardless of its value
func (r *Response) CookiePresent(cookieName string) *Response {
	r.cookiesPresent = append(r.cookiesPresent, cookieName)
	return r
}

// CookieNotPresent is used to assert that a cookie is not present in the response
func (r *Response) CookieNotPresent(cookieName string) *Response {
	r.cookiesNotPresent = append(r.cookiesNotPresent, cookieName)
	return r
}

// Header is a builder method to set the request headers
func (r *Response) Header(key, value string) *Response {
	normalizedName := textproto.CanonicalMIMEHeaderKey(key)
	r.headers[normalizedName] = append(r.headers[normalizedName], value)
	return r
}

// HeaderPresent is a builder method to set the request headers that should be present in the response
func (r *Response) HeaderPresent(name string) *Response {
	normalizedName := textproto.CanonicalMIMEHeaderKey(name)
	r.headersPresent = append(r.headersPresent, normalizedName)
	return r
}

// HeaderNotPresent is a builder method to set the request headers that should not be present in the response
func (r *Response) HeaderNotPresent(name string) *Response {
	normalizedName := textproto.CanonicalMIMEHeaderKey(name)
	r.headersNotPresent = append(r.headersNotPresent, normalizedName)
	return r
}

// Headers is a builder method to set the request headers
func (r *Response) Headers(headers map[string]string) *Response {
	for name, value := range headers {
		normalizedName := textproto.CanonicalMIMEHeaderKey(name)
		// TODO: BUG ?
		// appendAssign: append result not assigned to the same slice (gocritic)
		r.headers[normalizedName] = append(r.headers[textproto.CanonicalMIMEHeaderKey(normalizedName)], value)
	}
	return r
}

// Status is the expected response http status code
func (r *Response) Status(s int) *Response {
	r.status = s
	return r
}

// Assert allows the consumer to provide a user defined function containing their own
// custom assertions
func (r *Response) Assert(fn func(*http.Response, *http.Request) error) *Response {
	r.assert = append(r.assert, fn)
	return r.apiTest.response
}

// End runs the test returning the result to the caller
func (r *Response) End() Result {
	apiTest := r.apiTest
	defer func() {
		if apiTest.debugEnabled {
			apiTest.finished = time.Now()
			fmt.Printf("Duration: %s\n", apiTest.finished.Sub(apiTest.started))
		}
	}()

	if apiTest.handler == nil && !apiTest.networkingEnabled {
		apiTest.t.Fatal("either define a http.Handler or enable networking")
	}

	apiTest.started = time.Now()
	var res *http.Response
	if apiTest.reporter != nil {
		res = apiTest.report()
	} else {
		res = r.runTest()
	}

	var unmatchedMocks []UnmatchedMock
	for _, m := range r.apiTest.mocks {
		if !m.isUsed {
			unmatchedMocks = append(unmatchedMocks, UnmatchedMock{
				URL: *m.request.url,
			})
			break
		}
	}

	return Result{
		Response:       res,
		unmatchedMocks: unmatchedMocks,
	}
}

func (r *Response) runTest() *http.Response {
	a := r.apiTest
	if len(a.mocks) > 0 {
		a.transport = newTransport(
			a.mocks,
			a.httpClient,
			a.debugEnabled,
			a.mockResponseDelayEnabled,
			a.mocksObservers,
			r.apiTest,
		)
		defer a.transport.Reset()
		a.transport.Hijack()
	}
	res, req := a.doRequest()

	defer func() {
		if len(a.observers) > 0 {
			for _, observe := range a.observers {
				observe(res, req, a)
			}
		}
	}()

	if a.verifier == nil {
		a.verifier = DefaultVerifier{}
	}

	a.assertMocks()
	a.assertResponse(res)
	a.assertHeaders(res)
	a.assertCookies(res)
	a.assertFunc(res, req)

	return copyHTTPResponse(res)
}

func copyHTTPResponse(response *http.Response) *http.Response {
	if response == nil {
		return nil
	}

	var resBodyBytes []byte
	if response.Body != nil {
		resBodyBytes, _ = io.ReadAll(response.Body)
		response.Body = io.NopCloser(bytes.NewBuffer(resBodyBytes))
	}

	resCopy := &http.Response{
		Header:        map[string][]string{},
		StatusCode:    response.StatusCode,
		Status:        response.Status,
		Body:          io.NopCloser(bytes.NewBuffer(resBodyBytes)),
		Proto:         response.Proto,
		ProtoMinor:    response.ProtoMinor,
		ProtoMajor:    response.ProtoMajor,
		ContentLength: response.ContentLength,
	}

	for name, values := range response.Header {
		resCopy.Header[name] = values
	}

	return resCopy
}
