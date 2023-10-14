// Package spectest is simple and extensible behavioral testing library for Go. You can use api test to simplify REST API,
// HTTP handler and e2e tests. (forked from steinfletcher/apitest)
package spectest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	runtimeDebug "runtime/debug"
	"sort"
	"strings"
	"time"
)

// SpecTest is the top level struct holding the test spec
type SpecTest struct {
	// debug is used to log the http wire representation of all http interactions
	debug *debug
	// mockResponseDelayEnabled will turn on mock response delays (defaults to OFF)
	mockResponseDelayEnabled bool
	// network is used to enable/disable networking for the test
	network *network
	// reporter is the report formatter.
	reporter ReportFormatter
	// verifier is the assertion implementation. Default is DefaultVerifier.
	verifier Verifier
	// recorder is the test result recorder.
	recorder *Recorder
	// handler is the http handler that is invoked when the test is run
	handler http.Handler
	// name is the name of the test. It will appear in the test report as sub title.
	name string
	// request is the request spec. It is called by the test runner to build the request.
	request *Request
	// response is the expected response. It is called by the test runner to assert the response.
	response *Response
	// observers is a list of functions that will be called on completion of the test.
	// It is used to capture the inbound request and final response.
	observers []Observe
	// mocksObservers is a list of functions that will be called on completion of the test.
	// It is used to capture the mock request and response.
	mocksObservers []Observe
	// mocks is a list of mocks that will be used to intercept the request.
	mocks []*Mock
	// t is the testing.T instance.
	t TestingT
	// httpClient is the http client used when networking is enabled
	httpClient *http.Client
	// httpRequest is the native `http.Request`
	httpRequest *http.Request
	// transport is the http transport used when networking is enabled
	transport *Transport
	// meta is the meta data for the test report.
	meta *Meta
	// interval is the time interval for the test report.
	interval *Interval
}

// Observe will be called by with the request and response on completion
type Observe func(*http.Response, *http.Request, *SpecTest)

// New creates a new api test. The name is optional and will appear in test reports.
// The name is only used name[0]. name[1]... are ignored.
func New(name ...string) *SpecTest {
	specTest := &SpecTest{
		debug:    newDebug(),
		interval: NewInterval(),
		meta:     newMeta(),
		network:  newNetwork(),
	}
	specTest.request = newRequest(specTest)
	specTest.response = newResponse(specTest)

	if len(name) > 0 {
		specTest.name = name[0]
	}
	return specTest
}

// Handler is a convenience method for creating a new spectest with a handler
func Handler(handler http.Handler) *SpecTest {
	return New().Handler(handler)
}

// HandlerFunc is a convenience method for creating a new spectest with a handler func
func HandlerFunc(handlerFunc http.HandlerFunc) *SpecTest {
	return New().HandlerFunc(handlerFunc)
}

// EnableNetworking will enable networking for provided clients
// If no clients are provided, the default http client will be used.
// If multiple clients are provided, the first client will be used.
func (s *SpecTest) EnableNetworking(clients ...*http.Client) *SpecTest {
	s.network.enabled = true
	if len(clients) == 1 {
		s.network.Client = clients[0]
		return s
	}
	return s
}

// EnableMockResponseDelay turns on mock response delays (defaults to OFF)
func (s *SpecTest) EnableMockResponseDelay() *SpecTest {
	s.mockResponseDelayEnabled = true
	return s
}

// Debug logs to the console the http wire representation of all http interactions
// that are intercepted by spectest. This includes the inbound request to the application
// under test, the response returned by the application and any interactions that are
// intercepted by the mock server.
func (s *SpecTest) Debug() *SpecTest {
	s.debug.enable()
	return s
}

// Report provides a hook to add custom formatting to the output of the test
func (s *SpecTest) Report(reporter ReportFormatter) *SpecTest {
	s.reporter = reporter
	return s
}

// Recorder provides a hook to add a recorder to the test
func (s *SpecTest) Recorder(recorder *Recorder) *SpecTest {
	s.recorder = recorder
	return s
}

// CustomHost set hostname.
// This method is not change the host in the request. It is only for the report.
func (s *SpecTest) CustomHost(host string) *SpecTest {
	s.meta.Host = host
	return s
}

// CustomReportName allows the consumer to override the default report file name.
func (s *SpecTest) CustomReportName(name string) *SpecTest {
	s.meta.ReportFileName = name
	return s
}

// Handler defines the http handler that is invoked when the test is run
func (s *SpecTest) Handler(handler http.Handler) *SpecTest {
	s.handler = handler
	return s
}

// HandlerFunc defines the http handler that is invoked when the test is run
func (s *SpecTest) HandlerFunc(handlerFunc http.HandlerFunc) *SpecTest {
	s.handler = handlerFunc
	return s
}

// Mocks is a builder method for setting the mocks.
// A mock that expects multiple executions will reset the expected call
// count to 1 when generated as a mock that expects a single execution.
func (s *SpecTest) Mocks(mocks ...*Mock) *SpecTest {
	var m []*Mock
	for i := range mocks {
		times := mocks[i].response.mock.execCount.expect
		for j := 1; j <= int(times); j++ {
			mockCopy := mocks[i].copy()
			mockCopy.execCount = newExecCount(1)
			m = append(m, mockCopy)
		}
	}
	s.mocks = m
	return s
}

// HTTPClient allows the developer to provide a custom http client when using mocks
func (s *SpecTest) HTTPClient(client *http.Client) *SpecTest {
	s.httpClient = client
	return s
}

// Observe is a builder method for setting the observers
func (s *SpecTest) Observe(observers ...Observe) *SpecTest {
	s.observers = observers
	return s
}

// ObserveMocks is a builder method for setting the mocks observers
func (s *SpecTest) ObserveMocks(observer Observe) *SpecTest {
	s.mocksObservers = append(s.mocksObservers, observer)
	return s
}

// Request returns the request spec
func (s *SpecTest) Request() *Request {
	return s.request
}

// Response returns the expected response
func (s *SpecTest) Response() *Response {
	return s.response
}

// Intercept will be called before the request is made.
// Updates to the request will be reflected in the test
type Intercept func(*http.Request)

type pair struct {
	l string
	r string
}

// Intercept is a builder method for setting the request interceptor
func (s *SpecTest) Intercept(interceptor Intercept) *SpecTest {
	s.request.interceptor = interceptor
	return s
}

// Verifier allows consumers to override the verification implementation.
func (s *SpecTest) Verifier(v Verifier) *SpecTest {
	s.verifier = v
	return s
}

// Method is a builder method for setting the http method of the request
func (s *SpecTest) Method(method string) *Request {
	s.request.method = method
	return s.request
}

// HTTPRequest defines the native `http.Request`
func (s *SpecTest) HTTPRequest(req *http.Request) *Request {
	s.httpRequest = req
	return s.request
}

// Get is a convenience method for setting the request as http.MethodGet
func (s *SpecTest) Get(url string) *Request {
	s.request.method = http.MethodGet
	s.request.url = url
	return s.request
}

// Getf is a convenience method that adds formatting support to Get
func (s *SpecTest) Getf(format string, args ...interface{}) *Request {
	return s.Get(fmt.Sprintf(format, args...))
}

// Post is a convenience method for setting the request as http.MethodPost
func (s *SpecTest) Post(url string) *Request {
	s.request.method = http.MethodPost
	s.request.url = url
	return s.request
}

// Postf is a convenience method that adds formatting support to Post
func (s *SpecTest) Postf(format string, args ...interface{}) *Request {
	return s.Post(fmt.Sprintf(format, args...))
}

// Put is a convenience method for setting the request as http.MethodPut
func (s *SpecTest) Put(url string) *Request {
	s.request.method = http.MethodPut
	s.request.url = url
	return s.request
}

// Putf is a convenience method that adds formatting support to Put
func (s *SpecTest) Putf(format string, args ...interface{}) *Request {
	return s.Put(fmt.Sprintf(format, args...))
}

// Delete is a convenience method for setting the request as http.MethodDelete
func (s *SpecTest) Delete(url string) *Request {
	s.request.method = http.MethodDelete
	s.request.url = url
	return s.request
}

// Deletef is a convenience method that adds formatting support to Delete
func (s *SpecTest) Deletef(format string, args ...interface{}) *Request {
	return s.Delete(fmt.Sprintf(format, args...))
}

// Patch is a convenience method for setting the request as http.MethodPatch
func (s *SpecTest) Patch(url string) *Request {
	s.request.method = http.MethodPatch
	s.request.url = url
	return s.request
}

// Patchf is a convenience method that adds formatting support to Patch
func (s *SpecTest) Patchf(format string, args ...interface{}) *Request {
	return s.Patch(fmt.Sprintf(format, args...))
}

// Head is a convenience method for setting the request as http.MethodHead
func (s *SpecTest) Head(url string) *Request {
	s.request.method = http.MethodHead
	s.request.url = url
	return s.request
}

// Headf is a convenience method that adds formatting support to Head
func (s *SpecTest) Headf(format string, args ...interface{}) *Request {
	return s.Head(fmt.Sprintf(format, args...))
}

// Options is a convenience method for setting the request as http.MethodOptions
func (s *SpecTest) Options(url string) *Request {
	s.request.method = http.MethodOptions
	s.request.url = url
	return s.request
}

// Optionsf is a convenience method that adds formatting support to Options
func (s *SpecTest) Optionsf(format string, args ...interface{}) *Request {
	return s.Options(fmt.Sprintf(format, args...))
}

// Connect is a convenience method for setting the request as http.MethodConnect
func (s *SpecTest) Connect(url string) *Request {
	s.request.method = http.MethodConnect
	s.request.url = url
	return s.request
}

// Connectf is a convenience method that adds formatting support to Connect
func (s *SpecTest) Connectf(format string, args ...interface{}) *Request {
	return s.Connect(fmt.Sprintf(format, args...))
}

// Trace is a convenience method for setting the request as http.MethodTrace
func (s *SpecTest) Trace(url string) *Request {
	s.request.method = http.MethodTrace
	s.request.url = url
	return s.request
}

// Tracef is a convenience method that adds formatting support to Trace
func (s *SpecTest) Tracef(format string, args ...interface{}) *Request {
	return s.Trace(fmt.Sprintf(format, args...))
}

func (s *SpecTest) report() *http.Response {
	var capturedInboundReq *http.Request
	var capturedFinalRes *http.Response
	var capturedMockInteractions []*mockInteraction

	s.observers = append(s.observers, func(finalRes *http.Response, inboundReq *http.Request, a *SpecTest) {
		capturedFinalRes = copyHTTPResponse(finalRes)
		defer func() {
			if err := capturedFinalRes.Body.Close(); err != nil {
				panic(err) // FIXME: handle error
			}
		}()
		capturedInboundReq = copyHTTPRequest(inboundReq)
	})

	s.mocksObservers = append(s.mocksObservers, func(mockRes *http.Response, mockReq *http.Request, a *SpecTest) {
		capturedMockInteractions = append(capturedMockInteractions, &mockInteraction{
			request:   copyHTTPRequest(mockReq),
			response:  copyHTTPResponse(mockRes),
			timestamp: time.Now().UTC(),
		})
	})

	if s.recorder == nil {
		s.recorder = NewTestRecorder()
	}
	defer s.recorder.Reset()

	res := s.response.runTest()

	s.recorder.
		AddTitle(fmt.Sprintf("%s %s", capturedInboundReq.Method, capturedInboundReq.URL.String())).
		AddSubTitle(s.name).
		AddHTTPRequest(HTTPRequest{
			Source:    ConsumerDefaultName,
			Target:    SystemUnderTestDefaultName,
			Value:     capturedInboundReq,
			Timestamp: s.interval.Started,
		})

	for _, interaction := range capturedMockInteractions {
		s.recorder.AddHTTPRequest(HTTPRequest{
			Source:    SystemUnderTestDefaultName,
			Target:    interaction.GetRequestHost(),
			Value:     interaction.request,
			Timestamp: interaction.timestamp,
		})
		if interaction.response != nil {
			s.recorder.AddHTTPResponse(HTTPResponse{
				Source:    interaction.GetRequestHost(),
				Target:    SystemUnderTestDefaultName,
				Value:     interaction.response,
				Timestamp: interaction.timestamp,
			})
		}
	}

	s.recorder.AddHTTPResponse(HTTPResponse{
		Source:    SystemUnderTestDefaultName,
		Target:    ConsumerDefaultName,
		Value:     capturedFinalRes,
		Timestamp: s.interval.Finished,
	})

	sort.Slice(s.recorder.Events, func(i, j int) bool {
		return s.recorder.Events[i].GetTime().Before(s.recorder.Events[j].GetTime())
	})

	meta := newMeta()
	meta.StatusCode = capturedFinalRes.StatusCode
	meta.Path = capturedInboundReq.URL.String()
	meta.Method = capturedInboundReq.Method
	meta.Duration = s.interval.Duration().Nanoseconds()
	meta.Name = s.name
	meta.ReportFileName = s.meta.ReportFileName
	if s.meta.Host != "" {
		meta.Host = s.meta.Host
	}

	s.recorder.AddMeta(meta)
	s.reporter.Format(s.recorder)

	return res
}

func (s *SpecTest) assertMocks() {
	for _, mock := range s.mocks {
		if !mock.isUsed && mock.execCount.isComplete() {
			s.verifier.Fail(s.t, "mock was not invoked expected times", failureMessageArgs{Name: s.name})
		}
	}
}

func (s *SpecTest) assertFunc(res *http.Response, req *http.Request) {
	if len(s.response.assert) > 0 {
		for _, assertFn := range s.response.assert {
			err := assertFn(copyHTTPResponse(res), copyHTTPRequest(req))
			if err != nil {
				s.verifier.NoError(s.t, err, failureMessageArgs{Name: s.name})
			}
		}
	}
}

func (s *SpecTest) doRequest() (*http.Response, *http.Request) {
	req := s.buildRequest()
	if s.request.interceptor != nil {
		s.request.interceptor(req)
	}
	resRecorder := httptest.NewRecorder()

	s.debug.dumpRequest(req)

	var res *http.Response
	var err error
	if !s.network.isEnable() {
		s.serveHTTP(resRecorder, copyHTTPRequest(req))
		res = resRecorder.Result()
	} else {
		res, err = s.network.Do(copyHTTPRequest(req))
		if err != nil {
			s.t.Fatal(err)
		}
	}

	s.debug.dumpResponse(res)

	return res, req
}

func (s *SpecTest) serveHTTP(res *httptest.ResponseRecorder, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			s.t.Fatalf("%s: %s", err, runtimeDebug.Stack())
		}
	}()

	s.handler.ServeHTTP(res, req)
}

func (s *SpecTest) assertResponse(res *http.Response) {
	if s.response.status != 0 {
		s.verifier.Equal(s.t, s.response.status, res.StatusCode, fmt.Sprintf("Status code %d not equal to %d", res.StatusCode, s.response.status), failureMessageArgs{Name: s.name})
	}

	if s.response.body != "" {
		var resBodyBytes []byte
		if res.Body != nil {
			resBodyBytes, _ = io.ReadAll(res.Body)
			res.Body = io.NopCloser(bytes.NewBuffer(resBodyBytes))
		}
		if json.Valid([]byte(s.response.body)) {
			s.verifier.JSONEq(s.t, s.response.body, string(resBodyBytes), failureMessageArgs{Name: s.name})
		} else {
			s.verifier.Equal(s.t, s.response.body, string(resBodyBytes), failureMessageArgs{Name: s.name})
		}
	}
}

func (s *SpecTest) assertCookies(response *http.Response) {
	if len(s.response.cookies) > 0 {
		for _, expectedCookie := range s.response.cookies {
			var mismatchedFields []string
			foundCookie := false
			for _, actualCookie := range response.Cookies() {
				cookieFound, errors := compareCookies(expectedCookie, actualCookie)
				if cookieFound {
					foundCookie = true
					mismatchedFields = append(mismatchedFields, errors...)
				}
			}
			s.verifier.Equal(s.t, true, foundCookie, "ExpectedCookie not found - "+*expectedCookie.name, failureMessageArgs{Name: s.name})
			s.verifier.Equal(s.t, 0, len(mismatchedFields), strings.Join(mismatchedFields, ","), failureMessageArgs{Name: s.name})
		}
	}

	if len(s.response.cookiesPresent) > 0 {
		for _, cookieName := range s.response.cookiesPresent {
			foundCookie := false
			for _, cookie := range response.Cookies() {
				if cookie.Name == cookieName {
					foundCookie = true
				}
			}
			s.verifier.Equal(s.t, true, foundCookie, "ExpectedCookie not found - "+cookieName, failureMessageArgs{Name: s.name})
		}
	}

	if len(s.response.cookiesNotPresent) > 0 {
		for _, cookieName := range s.response.cookiesNotPresent {
			foundCookie := false
			for _, cookie := range response.Cookies() {
				if cookie.Name == cookieName {
					foundCookie = true
				}
			}
			s.verifier.Equal(s.t, false, foundCookie, "ExpectedCookie found - "+cookieName, failureMessageArgs{Name: s.name})
		}
	}
}

func (s *SpecTest) assertHeaders(res *http.Response) {
	for expectedHeader, expectedValues := range s.response.headers {
		resHeaderValues, foundHeader := res.Header[expectedHeader]
		s.verifier.Equal(s.t, true, foundHeader, fmt.Sprintf("expected header '%s' not present in response", expectedHeader), failureMessageArgs{Name: s.name})

		if foundHeader {
			for _, expectedValue := range expectedValues {
				foundValue := false
				for _, resValue := range resHeaderValues {
					if expectedValue == resValue {
						foundValue = true
						break
					}
				}
				s.verifier.Equal(s.t, true, foundValue, fmt.Sprintf("mismatched values for header '%s'. Expected %s but received %s", expectedHeader, expectedValue, strings.Join(resHeaderValues, ",")), failureMessageArgs{Name: s.name})
			}
		}
	}

	if len(s.response.headersPresent) > 0 {
		for _, expectedName := range s.response.headersPresent {
			if res.Header.Get(expectedName) == "" {
				s.verifier.Fail(s.t, fmt.Sprintf("expected header '%s' not present in response", expectedName), failureMessageArgs{Name: s.name})
			}
		}
	}

	if len(s.response.headersNotPresent) > 0 {
		for _, name := range s.response.headersNotPresent {
			if res.Header.Get(name) != "" {
				s.verifier.Fail(s.t, fmt.Sprintf("did not expect header '%s' in response", name), failureMessageArgs{Name: s.name})
			}
		}
	}
}

func (s *SpecTest) buildRequest() *http.Request {
	if s.httpRequest != nil {
		return s.httpRequest
	}

	if len(s.request.formData) > 0 {
		form := url.Values{}
		for k := range s.request.formData {
			for _, value := range s.request.formData[k] {
				form.Add(k, value)
			}
		}
		s.request.Body(form.Encode())
	}

	if s.request.multipart != nil {
		err := s.request.multipart.Close()
		if err != nil {
			s.request.specTest.t.Fatal(err)
		}

		s.request.Header("Content-Type", s.request.multipart.FormDataContentType())
		s.request.Body(s.request.multipartBody.String())
	}

	req, _ := http.NewRequest(s.request.method, s.request.url, bytes.NewBufferString(s.request.body))
	if s.request.context != nil {
		req = req.WithContext(s.request.context)
	}

	req.URL.RawQuery = formatQuery(s.request)
	req.Host = SystemUnderTestDefaultName
	if s.network.isEnable() {
		req.Host = req.URL.Host
	}

	for k, v := range s.request.headers {
		for _, headerValue := range v {
			req.Header.Add(k, headerValue)
		}
	}

	for _, cookie := range s.request.cookies {
		req.AddCookie(cookie.ToHTTPCookie())
	}

	if s.request.basicAuth != "" {
		parts := strings.Split(s.request.basicAuth, ":")
		req.SetBasicAuth(parts[0], parts[1])
	}

	return req
}

func formatQuery(request *Request) string {
	var out url.Values = map[string][]string{}

	if request.queryCollection != nil {
		for _, param := range buildQueryCollection(request.queryCollection) {
			out.Add(param.l, param.r)
		}
	}

	if request.query != nil {
		for k, v := range request.query {
			for _, p := range v {
				out.Add(k, p)
			}
		}
	}

	if len(out) > 0 {
		return out.Encode()
	}
	return ""
}

func buildQueryCollection(params map[string][]string) []pair {
	if len(params) == 0 {
		return []pair{}
	}

	var pairs []pair
	for k, v := range params {
		for _, paramValue := range v {
			pairs = append(pairs, pair{l: k, r: paramValue})
		}
	}
	return pairs
}

func copyHTTPRequest(request *http.Request) *http.Request {
	resCopy := &http.Request{
		Method:        request.Method,
		Host:          request.Host,
		Proto:         request.Proto,
		ProtoMinor:    request.ProtoMinor,
		ProtoMajor:    request.ProtoMajor,
		ContentLength: request.ContentLength,
		RemoteAddr:    request.RemoteAddr,
	}
	resCopy = resCopy.WithContext(request.Context())

	if request.Body != nil {
		bodyBytes, _ := io.ReadAll(request.Body)
		resCopy.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	if request.URL != nil {
		r2URL := new(url.URL)
		*r2URL = *request.URL
		resCopy.URL = r2URL
	}

	headers := make(http.Header)
	for k, values := range request.Header {
		for _, hValue := range values {
			headers.Add(k, hValue)
		}
	}
	resCopy.Header = headers

	return resCopy
}
