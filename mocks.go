package spectest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	difflib "github.com/go-spectest/diff"
)

// Transport wraps components used to observe and manipulate the real request and response objects
type Transport struct {
	// httpClient is the http client used when networking is enabled.
	httpClient *http.Client
	// mocks is the list of mocks to use when mocking the request
	mocks Mocks
	// mockResponseDelayEnabled will enable mock response delay
	mockResponseDelayEnabled bool
	// observers is the list of observers to use when observing the request and response
	observers []Observe
	// debug is used to enable/disable debug logging
	debug *debug
	// nativeTransport is the native http.RoundTripper
	nativeTransport http.RoundTripper
	// specTest is the spectest instance
	specTest *SpecTest
}

// newTransport creates a new transport
// If you set httpClient to nil, http.DefaultClient will be used.
// If you set debug to nil, debug will be disabled.
func newTransport(
	mocks Mocks,
	httpClient *http.Client,
	debug *debug,
	mockResponseDelayEnabled bool,
	observers []Observe,
	specTest *SpecTest) *Transport {
	t := &Transport{
		mocks:                    mocks,
		httpClient:               httpClient,
		debug:                    debug,
		mockResponseDelayEnabled: mockResponseDelayEnabled,
		observers:                observers,
		specTest:                 specTest,
	}

	if httpClient != nil {
		t.nativeTransport = httpClient.Transport
	} else {
		t.nativeTransport = http.DefaultTransport
	}
	return t
}

// RoundTrip implementation intended to match a given expected mock request
// or throw an error with a list of reasons why no match was found.
func (r *Transport) RoundTrip(req *http.Request) (mockResponse *http.Response, err error) {
	defer func() {
		r.debug.mock(mockResponse, req)
	}()

	if r.observers != nil && len(r.observers) > 0 {
		defer func() {
			for _, observe := range r.observers {
				observe(mockResponse, req, r.specTest)
			}
		}()
	}

	matchedResponse, err := matches(req, r.mocks)
	if err != nil {
		if r.debug.isEnable() {
			fmt.Printf("failed to match mocks. Errors: %s\n", err)
		}
		return nil, err
	}

	res := buildResponseFromMock(matchedResponse)
	res.Request = req

	if matchedResponse.timeout {
		return nil, ErrTimeout
	}
	if r.mockResponseDelayEnabled && matchedResponse.fixedDelayMillis > 0 {
		time.Sleep(time.Duration(matchedResponse.fixedDelayMillis) * time.Millisecond)
	}
	return res, nil
}

// Hijack replace the transport implementation of the interaction under test in order to observe, mock and inject expectations
func (r *Transport) Hijack() {
	if r.httpClient != nil {
		r.httpClient.Transport = r
		return
	}
	http.DefaultTransport = r
}

// Reset replace the hijacked transport implementation of the interaction under test to the original implementation
func (r *Transport) Reset() {
	if r.httpClient != nil {
		r.httpClient.Transport = r.nativeTransport
		return
	}
	http.DefaultTransport = r.nativeTransport
}

// buildResponseFromMock builds a http.Response from a MockResponse
func buildResponseFromMock(mockResponse *MockResponse) *http.Response {
	if mockResponse == nil {
		return nil
	}

	// if the content type isn't set and the body contains json, set content type as json
	contentTypeHeader := mockResponse.headers["Content-Type"]
	var contentType string
	if len(mockResponse.body) > 0 {
		if len(contentTypeHeader) == 0 {
			if json.Valid([]byte(mockResponse.body)) {
				contentType = "application/json"
			} else {
				contentType = "text/plain"
			}
		} else {
			contentType = contentTypeHeader[0]
		}
	}

	res := &http.Response{
		Body:          io.NopCloser(strings.NewReader(mockResponse.body)),
		Header:        mockResponse.headers,
		StatusCode:    mockResponse.statusCode,
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: int64(len(mockResponse.body)),
	}
	for _, cookie := range mockResponse.cookies {
		if v := cookie.ToHTTPCookie().String(); v != "" {
			res.Header.Add("Set-Cookie", v)
		}
	}
	if contentType != "" {
		res.Header.Set("Content-Type", contentType)
	}
	return res
}

// Mock represents the entire interaction for a mock to be used for testing
type Mock struct {
	m *sync.Mutex
	// state is mock runnig state
	state *state
	// request is used to configure the request of the mock
	request *MockRequest
	// resopnse is used to configure the response of the mock
	response *MockResponse
	// httpClient is used to enable/disable networking for the test
	httpClient *http.Client
	// debugStandalone is used to enable/disable debug logging for standalone mocks
	debugStandalone *debug
	// execCount is used to track the number of times the mock has been executed
	execCount *execCount
}

// Mocks is a slice of Mock
type Mocks []*Mock

// len returns the length of the mocks
func (mocks Mocks) len() int {
	return len(mocks)
}

// findUnmatchedMocks returns a list of unmatched mocks.
// An unmatched mock is a mock that was not used, e.g. there was not a matching http Request for the mock
func (mocks Mocks) findUnmatchedMocks() []UnmatchedMock {
	var unmatchedMocks []UnmatchedMock
	for _, m := range mocks {
		if !m.state.isRunning() {
			unmatchedMocks = append(unmatchedMocks, UnmatchedMock{
				URL: *m.request.url,
			})
			break
		}
	}
	return unmatchedMocks
}

// Matches checks whether the given request matches the mock
func (m *Mock) Matches(req *http.Request) []error {
	var errs []error
	for _, matcher := range m.request.matchers {
		if matcherError := matcher(req, m.request); matcherError != nil {
			errs = append(errs, matcherError)
		}
	}
	return errs
}

// deepCopy deepCopy Mock.
func (m *Mock) deepCopy() *Mock {
	newMock := *m

	newMock.m = &sync.Mutex{}

	req := *m.request
	newMock.request = &req

	res := *m.response
	newMock.response = &res

	state := *m.state
	newMock.state = &state

	return &newMock
}

// MockRequest represents the http request side of a mock interaction
type MockRequest struct {
	mock               *Mock
	url                *url.URL
	method             string
	headers            map[string][]string
	basicAuth          basicAuth
	headerPresent      []string
	headerNotPresent   []string
	formData           map[string][]string
	formDataPresent    []string
	formDataNotPresent []string
	query              map[string][]string
	queryPresent       []string
	queryNotPresent    []string
	cookie             []Cookie
	cookiePresent      []string
	cookieNotPresent   []string
	body               string
	bodyRegexp         string
	matchers           []Matcher
}

// newMockRequest return new MockRequest
func newMockRequest(m *Mock) *MockRequest {
	return &MockRequest{
		mock:               m,
		headers:            map[string][]string{},
		headerPresent:      []string{},
		headerNotPresent:   []string{},
		formData:           map[string][]string{},
		formDataPresent:    []string{},
		formDataNotPresent: []string{},
		query:              map[string][]string{},
		queryPresent:       []string{},
		queryNotPresent:    []string{},
		cookie:             []Cookie{},
		cookiePresent:      []string{},
		cookieNotPresent:   []string{},
		matchers:           defaultMatchers(),
	}
}

// UnmatchedMock exposes some information about mocks that failed to match a request
type UnmatchedMock struct {
	URL url.URL
}

// MockResponse represents the http response side of a mock interaction
type MockResponse struct {
	mock             *Mock
	timeout          bool
	headers          map[string][]string
	cookies          []*Cookie
	body             string
	statusCode       int
	fixedDelayMillis int64
}

// newMockResponse return new MockResponse
func newMockResponse(m *Mock) *MockResponse {
	return &MockResponse{
		mock:    m,
		headers: map[string][]string{},
		cookies: []*Cookie{},
	}
}

// StandaloneMocks for using mocks outside of API tests context
type StandaloneMocks struct {
	mocks      Mocks
	httpClient *http.Client
	debug      *debug
}

// NewStandaloneMocks create a series of StandaloneMocks
func NewStandaloneMocks(mocks ...*Mock) *StandaloneMocks {
	return &StandaloneMocks{
		mocks: mocks,
		debug: newDebug(),
	}
}

// HTTPClient use the given http client
func (r *StandaloneMocks) HTTPClient(cli *http.Client) *StandaloneMocks {
	r.httpClient = cli
	return r
}

// Debug switch on debugging mode
func (r *StandaloneMocks) Debug() *StandaloneMocks {
	r.debug.enable()
	return r
}

// End finalizes the mock, ready for use
func (r *StandaloneMocks) End() func() {
	transport := newTransport(
		r.mocks,
		r.httpClient,
		r.debug,
		false,
		nil,
		nil,
	)
	resetFunc := func() { transport.Reset() }
	transport.Hijack()
	return resetFunc
}

// NewMock create a new mock, ready for configuration using the builder pattern
func NewMock() *Mock {
	mock := &Mock{
		debugStandalone: newDebug(),
		m:               &sync.Mutex{},
		state:           newState(),
		execCount:       newExecCount(1),
	}
	mock.request = newMockRequest(mock)
	mock.response = newMockResponse(mock)
	return mock
}

// Debug is used to set debug mode for mocks in standalone mode.
// This is overridden by the debug setting in the `SpecTest` struct
func (m *Mock) Debug() *Mock {
	m.debugStandalone.enable()
	return m
}

// HTTPClient allows the developer to provide a custom http client when using mocks
func (m *Mock) HTTPClient(cli *http.Client) *Mock {
	m.httpClient = cli
	return m
}

// Get configures the mock to match http method GET
func (m *Mock) Get(u string) *MockRequest {
	m.parseURL(u)
	m.request.method = http.MethodGet
	return m.request
}

// Getf configures the mock to match http method GET and supports formatting
func (m *Mock) Getf(format string, args ...interface{}) *MockRequest {
	return m.Get(fmt.Sprintf(format, args...))
}

// Put configures the mock to match http method PUT
func (m *Mock) Put(u string) *MockRequest {
	m.parseURL(u)
	m.request.method = http.MethodPut
	return m.request
}

// Putf configures the mock to match http method PUT and supports formatting
func (m *Mock) Putf(format string, args ...interface{}) *MockRequest {
	return m.Put(fmt.Sprintf(format, args...))
}

// Post configures the mock to match http method POST
func (m *Mock) Post(u string) *MockRequest {
	m.parseURL(u)
	m.request.method = http.MethodPost
	return m.request
}

// Postf configures the mock to match http method POST and supports formatting
func (m *Mock) Postf(format string, args ...interface{}) *MockRequest {
	return m.Post(fmt.Sprintf(format, args...))
}

// Delete configures the mock to match http method DELETE
func (m *Mock) Delete(u string) *MockRequest {
	m.parseURL(u)
	m.request.method = http.MethodDelete
	return m.request
}

// Deletef configures the mock to match http method DELETE and supports formatting
func (m *Mock) Deletef(format string, args ...interface{}) *MockRequest {
	return m.Delete(fmt.Sprintf(format, args...))
}

// Patch configures the mock to match http method PATCH
func (m *Mock) Patch(u string) *MockRequest {
	m.parseURL(u)
	m.request.method = http.MethodPatch
	return m.request
}

// Patchf configures the mock to match http method PATCH and supports formatting
func (m *Mock) Patchf(format string, args ...interface{}) *MockRequest {
	return m.Patch(fmt.Sprintf(format, args...))
}

// Head configures the mock to match http method HEAD
func (m *Mock) Head(u string) *MockRequest {
	m.parseURL(u)
	m.request.method = http.MethodHead
	return m.request
}

// Headf configures the mock to match http method HEAD and supports formatting
func (m *Mock) Headf(format string, args ...interface{}) *MockRequest {
	return m.Head(fmt.Sprintf(format, args...))
}

// Connect configures the mock to match http method CONNECT
func (m *Mock) Connect(u string) *MockRequest {
	m.parseURL(u)
	m.request.method = http.MethodConnect
	return m.request
}

// Connectf configures the mock to match http method CONNECT and supports formatting
func (m *Mock) Connectf(format string, args ...interface{}) *MockRequest {
	return m.Connect(fmt.Sprintf(format, args...))
}

// Options configures the mock to match http method OPTIONS
func (m *Mock) Options(u string) *MockRequest {
	m.parseURL(u)
	m.request.method = http.MethodOptions
	return m.request
}

// Optionsf configures the mock to match http method OPTIONS and supports formatting
func (m *Mock) Optionsf(format string, args ...interface{}) *MockRequest {
	return m.Options(fmt.Sprintf(format, args...))
}

// Trace configures the mock to match http method TRACE
func (m *Mock) Trace(u string) *MockRequest {
	m.parseURL(u)
	m.request.method = http.MethodTrace
	return m.request
}

// Tracef configures the mock to match http method TRACE and supports formatting
func (m *Mock) Tracef(format string, args ...interface{}) *MockRequest {
	return m.Trace(fmt.Sprintf(format, args...))
}

// Method configures mock to match given http method
func (m *Mock) Method(method string) *MockRequest {
	m.request.method = method
	return m.request
}

// parseURL parses the given url and sets it on the mock request
func (m *Mock) parseURL(u string) {
	parsed, err := url.Parse(u)
	if err != nil {
		panic(err)
	}
	m.request.url = parsed
}

// matches checks whether the given request matches any of the given mocks
func matches(req *http.Request, mocks Mocks) (*MockResponse, error) {
	mockError := newUnmatchedMockError()
	for mockNumber, mock := range mocks {
		mock.m.Lock() // lock is for isUsed when matches is called concurrently by RoundTripper
		if mock.state.isRunning() {
			mock.m.Unlock()
			continue
		}

		errs := mock.Matches(req)
		if len(errs) == 0 {
			mock.state.Start()
			mock.m.Unlock()
			return mock.response, nil
		}

		mockError = mockError.append(mockNumber+1, errs...)
		mock.m.Unlock()
	}

	return nil, mockError
}

// Body configures the mock request to match the given body
func (r *MockRequest) Body(b string) *MockRequest {
	r.body = b
	return r
}

// BodyRegexp configures the mock request to match the given body using the regexp matcher
func (r *MockRequest) BodyRegexp(b string) *MockRequest {
	r.body = b
	return r
}

// Bodyf configures the mock request to match the given body. Supports formatting the body
func (r *MockRequest) Bodyf(format string, args ...interface{}) *MockRequest {
	return r.Body(fmt.Sprintf(format, args...))
}

// BodyFromFile configures the mock request to match the given body from a file
func (r *MockRequest) BodyFromFile(f string) *MockRequest {
	b, err := os.ReadFile(filepath.Clean(f))
	if err != nil {
		panic(err)
	}
	r.body = string(b)
	return r
}

// JSON is a convenience method for setting the mock request body
func (r *MockRequest) JSON(v interface{}) *MockRequest {
	switch x := v.(type) {
	case string:
		r.body = x
	case []byte:
		r.body = string(x)
	default:
		asJSON, _ := json.Marshal(x)
		r.body = string(asJSON)
	}
	return r
}

// Header configures the mock request to match the given header
func (r *MockRequest) Header(key, value string) *MockRequest {
	normalizedKey := textproto.CanonicalMIMEHeaderKey(key)
	r.headers[normalizedKey] = append(r.headers[normalizedKey], value)
	return r
}

// Headers configures the mock request to match the given headers
func (r *MockRequest) Headers(headers map[string]string) *MockRequest {
	for k, v := range headers {
		normalizedKey := textproto.CanonicalMIMEHeaderKey(k)
		r.headers[normalizedKey] = append(r.headers[normalizedKey], v)
	}
	return r
}

// HeaderPresent configures the mock request to match when this header is present, regardless of value
func (r *MockRequest) HeaderPresent(key string) *MockRequest {
	r.headerPresent = append(r.headerPresent, key)
	return r
}

// HeaderNotPresent configures the mock request to match when the header is not present
func (r *MockRequest) HeaderNotPresent(key string) *MockRequest {
	r.headerNotPresent = append(r.headerNotPresent, key)
	return r
}

// BasicAuth configures the mock request to match the given basic auth parameters
func (r *MockRequest) BasicAuth(userName, password string) *MockRequest {
	r.basicAuth = newBasicAuth(userName, password)
	return r
}

// FormData configures the mock request to math the given form data
func (r *MockRequest) FormData(key string, values ...string) *MockRequest {
	r.formData[key] = append(r.formData[key], values...)
	return r
}

// FormDataPresent configures the mock request to match when the form data is present, regardless of values
func (r *MockRequest) FormDataPresent(key string) *MockRequest {
	r.formDataPresent = append(r.formDataPresent, key)
	return r
}

// FormDataNotPresent configures the mock request to match when the form data is not present
func (r *MockRequest) FormDataNotPresent(key string) *MockRequest {
	r.formDataNotPresent = append(r.formDataNotPresent, key)
	return r
}

// Query configures the mock request to match a query param
func (r *MockRequest) Query(key, value string) *MockRequest {
	r.query[key] = append(r.query[key], value)
	return r
}

// QueryParams configures the mock request to match a number of query params
func (r *MockRequest) QueryParams(queryParams map[string]string) *MockRequest {
	for k, v := range queryParams {
		r.query[k] = append(r.query[k], v)
	}
	return r
}

// QueryCollection configures the mock request to match a number of repeating query params, e.g. ?a=1&a=2&a=3
func (r *MockRequest) QueryCollection(queryParams map[string][]string) *MockRequest {
	for k, v := range queryParams {
		r.query[k] = append(r.query[k], v...)
	}
	return r
}

// QueryPresent configures the mock request to match when a query param is present, regardless of value
func (r *MockRequest) QueryPresent(key string) *MockRequest {
	r.queryPresent = append(r.queryPresent, key)
	return r
}

// QueryNotPresent configures the mock request to match when the query param is not present
func (r *MockRequest) QueryNotPresent(key string) *MockRequest {
	r.queryNotPresent = append(r.queryNotPresent, key)
	return r
}

// Cookie configures the mock request to match a cookie
func (r *MockRequest) Cookie(name, value string) *MockRequest {
	r.cookie = append(r.cookie, Cookie{name: &name, value: &value})
	return r
}

// CookiePresent configures the mock request to match when a cookie is present, regardless of value
func (r *MockRequest) CookiePresent(name string) *MockRequest {
	r.cookiePresent = append(r.cookiePresent, name)
	return r
}

// CookieNotPresent configures the mock request to match when a cookie is not present
func (r *MockRequest) CookieNotPresent(name string) *MockRequest {
	r.cookieNotPresent = append(r.cookieNotPresent, name)
	return r
}

// AddMatcher configures the mock request to match using a custom matcher
func (r *MockRequest) AddMatcher(matcher Matcher) *MockRequest {
	r.matchers = append(r.matchers, matcher)
	return r
}

// RespondWith finalizes the mock request phase of set up and allowing the definition of response attributes to be defined
func (r *MockRequest) RespondWith() *MockResponse {
	return r.mock.response
}

// Timeout forces the mock to return a http timeout
func (r *MockResponse) Timeout() *MockResponse {
	r.timeout = true
	return r
}

// Header respond with the given header
func (r *MockResponse) Header(key string, value string) *MockResponse {
	normalizedKey := textproto.CanonicalMIMEHeaderKey(key)
	r.headers[normalizedKey] = append(r.headers[normalizedKey], value)
	return r
}

// Headers respond with the given headers
func (r *MockResponse) Headers(headers map[string]string) *MockResponse {
	for k, v := range headers {
		normalizedKey := textproto.CanonicalMIMEHeaderKey(k)
		r.headers[normalizedKey] = append(r.headers[normalizedKey], v)
	}
	return r
}

// Cookies respond with the given cookies
func (r *MockResponse) Cookies(cookie ...*Cookie) *MockResponse {
	r.cookies = append(r.cookies, cookie...)
	return r
}

// Cookie respond with the given cookie
func (r *MockResponse) Cookie(name, value string) *MockResponse {
	r.cookies = append(r.cookies, NewCookie(name).Value(value))
	return r
}

// Body sets the mock response body
func (r *MockResponse) Body(body string) *MockResponse {
	r.body = body
	return r
}

// Bodyf sets the mock response body. Supports formatting
func (r *MockResponse) Bodyf(format string, args ...interface{}) *MockResponse {
	return r.Body(fmt.Sprintf(format, args...))
}

// BodyFromFile defines the mock response body from a file
func (r *MockResponse) BodyFromFile(f string) *MockResponse {
	b, err := os.ReadFile(filepath.Clean(f))
	if err != nil {
		panic(err)
	}
	r.body = string(b)
	return r
}

// JSON is a convenience method for setting the mock response body
func (r *MockResponse) JSON(v interface{}) *MockResponse {
	switch x := v.(type) {
	case string:
		r.body = x
	case []byte:
		r.body = string(x)
	default:
		asJSON, _ := json.Marshal(x)
		r.body = string(asJSON)
	}
	return r
}

// Status respond with the given status
func (r *MockResponse) Status(statusCode int) *MockResponse {
	r.statusCode = statusCode
	return r
}

// FixedDelay will return the response after the given number of milliseconds.
// SpecTest::EnableMockResponseDelay must be set for this to take effect.
// If Timeout is set this has no effect.
func (r *MockResponse) FixedDelay(delay int64) *MockResponse {
	r.fixedDelayMillis = delay
	return r
}

// Times respond the given number of times
func (r *MockResponse) Times(times uint) *MockResponse {
	r.mock.execCount.updateExpectCount(times)
	return r
}

// End finalizes the response definition phase in order for the mock to be used
func (r *MockResponse) End() *Mock {
	return r.mock
}

// EndStandalone finalizes the response definition of standalone mocks
func (r *MockResponse) EndStandalone(other ...*Mock) func() {
	transport := newTransport(
		append(Mocks{r.mock}, other...),
		r.mock.httpClient,
		r.mock.debugStandalone,
		false,
		nil,
		nil,
	)
	resetFunc := func() { transport.Reset() }
	transport.Hijack()
	return resetFunc
}

// Matcher type accepts the actual request and a mock request to match against.
// Will return an error that describes why there was a mismatch if the inputs do not match or nil if they do.
type Matcher func(*http.Request, *MockRequest) error

// defaultMatchers returns the default list of matchers used by the mock server.
func defaultMatchers() []Matcher {
	return []Matcher{
		pathMatcher,
		hostMatcher,
		schemeMatcher,
		methodMatcher,
		headerMatcher,
		basicAuthMatcher,
		headerPresentMatcher,
		headerNotPresentMatcher,
		queryParamMatcher,
		queryPresentMatcher,
		queryNotPresentMatcher,
		formDataMatcher,
		formDataPresentMatcher,
		formDataNotPresentMatcher,
		bodyMatcher,
		bodyRegexpMatcher,
		cookieMatcher,
		cookiePresentMatcher,
		cookieNotPresentMatcher,
	}
}

// pathMatcher compares the path of the received HTTP request with the path specified in the mock request.
// If the paths match, it returns nil, indicating a successful match. If the paths do not match,
// it attempts to match using regular expressions. If a match is found, it returns an error describing
// the mismatch. If no match is found, it returns nil. The error message includes details about
// the received path and the expected mock path that did not match.
func pathMatcher(r *http.Request, spec *MockRequest) error {
	receivedPath := r.URL.Path
	mockPath := spec.url.Path
	if receivedPath == mockPath {
		return nil
	}
	matched, err := regexp.MatchString(mockPath, receivedPath)
	return errorOrNil(matched && err == nil, func() string {
		return fmt.Sprintf("received path %s did not match mock path %s", receivedPath, mockPath)
	})
}

// hostMatcher compares the host of the received HTTP request with the host specified in the mock request.
// If the hosts match, it returns nil, indicating a successful match. If the hosts do not match,
// it attempts to match using regular expressions. If a match is found, it returns an error describing
// the mismatch. If no match is found, it returns nil. The error message includes details about
// the received host and the expected mock host that did not match.
func hostMatcher(r *http.Request, spec *MockRequest) error {
	receivedHost := r.Host
	if receivedHost == "" {
		receivedHost = r.URL.Host
	}
	mockHost := spec.url.Host
	if mockHost == "" {
		return nil
	}
	if receivedHost == mockHost {
		return nil
	}
	matched, err := regexp.MatchString(mockHost, r.URL.Path)
	return errorOrNil(matched && err != nil, func() string {
		return fmt.Sprintf("received host %s did not match mock host %s", receivedHost, mockHost)
	})
}

// methodMatcher compares the HTTP request method (GET, POST, PUT, etc.) with the method specified in the mock request.
// If the methods match, it returns nil, indicating a successful match. If the methods do not match,
// it returns an error describing the mismatch. If the mock method is an empty string, it matches any request method.
// The error message includes details about the received method and the expected mock method that did not match.
func methodMatcher(r *http.Request, spec *MockRequest) error {
	receivedMethod := r.Method
	mockMethod := spec.method
	if receivedMethod == mockMethod {
		return nil
	}
	if mockMethod == "" {
		return nil
	}
	return fmt.Errorf("received method %s did not match mock method %s", receivedMethod, mockMethod)
}

// schemeMatcher compares the scheme (http, https, etc.) of the received HTTP request URL
// with the scheme specified in the mock request URL.
// If the schemes match, it returns nil, indicating a successful match. If the schemes do not match,
// it returns an error describing the mismatch. If either the received or mock scheme is an empty string,
// it matches any scheme. The error message includes details about the received scheme and the expected
// mock scheme that did not match.
func schemeMatcher(r *http.Request, spec *MockRequest) error {
	receivedScheme := r.URL.Scheme
	mockScheme := spec.url.Scheme
	if receivedScheme == "" {
		return nil
	}
	if mockScheme == "" {
		return nil
	}
	return errorOrNil(receivedScheme == mockScheme, func() string {
		return fmt.Sprintf("received scheme %s did not match mock scheme %s", receivedScheme, mockScheme)
	})
}

// headerMatcher compares the headers of the received HTTP request with the headers specified in the mock request.
// It checks each header key-value pair in the mock request against the corresponding header values in the received request.
// If all the headers in the mock request match the received request headers (based on regular expressions),
// it returns nil, indicating a successful match. If any header does not match, it returns an error describing the mismatch.
// The error message includes details about the specific headers that did not match between the received and expected requests.
func headerMatcher(req *http.Request, spec *MockRequest) error {
	mockHeaders := spec.headers
	for key, values := range mockHeaders {
		var match bool
		var err error
		receivedHeaders := req.Header

		for _, field := range receivedHeaders[key] {
			for _, value := range values {
				match, err = regexp.MatchString(value, field)
				if err != nil {
					return fmt.Errorf("failed to parse regexp for header %s with value %s", key, value)
				}
			}
			if match {
				break
			}
		}
		if !match {
			return fmt.Errorf("not all of received headers %s matched expected mock headers %s", receivedHeaders, mockHeaders)
		}
	}
	return nil
}

// basicAuthMatcher compares the basic auth credentials of the received HTTP request with the credentials specified in the mock request.
func basicAuthMatcher(req *http.Request, spec *MockRequest) error {
	if spec.basicAuth.isUserNameEmpty() || spec.basicAuth.isPasswordEmpty() {
		return nil
	}
	username, password, ok := req.BasicAuth()
	if !ok {
		return errors.New("request did not contain valid HTTP Basic Authentication string")
	}
	return spec.basicAuth.auth(username, password)
}

// headerPresentMatcher compares the headers of the received HTTP request with the headers specified in the mock request.
func headerPresentMatcher(req *http.Request, spec *MockRequest) error {
	for _, header := range spec.headerPresent {
		if req.Header.Get(header) == "" {
			return fmt.Errorf("expected header '%s' was not present", header)
		}
	}
	return nil
}

// headerNotPresentMatcher checks that specific headers are not present in the received HTTP request.
// It compares the list of headers specified in the mock request with the headers in the received request.
// If any of the specified headers are found in the received request, it returns an error indicating
// that an unexpected header was present. If all specified headers are not present, it returns nil,
// indicating a successful match.
func headerNotPresentMatcher(req *http.Request, spec *MockRequest) error {
	for _, header := range spec.headerNotPresent {
		if req.Header.Get(header) != "" {
			return fmt.Errorf("unexpected header '%s' was present", header)
		}
	}
	return nil
}

// queryParamMatcher compares the query parameters of the received HTTP request with the query parameters specified in the mock request.
// It checks each query parameter key-value pair in the mock request against the corresponding query parameter values in the received request.
// If all the query parameters in the mock request match the received request query parameters (based on regular expressions),
// it returns nil, indicating a successful match. If any query parameter does not match, it returns an error describing the mismatch.
// The error message includes details about the specific query parameters that did not match between the received and expected requests.
func queryParamMatcher(req *http.Request, spec *MockRequest) error {
	mockQueryParams := spec.query
	for key, values := range mockQueryParams {
		receivedQueryParams := req.URL.Query()

		if _, ok := receivedQueryParams[key]; !ok {
			return fmt.Errorf("not all of received query params %s matched expected mock query params %s", receivedQueryParams, mockQueryParams)
		}

		found := 0
		for _, field := range receivedQueryParams[key] {
			for _, value := range values {
				match, err := regexp.MatchString(value, field)
				if err != nil {
					return fmt.Errorf("failed to parse regexp for query param %s with value %s", key, value)
				}

				if match {
					found++
				}
			}
		}

		if found != len(values) {
			return fmt.Errorf("not all of received query params %s matched expected mock query params %s", receivedQueryParams, mockQueryParams)
		}
	}
	return nil
}

// queryPresentMatcher checks if specific query parameters specified in the mock request are present in the received HTTP request.
// It compares each expected query parameter with the corresponding query parameter in the received request's URL.
// If any expected query parameter is not found in the received request, it returns an error indicating
// that the expected query parameter was not received. If all expected query parameters are present,
// it returns nil, indicating a successful match.
func queryPresentMatcher(req *http.Request, spec *MockRequest) error {
	for _, query := range spec.queryPresent {
		if req.URL.Query().Get(query) == "" {
			return fmt.Errorf("expected query param %s not received", query)
		}
	}
	return nil
}

// queryNotPresentMatcher checks if specific query parameters specified in the mock request are not present in the received HTTP request.
// It compares each query parameter specified in the `queryNotPresent` list with the corresponding query parameter in the received request's URL.
// If any query parameter from the `queryNotPresent` list is found in the received request, it returns an error indicating
// that an unexpected query parameter was present. If none of the specified query parameters are present,
// it returns nil, indicating a successful match.
func queryNotPresentMatcher(req *http.Request, spec *MockRequest) error {
	for _, query := range spec.queryNotPresent {
		if req.URL.Query().Get(query) != "" {
			return fmt.Errorf("unexpected query param '%s' present", query)
		}
	}
	return nil
}

// formDataMatcher checks if specific form data parameters specified in the mock request are present in the received HTTP request.
// It compares each form data parameter key-value pair in the `formData` map with the corresponding form data parameters in the received request.
// If all the form data parameters in the mock request match the received request form data parameters (based on regular expressions),
// it returns nil, indicating a successful match. If any form data parameter does not match, it returns an error describing the mismatch.
// The error message includes details about the specific form data parameters that did not match between the received and expected requests.
func formDataMatcher(req *http.Request, spec *MockRequest) error {
	mockFormData := spec.formData

	for key, values := range mockFormData {
		r := copyHTTPRequest(req)
		err := r.ParseForm()
		if err != nil {
			return errors.New("unable to parse form data")
		}

		receivedFormData := r.PostForm

		if _, ok := receivedFormData[key]; !ok {
			return fmt.Errorf("not all of received form data values %s matched expected mock form data values %s",
				receivedFormData, mockFormData)
		}

		found := 0
		for _, field := range receivedFormData[key] {
			for _, value := range values {
				match, err := regexp.MatchString(value, field)
				if err != nil {
					return fmt.Errorf("failed to parse regexp for form data %s with value %s", key, value)
				}

				if match {
					found++
				}
			}
		}

		if found != len(values) {
			return fmt.Errorf("not all of received form data values %s matched expected mock form data values %s", receivedFormData, mockFormData)
		}
	}
	return nil
}

// formDataPresentMatcher checks if specific form data parameters specified in the mock request are present in the received HTTP request.
// It compares each form data parameter key specified in the `formDataPresent` list with the corresponding form data parameters in the received request.
// If any expected form data parameter is not found in the received request, it returns an error indicating
// that the expected form data parameter was not received. If all expected form data parameters are present,
// it returns nil, indicating a successful match.
func formDataPresentMatcher(req *http.Request, spec *MockRequest) error {
	if len(spec.formDataPresent) > 0 {
		r := copyHTTPRequest(req)
		if err := r.ParseForm(); err != nil {
			return errors.New("unable to parse form data")
		}

		receivedFormData := r.PostForm

		for _, key := range spec.formDataPresent {
			if _, ok := receivedFormData[key]; !ok {
				return fmt.Errorf("expected form data key %s not received", key)
			}
		}
	}
	return nil
}

// formDataPresentMatcher checks if specific form data parameters specified in the mock request are present in the received HTTP request.
// It compares each form data parameter key specified in the `formDataPresent` list with the corresponding form data parameters in the received request.
// If any expected form data parameter is not found in the received request, it returns an error indicating
// that the expected form data parameter was not received. If all expected form data parameters are present,
// it returns nil, indicating a successful match.
func formDataNotPresentMatcher(req *http.Request, spec *MockRequest) error {
	if len(spec.formDataNotPresent) > 0 {
		r := copyHTTPRequest(req)
		if err := r.ParseForm(); err != nil {
			return errors.New("unable to parse form data")
		}

		receivedFormData := r.PostForm

		for _, key := range spec.formDataNotPresent {
			if _, ok := receivedFormData[key]; ok {
				return fmt.Errorf("did not expect a form data key %s", key)
			}
		}
	}
	return nil
}

// cookieMatcher checks if specific cookies specified in the mock request are present in the received HTTP request.
func cookieMatcher(req *http.Request, spec *MockRequest) error {
	for i := range spec.cookie {
		foundCookie, _ := req.Cookie(*spec.cookie[i].name)
		if foundCookie == nil {
			return fmt.Errorf("expected cookie with name '%s' not received", *spec.cookie[i].name)
		}
		if _, mismatches := compareCookies(&spec.cookie[i], foundCookie); len(mismatches) > 0 {
			return fmt.Errorf("failed to match cookie: %v", mismatches)
		}
	}
	return nil
}

// cookiePresentMatcher checks if specific cookies specified in the mock request are present in the received HTTP request.
func cookiePresentMatcher(req *http.Request, spec *MockRequest) error {
	for _, c := range spec.cookiePresent {
		foundCookie, _ := req.Cookie(c)
		if foundCookie == nil {
			return fmt.Errorf("expected cookie with name '%s' not received", c)
		}
	}
	return nil
}

// cookieNotPresentMatcher checks if specific cookies specified in the mock request are present in the received HTTP request.
func cookieNotPresentMatcher(req *http.Request, spec *MockRequest) error {
	for _, c := range spec.cookieNotPresent {
		foundCookie, _ := req.Cookie(c)
		if foundCookie != nil {
			return fmt.Errorf("did not expect a cookie with name '%s'", c)
		}
	}
	return nil
}

// bodyMatcher compares the body of the received HTTP request with the body specified in the mock request.
func bodyMatcher(req *http.Request, spec *MockRequest) error {
	mockBody := spec.body

	if len(mockBody) == 0 {
		return nil
	}

	if req.Body == nil {
		return errors.New("expected a body but received none")
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("expected a body but received none")
	}

	// replace body so it can be read again
	req.Body = io.NopCloser(bytes.NewReader(body))

	// Perform exact string match
	bodyStr := string(body)
	if bodyStr == mockBody {
		return nil
	}

	// Perform JSON match
	var reqJSON interface{}
	reqJSONErr := json.Unmarshal(body, &reqJSON)

	var matchJSON interface{}
	specJSONErr := json.Unmarshal([]byte(mockBody), &matchJSON)

	isJSON := reqJSONErr == nil && specJSONErr == nil
	if isJSON && reflect.DeepEqual(reqJSON, matchJSON) {
		return nil
	}

	if isJSON {
		return fmt.Errorf("received body did not match expected mock body\n%s", diff(matchJSON, reqJSON))
	}

	return fmt.Errorf("received body did not match expected mock body\n%s", diff(mockBody, bodyStr))
}

// bodyRegexpMatcher checks if the body of the received HTTP request matches the regular expression specified in the mock request.
// If the regular expression is empty, it returns nil, indicating a successful match without checking the body.
// If the body of the received request does not match the regular expression, it returns an error describing the mismatch.
// The error message includes the details of the received body and the expected regular expression that did not match.
func bodyRegexpMatcher(req *http.Request, spec *MockRequest) error {
	expression := spec.bodyRegexp

	if len(expression) == 0 {
		return nil
	}

	if req.Body == nil {
		return errors.New("expected a body but received none")
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("expected a body but received none")
	}

	// replace body so it can be read again
	req.Body = io.NopCloser(bytes.NewReader(body))

	// Perform regexp match
	bodyStr := string(body)
	match, _ := regexp.MatchString(expression, bodyStr)
	if match {
		return nil
	}

	return fmt.Errorf("received body did not match expected mock body\n%s", diff(expression, bodyStr))
}

var spewConfig = spew.ConfigState{
	Indent:                  " ",
	DisablePointerAddresses: true,
	DisableCapacities:       true,
	SortKeys:                true,
	DisableMethods:          true,
}

func diff(expected interface{}, actual interface{}) string {
	if expected == nil || actual == nil {
		return ""
	}

	et, ek := typeAndKind(expected)
	at, _ := typeAndKind(actual)

	if et != at {
		return ""
	}

	if ek != reflect.Struct && ek != reflect.Map && ek != reflect.Slice && ek != reflect.Array && ek != reflect.String {
		return ""
	}

	var e, a string
	if et != reflect.TypeOf("") {
		e = spewConfig.Sdump(expected)
		a = spewConfig.Sdump(actual)
	} else {
		e = reflect.ValueOf(expected).String()
		a = reflect.ValueOf(actual).String()
	}

	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(e),
		B:        difflib.SplitLines(a),
		FromFile: "Expected",
		FromDate: "",
		ToFile:   "Actual",
		ToDate:   "",
		Context:  2,
	})

	return "\n\nDiff:\n" + diff
}

func typeAndKind(v interface{}) (reflect.Type, reflect.Kind) {
	t := reflect.TypeOf(v)
	k := t.Kind()

	if k == reflect.Ptr {
		t = t.Elem()
		k = t.Kind()
	}
	return t, k
}

type mockInteraction struct {
	request   *http.Request
	response  *http.Response
	timestamp time.Time
}

func (r *mockInteraction) GetRequestHost() string {
	host := r.request.Host
	if host == "" {
		host = r.request.URL.Host
	}
	return host
}

// execCount is used to track the number of times a mock has been executed.
type execCount struct {
	// expect is the expected number of times the mock will be executed.
	expect uint
	// actual is the actual number of times the mock has been executed.
	actual uint
}

// newExecCount creates a new execCount with the given expected number of executions.
func newExecCount(expect uint) *execCount {
	return &execCount{expect: expect}
}

// updateExpectCount updates the expected number of executions.
func (e *execCount) updateExpectCount(expect uint) {
	e.expect = expect
}

// isComplete returns true if the actual number of executions matches the expected number of executions.
func (e *execCount) isComplete() bool {
	return e.actual == e.expect
}

// state is used to track the state of a mock. It's very simple state machine
type state struct {
	running bool
}

func newState() *state {
	return &state{}
}

// Start sets the state to running.
func (s *state) Start() {
	s.running = true
}

// Stop sets the state to not running.
func (s *state) Stop() {
	s.running = false
}

// isRunning returns true if the state is running.
func (s *state) isRunning() bool {
	return s.running
}
