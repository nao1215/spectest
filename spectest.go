// Package spectest is simple and extensible behavioral testing library for Go. You can use api test to simplify REST API, HTTP handler and e2e tests. (forked from steinfletcher/apitest)
package spectest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"runtime/debug"
	"sort"
	"strings"
	"time"
)

// APITest is the top level struct holding the test spec
type APITest struct {
	// debugEnabled will log the http wire representation of all http interactions
	debugEnabled bool
	// mockResponseDelayEnabled will turn on mock response delays (defaults to OFF)
	mockResponseDelayEnabled bool
	// networkingEnabled will enable networking for provided clients
	networkingEnabled bool
	// networkingHTTPClient is the http client used when networking is enabled
	networkingHTTPClient *http.Client
	// reporter is the report formatter.
	reporter ReportFormatter
	// verifier is the assertion implementation
	verifier       Verifier
	recorder       *Recorder
	handler        http.Handler
	name           string
	request        *Request
	response       *Response
	observers      []Observe
	mocksObservers []Observe
	recorderHook   RecorderHook
	mocks          []*Mock
	t              TestingT
	httpClient     *http.Client
	httpRequest    *http.Request
	transport      *Transport
	// meta is the meta data for the test report.
	meta     *Meta
	started  time.Time
	finished time.Time
}

// Observe will be called by with the request and response on completion
type Observe func(*http.Response, *http.Request, *APITest)

// RecorderHook used to implement a custom interaction recorder
type RecorderHook func(*Recorder)

// New creates a new api test. The name is optional and will appear in test reports
func New(name ...string) *APITest {
	apiTest := &APITest{
		meta: newMeta(),
	}

	request := &Request{
		apiTest:  apiTest,
		headers:  map[string][]string{},
		query:    map[string][]string{},
		formData: map[string][]string{},
	}
	response := &Response{
		apiTest: apiTest,
		headers: map[string][]string{},
	}
	apiTest.request = request
	apiTest.response = response

	if len(name) > 0 {
		apiTest.name = name[0]
	}
	return apiTest
}

// Handler is a convenience method for creating a new spectest with a handler
func Handler(handler http.Handler) *APITest {
	return New().Handler(handler)
}

// HandlerFunc is a convenience method for creating a new spectest with a handler func
func HandlerFunc(handlerFunc http.HandlerFunc) *APITest {
	return New().HandlerFunc(handlerFunc)
}

// EnableNetworking will enable networking for provided clients
func (a *APITest) EnableNetworking(clients ...*http.Client) *APITest {
	a.networkingEnabled = true
	if len(clients) == 1 {
		a.networkingHTTPClient = clients[0]
		return a
	}
	a.networkingHTTPClient = http.DefaultClient
	return a
}

// EnableMockResponseDelay turns on mock response delays (defaults to OFF)
func (a *APITest) EnableMockResponseDelay() *APITest {
	a.mockResponseDelayEnabled = true
	return a
}

// Debug logs to the console the http wire representation of all http interactions
// that are intercepted by spectest. This includes the inbound request to the application
// under test, the response returned by the application and any interactions that are
// intercepted by the mock server.
func (a *APITest) Debug() *APITest {
	a.debugEnabled = true
	return a
}

// Report provides a hook to add custom formatting to the output of the test
func (a *APITest) Report(reporter ReportFormatter) *APITest {
	a.reporter = reporter
	return a
}

// Recorder provides a hook to add a recorder to the test
func (a *APITest) Recorder(recorder *Recorder) *APITest {
	a.recorder = recorder
	return a
}

// CustomHost set hostname.
// This method is not change the host in the request. It is only for the report.
func (a *APITest) CustomHost(host string) *APITest {
	a.meta.Host = host
	return a
}

// CustomReportName allows the consumer to override the default report file name.
func (a *APITest) CustomReportName(name string) *APITest {
	a.meta.ReportFileName = name
	return a
}

// Handler defines the http handler that is invoked when the test is run
func (a *APITest) Handler(handler http.Handler) *APITest {
	a.handler = handler
	return a
}

// HandlerFunc defines the http handler that is invoked when the test is run
func (a *APITest) HandlerFunc(handlerFunc http.HandlerFunc) *APITest {
	a.handler = handlerFunc
	return a
}

// Mocks is a builder method for setting the mocks
func (a *APITest) Mocks(mocks ...*Mock) *APITest {
	var m []*Mock
	for i := range mocks {
		times := mocks[i].response.mock.times
		for j := 1; j <= times; j++ {
			mockCopy := mocks[i].copy()
			mockCopy.times = 1
			m = append(m, mockCopy)
		}
	}
	a.mocks = m
	return a
}

// HTTPClient allows the developer to provide a custom http client when using mocks
func (a *APITest) HTTPClient(client *http.Client) *APITest {
	a.httpClient = client
	return a
}

// Observe is a builder method for setting the observers
func (a *APITest) Observe(observers ...Observe) *APITest {
	a.observers = observers
	return a
}

// ObserveMocks is a builder method for setting the mocks observers
func (a *APITest) ObserveMocks(observer Observe) *APITest {
	a.mocksObservers = append(a.mocksObservers, observer)
	return a
}

// RecorderHook allows the consumer to provider a function that will receive the recorder instance before the
// test runs. This can be used to inject custom events which can then be rendered in diagrams
// Deprecated: use Recorder() instead
func (a *APITest) RecorderHook(hook RecorderHook) *APITest {
	a.recorderHook = hook
	return a
}

// Request returns the request spec
func (a *APITest) Request() *Request {
	return a.request
}

// Response returns the expected response
func (a *APITest) Response() *Response {
	return a.response
}

// Intercept will be called before the request is made.
// Updates to the request will be reflected in the test
type Intercept func(*http.Request)

type pair struct {
	l string
	r string
}

// Intercept is a builder method for setting the request interceptor
func (a *APITest) Intercept(interceptor Intercept) *APITest {
	a.request.interceptor = interceptor
	return a
}

// Verifier allows consumers to override the verification implementation.
func (a *APITest) Verifier(v Verifier) *APITest {
	a.verifier = v
	return a
}

// Method is a builder method for setting the http method of the request
func (a *APITest) Method(method string) *Request {
	a.request.method = method
	return a.request
}

// HTTPRequest defines the native `http.Request`
func (a *APITest) HTTPRequest(req *http.Request) *Request {
	a.httpRequest = req
	return a.request
}

// Get is a convenience method for setting the request as http.MethodGet
func (a *APITest) Get(url string) *Request {
	a.request.method = http.MethodGet
	a.request.url = url
	return a.request
}

// Getf is a convenience method that adds formatting support to Get
func (a *APITest) Getf(format string, args ...interface{}) *Request {
	return a.Get(fmt.Sprintf(format, args...))
}

// Post is a convenience method for setting the request as http.MethodPost
func (a *APITest) Post(url string) *Request {
	r := a.request
	r.method = http.MethodPost
	r.url = url
	return r
}

// Postf is a convenience method that adds formatting support to Post
func (a *APITest) Postf(format string, args ...interface{}) *Request {
	return a.Post(fmt.Sprintf(format, args...))
}

// Put is a convenience method for setting the request as http.MethodPut
func (a *APITest) Put(url string) *Request {
	r := a.request
	r.method = http.MethodPut
	r.url = url
	return r
}

// Putf is a convenience method that adds formatting support to Put
func (a *APITest) Putf(format string, args ...interface{}) *Request {
	return a.Put(fmt.Sprintf(format, args...))
}

// Delete is a convenience method for setting the request as http.MethodDelete
func (a *APITest) Delete(url string) *Request {
	a.request.method = http.MethodDelete
	a.request.url = url
	return a.request
}

// Deletef is a convenience method that adds formatting support to Delete
func (a *APITest) Deletef(format string, args ...interface{}) *Request {
	return a.Delete(fmt.Sprintf(format, args...))
}

// Patch is a convenience method for setting the request as http.MethodPatch
func (a *APITest) Patch(url string) *Request {
	a.request.method = http.MethodPatch
	a.request.url = url
	return a.request
}

// Patchf is a convenience method that adds formatting support to Patch
func (a *APITest) Patchf(format string, args ...interface{}) *Request {
	return a.Patch(fmt.Sprintf(format, args...))
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

func (a *APITest) report() *http.Response {
	var capturedInboundReq *http.Request
	var capturedFinalRes *http.Response
	var capturedMockInteractions []*mockInteraction

	a.observers = append(a.observers, func(finalRes *http.Response, inboundReq *http.Request, a *APITest) {
		capturedFinalRes = copyHTTPResponse(finalRes)
		defer func() {
			capturedFinalRes.Body.Close() //nolint errcheck TODO:
		}()
		capturedInboundReq = copyHTTPRequest(inboundReq)
	})

	a.mocksObservers = append(a.mocksObservers, func(mockRes *http.Response, mockReq *http.Request, a *APITest) {
		capturedMockInteractions = append(capturedMockInteractions, &mockInteraction{
			request:   copyHTTPRequest(mockReq),
			response:  copyHTTPResponse(mockRes),
			timestamp: time.Now().UTC(),
		})
	})

	if a.recorder == nil {
		a.recorder = NewTestRecorder()
	}
	defer a.recorder.Reset()

	if a.recorderHook != nil {
		a.recorderHook(a.recorder)
	}

	a.started = time.Now()
	res := a.response.runTest()
	a.finished = time.Now()

	a.recorder.
		AddTitle(fmt.Sprintf("%s %s", capturedInboundReq.Method, capturedInboundReq.URL.String())).
		AddSubTitle(a.name).
		AddHTTPRequest(HTTPRequest{
			Source:    ConsumerDefaultName,
			Target:    SystemUnderTestDefaultName,
			Value:     capturedInboundReq,
			Timestamp: a.started,
		})

	for _, interaction := range capturedMockInteractions {
		a.recorder.AddHTTPRequest(HTTPRequest{
			Source:    SystemUnderTestDefaultName,
			Target:    interaction.GetRequestHost(),
			Value:     interaction.request,
			Timestamp: interaction.timestamp,
		})
		if interaction.response != nil {
			a.recorder.AddHTTPResponse(HTTPResponse{
				Source:    interaction.GetRequestHost(),
				Target:    SystemUnderTestDefaultName,
				Value:     interaction.response,
				Timestamp: interaction.timestamp,
			})
		}
	}

	a.recorder.AddHTTPResponse(HTTPResponse{
		Source:    SystemUnderTestDefaultName,
		Target:    ConsumerDefaultName,
		Value:     capturedFinalRes,
		Timestamp: a.finished,
	})

	sort.Slice(a.recorder.Events, func(i, j int) bool {
		return a.recorder.Events[i].GetTime().Before(a.recorder.Events[j].GetTime())
	})

	meta := newMeta()
	meta.StatusCode = capturedFinalRes.StatusCode
	meta.Path = capturedInboundReq.URL.String()
	meta.Method = capturedInboundReq.Method
	meta.Duration = a.finished.Sub(a.started).Nanoseconds()
	meta.Name = a.name
	meta.ReportFileName = a.meta.ReportFileName
	if a.meta.Host != "" {
		meta.Host = a.meta.Host
	}

	a.recorder.AddMeta(meta)
	a.reporter.Format(a.recorder)

	return res
}

func (a *APITest) assertMocks() {
	for _, mock := range a.mocks {
		if !mock.isUsed && mock.timesSet {
			a.verifier.Fail(a.t, "mock was not invoked expected times", failureMessageArgs{Name: a.name})
		}
	}
}

func (a *APITest) assertFunc(res *http.Response, req *http.Request) {
	if len(a.response.assert) > 0 {
		for _, assertFn := range a.response.assert {
			err := assertFn(copyHTTPResponse(res), copyHTTPRequest(req))
			if err != nil {
				a.verifier.NoError(a.t, err, failureMessageArgs{Name: a.name})
			}
		}
	}
}

func (a *APITest) doRequest() (*http.Response, *http.Request) {
	req := a.buildRequest()
	if a.request.interceptor != nil {
		a.request.interceptor(req)
	}
	resRecorder := httptest.NewRecorder()

	if a.debugEnabled {
		requestDump, err := httputil.DumpRequest(req, true)
		if err == nil {
			debugLog(requestDebugPrefix(), "inbound http request", string(requestDump))
		}
	}

	var res *http.Response
	var err error
	if !a.networkingEnabled {
		a.serveHTTP(resRecorder, copyHTTPRequest(req))
		res = resRecorder.Result()
	} else {
		res, err = a.networkingHTTPClient.Do(copyHTTPRequest(req))
		if err != nil {
			a.t.Fatal(err)
		}
	}

	if a.debugEnabled {
		responseDump, err := httputil.DumpResponse(res, true)
		if err == nil {
			debugLog(responseDebugPrefix(), "final response", string(responseDump))
		}
	}

	return res, req
}

func (a *APITest) serveHTTP(res *httptest.ResponseRecorder, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			a.t.Fatalf("%s: %s", err, debug.Stack())
		}
	}()

	a.handler.ServeHTTP(res, req)
}

func (a *APITest) assertResponse(res *http.Response) {
	if a.response.status != 0 {
		a.verifier.Equal(a.t, a.response.status, res.StatusCode, fmt.Sprintf("Status code %d not equal to %d", res.StatusCode, a.response.status), failureMessageArgs{Name: a.name})
	}

	if a.response.body != "" {
		var resBodyBytes []byte
		if res.Body != nil {
			resBodyBytes, _ = io.ReadAll(res.Body)
			res.Body = io.NopCloser(bytes.NewBuffer(resBodyBytes))
		}
		if json.Valid([]byte(a.response.body)) {
			a.verifier.JSONEq(a.t, a.response.body, string(resBodyBytes), failureMessageArgs{Name: a.name})
		} else {
			a.verifier.Equal(a.t, a.response.body, string(resBodyBytes), failureMessageArgs{Name: a.name})
		}
	}
}

func (a *APITest) assertCookies(response *http.Response) {
	if len(a.response.cookies) > 0 {
		for _, expectedCookie := range a.response.cookies {
			var mismatchedFields []string
			foundCookie := false
			for _, actualCookie := range response.Cookies() {
				cookieFound, errors := compareCookies(expectedCookie, actualCookie)
				if cookieFound {
					foundCookie = true
					mismatchedFields = append(mismatchedFields, errors...)
				}
			}
			a.verifier.Equal(a.t, true, foundCookie, "ExpectedCookie not found - "+*expectedCookie.name, failureMessageArgs{Name: a.name})
			a.verifier.Equal(a.t, 0, len(mismatchedFields), strings.Join(mismatchedFields, ","), failureMessageArgs{Name: a.name})
		}
	}

	if len(a.response.cookiesPresent) > 0 {
		for _, cookieName := range a.response.cookiesPresent {
			foundCookie := false
			for _, cookie := range response.Cookies() {
				if cookie.Name == cookieName {
					foundCookie = true
				}
			}
			a.verifier.Equal(a.t, true, foundCookie, "ExpectedCookie not found - "+cookieName, failureMessageArgs{Name: a.name})
		}
	}

	if len(a.response.cookiesNotPresent) > 0 {
		for _, cookieName := range a.response.cookiesNotPresent {
			foundCookie := false
			for _, cookie := range response.Cookies() {
				if cookie.Name == cookieName {
					foundCookie = true
				}
			}
			a.verifier.Equal(a.t, false, foundCookie, "ExpectedCookie found - "+cookieName, failureMessageArgs{Name: a.name})
		}
	}
}

func (a *APITest) assertHeaders(res *http.Response) {
	for expectedHeader, expectedValues := range a.response.headers {
		resHeaderValues, foundHeader := res.Header[expectedHeader]
		a.verifier.Equal(a.t, true, foundHeader, fmt.Sprintf("expected header '%s' not present in response", expectedHeader), failureMessageArgs{Name: a.name})

		if foundHeader {
			for _, expectedValue := range expectedValues {
				foundValue := false
				for _, resValue := range resHeaderValues {
					if expectedValue == resValue {
						foundValue = true
						break
					}
				}
				a.verifier.Equal(a.t, true, foundValue, fmt.Sprintf("mismatched values for header '%s'. Expected %s but received %s", expectedHeader, expectedValue, strings.Join(resHeaderValues, ",")), failureMessageArgs{Name: a.name})
			}
		}
	}

	if len(a.response.headersPresent) > 0 {
		for _, expectedName := range a.response.headersPresent {
			if res.Header.Get(expectedName) == "" {
				a.verifier.Fail(a.t, fmt.Sprintf("expected header '%s' not present in response", expectedName), failureMessageArgs{Name: a.name})
			}
		}
	}

	if len(a.response.headersNotPresent) > 0 {
		for _, name := range a.response.headersNotPresent {
			if res.Header.Get(name) != "" {
				a.verifier.Fail(a.t, fmt.Sprintf("did not expect header '%s' in response", name), failureMessageArgs{Name: a.name})
			}
		}
	}
}

func (a *APITest) buildRequest() *http.Request {
	if a.httpRequest != nil {
		return a.httpRequest
	}

	if len(a.request.formData) > 0 {
		form := url.Values{}
		for k := range a.request.formData {
			for _, value := range a.request.formData[k] {
				form.Add(k, value)
			}
		}
		a.request.Body(form.Encode())
	}

	if a.request.multipart != nil {
		err := a.request.multipart.Close()
		if err != nil {
			a.request.apiTest.t.Fatal(err)
		}

		a.request.Header("Content-Type", a.request.multipart.FormDataContentType())
		a.request.Body(a.request.multipartBody.String())
	}

	req, _ := http.NewRequest(a.request.method, a.request.url, bytes.NewBufferString(a.request.body))
	if a.request.context != nil {
		req = req.WithContext(a.request.context)
	}

	req.URL.RawQuery = formatQuery(a.request)
	req.Host = SystemUnderTestDefaultName
	if a.networkingEnabled {
		req.Host = req.URL.Host
	}

	for k, v := range a.request.headers {
		for _, headerValue := range v {
			req.Header.Add(k, headerValue)
		}
	}

	for _, cookie := range a.request.cookies {
		req.AddCookie(cookie.ToHTTPCookie())
	}

	if a.request.basicAuth != "" {
		parts := strings.Split(a.request.basicAuth, ":")
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
