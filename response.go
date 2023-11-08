package spectest

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"

	"github.com/nao1215/gorky/file"
)

// Response is the user defined expected response from the application under test
type Response struct {
	specTest          *SpecTest
	status            int
	body              string
	headers           map[string][]string
	headersPresent    []string
	headersNotPresent []string
	cookies           []*Cookie
	cookiesPresent    []string
	cookiesNotPresent []string
	assert            []Assert
	goldenFile        *goldenFile
}

func newResponse(s *SpecTest) *Response {
	return &Response{
		specTest:          s,
		headers:           map[string][]string{},
		headersPresent:    []string{},
		headersNotPresent: []string{},
		cookies:           []*Cookie{},
		cookiesPresent:    []string{},
		cookiesNotPresent: []string{},
		goldenFile:        &goldenFile{},
	}
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
func (r *Response) BodyFromFile(path string) *Response {
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		r.specTest.t.Fatal(err)
	}
	r.body = string(b)
	return r
}

// BodyFronGoldenFile reads the given file and uses the content as the expected response body.
// If the update flag is set, the golden file will be updated with the actual response body.
// Example: go test -update
func (r *Response) BodyFronGoldenFile(path string) *Response {
	update := false
	flag.BoolVar(&update, "update", false, "update golden files")

	r.goldenFile = newGoldenFile(path, update, &defaultFileSystem{})
	if !r.goldenFile.update {
		if !file.IsFile(path) {
			r.goldenFile.update = true // create a new golden file
		} else {
			b, err := r.goldenFile.read()
			if err != nil {
				r.specTest.t.Fatal(err)
			}
			r.body = string(b)
		}
	}
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
	return r.specTest.response
}

// End runs the test returning the result to the caller
func (r *Response) End() Result {
	defer func() {
		r.specTest.debug.duration(r.specTest.interval)
	}()
	r.specTest.assertValidHandlerOrNetwork()

	return Result{
		Response:       r.runTestAndGenerateReportIfNeeded(),
		unmatchedMocks: r.specTest.mocks.findUnmatchedMocks(),
	}
}

// runTestWithReportIfNeeded runs the test and returns the response.
// If the reporter is set, it will return the report response.
func (r *Response) runTestAndGenerateReportIfNeeded() *http.Response {
	if r.specTest.reporter != nil {
		return r.specTest.report()
	}
	return r.runTest()
}

// runTest runs the test. This method is not thread safe.
func (r *Response) runTest() *http.Response {
	specTest := r.specTest
	specTest.interval.Start()
	defer specTest.interval.End()

	if specTest.mocks.len() > 0 {
		specTest.transport = newTransport(
			specTest.mocks,
			specTest.httpClient,
			specTest.debug,
			specTest.mockResponseDelayEnabled,
			specTest.mocksObservers,
			r.specTest,
		)
		defer specTest.transport.Reset()
		specTest.transport.Hijack()
	}
	res, req := specTest.doRequest()

	defer func() {
		if len(specTest.observers) > 0 {
			for _, observe := range specTest.observers {
				observe(res, req, specTest)
			}
		}
	}()

	if err := r.updateGoldenFileIfNeeded(res); err != nil {
		r.specTest.t.Fatal(err)
	}
	specTest.assertAll(res, req)
	return copyHTTPResponse(res)
}

// updateGoldenFileIfNeeded updates the golden file if needed.
func (r *Response) updateGoldenFileIfNeeded(res *http.Response) error {
	if !r.goldenFile.update || res == nil || res.Body == nil {
		return nil
	}
	copyRes := copyHTTPResponse(res)
	body, err := io.ReadAll(copyRes.Body)
	if err != nil {
		return err
	}
	return r.goldenFile.write(body)
}

// assertAll runs all the assertions.
func (s *SpecTest) assertAll(res *http.Response, req *http.Request) {
	if s.verifier == nil {
		s.verifier = DefaultVerifier{}
	}
	s.assertMocks()
	s.assertResponse(res)
	s.assertHeaders(res)
	s.assertCookies(res)
	s.assertFunc(res, req)
}

// copyHTTPResponse copies the given http.Response
func copyHTTPResponse(response *http.Response) *http.Response {
	if response == nil {
		return nil
	}
	var resBodyBytes []byte
	if response.Body != nil {
		resBodyBytes, _ = io.ReadAll(response.Body)
		response.Body = io.NopCloser(bytes.NewBuffer(resBodyBytes))
	}

	return &http.Response{
		Header:        copyHeader(response.Header),
		StatusCode:    response.StatusCode,
		Status:        response.Status,
		Body:          io.NopCloser(bytes.NewBuffer(resBodyBytes)),
		Proto:         response.Proto,
		ProtoMinor:    response.ProtoMinor,
		ProtoMajor:    response.ProtoMajor,
		ContentLength: response.ContentLength,
	}
}

// copyHeader copies the given http.Header.
func copyHeader(src http.Header) http.Header {
	header := http.Header{}
	for name, values := range src {
		header[name] = values
	}
	return header
}
