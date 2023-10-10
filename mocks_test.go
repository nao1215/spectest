package spectest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestMocksCookieMatches(t *testing.T) {
	reqURL := "http://test.com/v1/path"
	req := httptest.NewRequest(http.MethodGet, reqURL, nil)
	req.Header.Set("Cookie", "k=v")
	mockRequest := NewMock().Get(reqURL).Cookie("k", "v")

	matchError := cookieMatcher(req, mockRequest)

	assert.NoError(t, matchError)
}

func TestMocksCookieNameFailsToMatch(t *testing.T) {
	reqURL := "http://test.com/v1/path"
	req := httptest.NewRequest(http.MethodGet, reqURL, nil)
	req.Header.Set("Cookie", "a=c")
	mockRequest := NewMock().Get(reqURL).Cookie("x", "y")

	matchError := cookieMatcher(req, mockRequest)

	assert.Equal(t, matchError.Error(),
		"expected cookie with name 'x' not received")
}

func TestMocksCookieValueFailsToMatch(t *testing.T) {
	reqURL := "http://test.com/v1/path"
	req := httptest.NewRequest(http.MethodGet, reqURL, nil)
	req.Header.Set("Cookie", "a=c")
	mockRequest := NewMock().Get(reqURL).Cookie("a", "v")

	matchError := cookieMatcher(req, mockRequest)

	assert.Equal(t, matchError.Error(),
		"failed to match cookie: [Mismatched field Value. Expected v but received c]")
}

func TestMocksCookiePresentMatches(t *testing.T) {
	reqURL := "http://test.com/v1/path"
	req := httptest.NewRequest(http.MethodGet, reqURL, nil)
	req.Header.Set("Cookie", "k=v")
	mockRequest := NewMock().Get(reqURL).CookiePresent("k")

	matchError := cookiePresentMatcher(req, mockRequest)

	assert.NoError(t, matchError)
}

func TestMocksCookiePresentFailsToMatch(t *testing.T) {
	reqURL := "http://test.com/v1/path"
	req := httptest.NewRequest(http.MethodGet, reqURL, nil)
	req.Header.Set("Cookie", "k=v")
	mockRequest := NewMock().Get(reqURL).CookiePresent("a")

	matchError := cookiePresentMatcher(req, mockRequest)

	assert.Equal(t, matchError.Error(), "expected cookie with name 'a' not received")
}

func TestMocksCookieNotPresentMatches(t *testing.T) {
	reqURL := "http://test.com/v1/path"
	req := httptest.NewRequest(http.MethodGet, reqURL, nil)
	req.Header.Set("Cookie", "k=v")
	mockRequest := NewMock().Get(reqURL).CookieNotPresent("a")

	matchError := cookieNotPresentMatcher(req, mockRequest)

	assert.NoError(t, matchError)
}

func TestMocksCookieNotPresentFailsToMatch(t *testing.T) {
	reqURL := "http://test.com/v1/path"
	req := httptest.NewRequest(http.MethodGet, reqURL, nil)
	req.Header.Set("Cookie", "k=v")
	mockRequest := NewMock().Get(reqURL).CookieNotPresent("k")

	matchError := cookieNotPresentMatcher(req, mockRequest)

	assert.Equal(t, matchError.Error(), "did not expect a cookie with name 'k'")
}

func TestMocksNewUnmatchedMockErrorEmpty(t *testing.T) {
	mockError := newUnmatchedMockError()

	assert.Equal(t, true, mockError != nil)
	assert.Equal(t, 0, len(mockError.errors))
}

func TestMocksNewEmptyUnmatchedMockErrorExpectedErrorsString(t *testing.T) {
	mockError := newUnmatchedMockError().
		addErrors(1, errors.New("a boo boo has occurred")).
		addErrors(2, errors.New("tom drank too much beer"))

	assert.Equal(t, true, mockError != nil)
	assert.Equal(t, 2, len(mockError.errors))
	assert.Equal(t,
		"received request did not match any mocks\n\nMock 1 mismatches:\n• a boo boo has occurred\n\nMock 2 mismatches:\n• tom drank too much beer\n\n",
		mockError.Error())
}

func TestMocksHostMatcher(t *testing.T) {
	tests := map[string]struct {
		request       *http.Request
		mockURL       string
		expectedError error
	}{
		"matching": {
			request:       httptest.NewRequest(http.MethodGet, "http://test.com", nil),
			mockURL:       "https://test.com",
			expectedError: nil,
		},
		"not matching": {
			request:       httptest.NewRequest(http.MethodGet, "https://test.com", nil),
			mockURL:       "https://testa.com",
			expectedError: errors.New("received host test.com did not match mock host testa.com"),
		},
		"no expected host": {
			request:       httptest.NewRequest(http.MethodGet, "https://test.com", nil),
			mockURL:       "",
			expectedError: nil,
		},
		"matching using URL host": {
			request: &http.Request{URL: &url.URL{
				Host: "test.com",
				Path: "/",
			}},
			mockURL:       "https://test.com",
			expectedError: nil,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			matchError := hostMatcher(test.request, NewMock().Get(test.mockURL))
			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocksHeaderMatcher(t *testing.T) {
	tests := []struct {
		requestHeaders     map[string]string
		headerToMatchKey   string
		headerToMatchValue string
		expectedError      error
	}{
		{map[string]string{"B": "5", "A": "123"}, "A", "123", nil},
		{map[string]string{"A": "123"}, "C", "3", errors.New("not all of received headers map[A:[123]] matched expected mock headers map[C:[3]]")},
		{map[string]string{}, "", "", nil},
		{map[string]string{"A": "apple"}, "A", "a([a-z]+)ple", nil},
		{map[string]string{"A": "apple"}, "A", "a-z]+)ch_invalid_regexp", errors.New("failed to parse regexp for header A with value a-z]+)ch_invalid_regexp")},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s %s", test.headerToMatchKey, test.headerToMatchValue), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/assert", nil)
			for k, v := range test.requestHeaders {
				req.Header.Set(k, v)
			}
			mockRequest := NewMock().Get("/assert")
			if test.headerToMatchKey != "" {
				mockRequest.Header(test.headerToMatchKey, test.headerToMatchValue)
			}
			matchError := headerMatcher(req, mockRequest)
			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocksMockRequestHeaderWorksWithHeaders(t *testing.T) {
	mock := NewMock().
		Get("/path").
		Header("A", "12345").
		Headers(map[string]string{"B": "67890"})
	req := httptest.NewRequest(http.MethodGet, "/path", nil)
	req.Header.Set("A", "12345")
	req.Header.Set("B", "67890")

	matchError := headerMatcher(req, mock)

	assert.Equal(t, true, matchError == nil)
}

func TestMocksHeaderPresentMatcher(t *testing.T) {
	tests := map[string]struct {
		requestHeaders map[string]string
		headerPresent  string
		expectedError  error
	}{
		"present":     {map[string]string{"A": "123", "X": "456"}, "X", nil},
		"not present": {map[string]string{"A": "123"}, "C", errors.New("expected header 'C' was not present")},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/assert", nil)
			for k, v := range test.requestHeaders {
				req.Header.Add(k, v)
			}
			mockRequest := NewMock().Get("/assert").HeaderPresent(test.headerPresent)

			matchError := headerPresentMatcher(req, mockRequest)

			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocksHeaderNotPresentMatcher(t *testing.T) {
	tests := map[string]struct {
		requestHeaders   map[string]string
		headerNotPresent string
		expectedError    error
	}{
		"not present": {map[string]string{"A": "123"}, "C", nil},
		"present":     {map[string]string{"A": "123", "X": "456"}, "X", errors.New("unexpected header 'X' was present")},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/assert", nil)
			for k, v := range test.requestHeaders {
				req.Header.Add(k, v)
			}
			mockRequest := NewMock().Get("/assert").HeaderNotPresent(test.headerNotPresent)

			matchError := headerNotPresentMatcher(req, mockRequest)

			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocksBasicAuth(t *testing.T) {
	tests := map[string]struct {
		reqUsername   string
		reqPassword   string
		mockUsername  string
		mockPassword  string
		expectedError error
	}{
		"matches": {
			reqUsername:   "myUser",
			reqPassword:   "myPassword",
			mockUsername:  "myUser",
			mockPassword:  "myPassword",
			expectedError: nil,
		},
		"not matches username": {
			reqUsername:   "notMyUser",
			reqPassword:   "myPassword",
			mockUsername:  "myUser",
			mockPassword:  "myPassword",
			expectedError: errors.New("basic auth request username 'notMyUser' did not match mock username 'myUser'"),
		},
		"not matches password": {
			reqUsername:   "myUser",
			reqPassword:   "notMyPassword",
			mockUsername:  "myUser",
			mockPassword:  "myPassword",
			expectedError: errors.New("basic auth request password 'notMyPassword' did not match mock password 'myPassword'"),
		},
		"not matches if no auth header": {
			mockUsername:  "myUser",
			mockPassword:  "myPassword",
			expectedError: errors.New("request did not contain valid HTTP Basic Authentication string"),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if test.reqUsername != "" {
				req.SetBasicAuth(test.reqUsername, test.reqPassword)
			}

			mockRequest := NewMock().Get("/").BasicAuth(test.mockUsername, test.mockPassword)

			matchError := basicAuthMatcher(req, mockRequest)

			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocksQueryMatcherSuccess(t *testing.T) {
	tests := []struct {
		requestURL   string
		queryToMatch map[string][]string
	}{
		{"http://test.com/v1/path?a=1", map[string][]string{"a": {"1"}}},
		{"http://test.com/v2/path?b=2&a=1", map[string][]string{"b": {"2"}, "a": {"1"}}},
		{"http://test.com/v2/path?b=2&a=1&a=2", map[string][]string{"a": {"2"}}},
		{"http://test.com/v2/path?b=2&a=1&a=2", map[string][]string{"a": {"2", "1"}}},
		{"http://test.com/v2/path?b=2&a=apple", map[string][]string{"a": {"a([a-z]+)ple"}}},
	}
	for _, test := range tests {
		t.Run(test.requestURL, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.requestURL, nil)
			mockRequest := NewMock().Get(test.requestURL)
			for k := range test.queryToMatch {
				for _, value := range test.queryToMatch[k] {
					mockRequest.Query(k, value)
				}
			}

			matchError := queryParamMatcher(req, mockRequest)
			assert.NoError(t, matchError)
		})
	}
}

func TestMocksQueryMatcherErrors(t *testing.T) {
	tests := []struct {
		requestURL    string
		queryToMatch  map[string][]string
		expectedError error
	}{
		{"http://test.com/v1/path", map[string][]string{"a": {"1"}}, errors.New("not all of received query params map[] matched expected mock query params map[a:[1]]")},
		{"http://test.com/v2/path?a=1", map[string][]string{"b": {"1"}}, errors.New("not all of received query params map[a:[1]] matched expected mock query params map[b:[1]]")},
		{"http://test.com/v2/path?b=2&a=1&a=2&a=3", map[string][]string{"a": {"4", "1", "2"}}, errors.New("b:[2]")},
		{"http://test.com/v2/path?b=2&a=1&a=2&a=3", map[string][]string{"a": {"4", "1", "2"}}, errors.New("a:[1 2 3]")},
		{"http://test.com/v2/path?b=2&a=1", map[string][]string{"a": {"a-z]+)ch_invalid_regexp"}}, errors.New("failed to parse regexp for query param a with value a-z]+)ch_invalid_regexp")},
	}
	for _, test := range tests {
		t.Run(test.requestURL, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.requestURL, nil)
			mockRequest := NewMock().Get(test.requestURL)
			for k := range test.queryToMatch {
				for _, value := range test.queryToMatch[k] {
					mockRequest.Query(k, value)
				}
			}

			matchError := queryParamMatcher(req, mockRequest)
			assert.Equal(t, true, strings.Contains(matchError.Error(), test.expectedError.Error()))
		})
	}
}

func TestMocksQueryParamsDoesNotOverwriteQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://test.com/v2/path?b=2&a=1", nil)
	mockRequest := NewMock().
		Get("http://test.com").
		Query("b", "2").
		QueryParams(map[string]string{"a": "1"})

	matchError := queryParamMatcher(req, mockRequest)

	assert.Equal(t, 2, len(mockRequest.query))
	assert.Equal(t, true, matchError == nil)
}

func TestMocksQueryCollection(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://test.com/v2/path?a=1&a=2&b=3&c=4", nil)
	mockRequest := NewMock().
		Get("http://test.com").
		Query("b", "3").
		QueryParams(map[string]string{"c": "4"}).
		QueryCollection(map[string][]string{"a": {"1", "2"}})

	matchError := queryParamMatcher(req, mockRequest)

	assert.Equal(t, 3, len(mockRequest.query))
	assert.Equal(t, true, matchError == nil)
}

func TestMocksQueryCollectionFails(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://test.com/v2/path?a=1&a=2&b=3&c=4", nil)
	mockRequest := NewMock().
		Get("http://test.com").
		Query("b", "3").
		QueryParams(map[string]string{"c": "4"}).
		QueryCollection(map[string][]string{"a": {"1", "2", "3"}})

	matchError := queryParamMatcher(req, mockRequest)

	assert.Equal(t, 3, len(mockRequest.query))
	assert.Equal(t, true, matchError != nil)
}

func TestMocksQueryPresent(t *testing.T) {
	tests := []struct {
		requestURL    string
		queryParam    string
		expectedError error
	}{
		{"http://test.com/v1/path?a=1", "a", nil},
		{"http://test.com/v1/path", "a", errors.New("expected query param a not received")},
		{"http://test.com/v1/path?c=1", "b", errors.New("expected query param b not received")},
		{"http://test.com/v2/path?b=2&a=1", "a", nil},
	}
	for _, test := range tests {
		t.Run(test.requestURL, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.requestURL, nil)
			mockRequest := NewMock().Get(test.requestURL).QueryPresent(test.queryParam)
			matchError := queryPresentMatcher(req, mockRequest)
			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocksQueryNotPresent(t *testing.T) {
	tests := []struct {
		queryString   string
		queryParam    string
		expectedError error
	}{
		{"http://test.com/v1/path?a=1", "a", errors.New("unexpected query param 'a' present")},
		{"http://test.com/v1/path", "a", nil},
		{"http://test.com/v1/path?c=1", "b", nil},
		{"http://test.com/v2/path?b=2&a=1", "a", errors.New("unexpected query param 'a' present")},
	}
	for _, test := range tests {
		t.Run(test.queryString, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.queryString, nil)
			mockRequest := NewMock().Get("http://test.com/v1/path" + test.queryString).QueryNotPresent(test.queryParam)
			matchError := queryNotPresentMatcher(req, mockRequest)
			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocksFormDataMatcher(t *testing.T) {
	tests := []struct {
		name             string
		requestFormData  map[string][]string
		expectedFormData map[string][]string
		expectedError    error
	}{
		{
			"single key match",
			map[string][]string{"a": {"1"}},
			map[string][]string{"a": {"1"}},
			nil,
		},
		{
			"single key match with regular expression",
			map[string][]string{"a": {"apple"}},
			map[string][]string{"a": {"a([a-z]+)ple"}},
			nil,
		},
		{
			"multiple key match",
			map[string][]string{"a": {"1"}, "b": {"1"}},
			map[string][]string{"a": {"1"}, "b": {"1"}},
			nil,
		},
		{
			"multiple value same key match",
			map[string][]string{"a": {"1", "2"}},
			map[string][]string{"a": {"2", "1"}},
			nil,
		},
		{
			"error when no form data present",
			map[string][]string{},
			map[string][]string{"a": {"1"}},
			errors.New("not all of received form data values map[] matched expected mock form data values map[a:[1]]"),
		},
		{
			"error when form data value does not match",
			map[string][]string{"a": {"1"}},
			map[string][]string{"a": {"2"}},
			errors.New("not all of received form data values map[a:[1]] matched expected mock form data values map[a:[2]]"),
		},
		{
			"error when form data key does not match",
			map[string][]string{"a": {"1"}},
			map[string][]string{"b": {"1"}},
			errors.New("not all of received form data values map[a:[1]] matched expected mock form data values map[b:[1]]"),
		},
		{
			"error when form data same key multiple values do not match",
			map[string][]string{"a": {"1", "2", "4"}},
			map[string][]string{"a": {"1", "3", "4"}},
			errors.New("not all of received form data values map[a:[1 2 4]] matched expected mock form data values map[a:[1 3 4]]"),
		},
		{
			"error when regular expression provided is invalid",
			map[string][]string{"a": {"1"}},
			map[string][]string{"a": {"a-z]+)ch_invalid_regexp"}},
			errors.New("failed to parse regexp for form data a with value a-z]+)ch_invalid_regexp"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			form := url.Values{}
			for key := range test.requestFormData {
				for _, value := range test.requestFormData[key] {
					form.Add(key, value)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "http://test.com/v1/path", strings.NewReader(form.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			mockRequest := NewMock().Post("http://test.com/v1/path")
			for key := range test.expectedFormData {
				for _, value := range test.expectedFormData[key] {
					mockRequest.FormData(key, value)
				}
			}
			matchError := formDataMatcher(req, mockRequest)
			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocksFormDataPresent(t *testing.T) {
	tests := []struct {
		name                       string
		requestFormData            map[string]string
		expectedFormDataKeyPresent []string
		expectedError              error
	}{
		{"single form data key present", map[string]string{"a": "1", "b": "1"}, []string{"a"}, nil},
		{"multiple form data key present", map[string]string{"a": "1", "b": "1"}, []string{"a", "b"}, nil},
		{"error when no form data present", map[string]string{}, []string{"a"}, errors.New("expected form data key a not received")},
		{"error when form data key not found", map[string]string{"b": "1", "c": "1"}, []string{"a"}, errors.New("expected form data key a not received")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			form := url.Values{}
			for i := range test.requestFormData {
				form.Add(i, test.requestFormData[i])
			}

			req := httptest.NewRequest(http.MethodPost, "http://test.com/v1/path", strings.NewReader(form.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			mockRequest := NewMock().Post("http://test.com/v1/path")
			for _, key := range test.expectedFormDataKeyPresent {
				mockRequest.FormDataPresent(key)
			}

			matchError := formDataPresentMatcher(req, mockRequest)

			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocksFormDataNotPresent(t *testing.T) {
	tests := []struct {
		name                          string
		requestFormData               map[string]string
		expectedFormDataKeyNotPresent []string
		expectedError                 error
	}{
		{"single form data key not present", map[string]string{"a": "1", "b": "1"}, []string{"c"}, nil},
		{"multiple form data key not present", map[string]string{"a": "1", "b": "1"}, []string{"d", "e"}, nil},
		{"no form data present", map[string]string{}, []string{"a"}, nil},
		{"error when form data key found", map[string]string{"a": "1", "b": "1"}, []string{"a"}, errors.New("did not expect a form data key a")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			form := url.Values{}
			for i := range test.requestFormData {
				form.Add(i, test.requestFormData[i])
			}

			req := httptest.NewRequest(http.MethodPost, "http://test.com/v1/path", strings.NewReader(form.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			mockRequest := NewMock().Post("http://test.com/v1/path")
			for _, key := range test.expectedFormDataKeyNotPresent {
				mockRequest.FormDataNotPresent(key)
			}

			matchError := formDataNotPresentMatcher(req, mockRequest)

			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocksSchemeMatcher(t *testing.T) {
	tests := []struct {
		requestURL    string
		mockURL       string
		expectedError error
	}{
		{"http://test.com", "https://test.com", errors.New("received scheme http did not match mock scheme https")},
		{"https://test.com", "https://test.com", nil},
		{"https://test.com", "test.com", nil},
		{"localhost:80", "localhost:80", nil},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s %s", test.requestURL, test.mockURL), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.requestURL, nil)
			matchError := schemeMatcher(req, NewMock().Get(test.mockURL))
			if !reflect.DeepEqual(matchError, test.expectedError) {
				t.Fatalf("mockUrl='%s' requestUrl='%s' actual=%v shouldMatch=%v",
					test.mockURL, test.requestURL, matchError, test.expectedError)
			}
		})
	}
}

func TestMocksBodyMatcher(t *testing.T) {
	tests := []struct {
		requestBody   string
		matchBody     string
		expectedError error
	}{
		{`{"a": 1}`, "", nil},
		{``, `{"a":1}`, errors.New("expected a body but received none")},
		{`{"x":"12345"}`, `{"x":"12345"}`, nil},
		{`{"a": 12345, "b": [{"key": "c", "value": "result"}]}`,
			`{"b":[{"key":"c","value":"result"}],"a":12345}`, nil},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("body=%v", test.matchBody), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/path", strings.NewReader(test.requestBody))
			matchError := bodyMatcher(req, NewMock().Get("/path").Body(test.matchBody))
			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocksBodyMatcherRegexp(t *testing.T) {
	tests := []struct {
		requestBody   string
		matchBody     string
		expectedError error
	}{
		{"golang\n", "go[lang]?", nil},
		{"golang\n", "go[lang]?", nil},
		{"go\n", "go[lang]?", nil},
		{`{"a":"12345"}\n`, `{"a":"12345"}`, nil},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("body=%v", test.matchBody), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/path", strings.NewReader(test.requestBody))
			matchError := bodyRegexpMatcher(req, NewMock().Get("/path").Body(test.matchBody))
			assert.Equal(t, test.expectedError, matchError)
		})
	}
}

func TestMocksBodyMatcherSupportsRawArrays(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/path", strings.NewReader(`[{"a":1, "b": 2, "c": "something"}]`))
	matchError := bodyMatcher(req, NewMock().Get("/path").JSON(`[{"b": 2, "c": "something", "a": 1}]`))
	assert.NoError(t, matchError)
}

func TestMocksRequestBody(t *testing.T) {
	tests := map[string]struct {
		requestBody interface{}
	}{
		"supports string input": {`{"a":1}`},
		"supports maps":         {map[string]int{"a": 1}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/path", strings.NewReader(`{"a":1}`))
			err := bodyMatcher(req, NewMock().Get("/").JSON(test.requestBody))
			assert.NoError(t, err)
		})
	}
}

func TestMocksPathMatcher(t *testing.T) {
	tests := []struct {
		requestURL    string
		pathToMatch   string
		expectedError error
	}{
		{"http://test.com/v1/path", "/v1/path", nil},
		{"http://test.com/v1/path", "/v1/not", errors.New("received path /v1/path did not match mock path /v1/not")},
		{"http://test.com/v1/path", "", nil},
		{"http://test.com", "", nil},
		{"http://test.com/v2/path", "/v2/.+th", nil},
	}
	for _, test := range tests {
		t.Run(test.pathToMatch, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.requestURL, nil)
			matchError := pathMatcher(req, NewMock().Get(test.pathToMatch))
			if matchError != nil && !reflect.DeepEqual(matchError, test.expectedError) {
				t.Fatalf("methodToMatch='%s' requestUrl='%s' shouldMatch=%v",
					test.pathToMatch, test.requestURL, matchError)
			}
		})
	}
}

func TestMocksAddMatcher(t *testing.T) {
	tests := map[string]struct {
		matcherResponse error
		mockResponse    *MockResponse
		matchErrors     error
	}{
		"match": {
			matcherResponse: nil,
			mockResponse: &MockResponse{
				body:       `{"ok": true}`,
				statusCode: 200,
			},
			matchErrors: nil,
		},
		"no match": {
			matcherResponse: errors.New("nope"),
			mockResponse:    nil,
			matchErrors: &unmatchedMockError{errors: map[int][]error{
				1: {errors.New("nope")},
			}},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test/mock", nil)
			matcher := func(r *http.Request, mr *MockRequest) error {
				return test.matcherResponse
			}

			testMock := NewMock().
				Get("/test/mock").
				AddMatcher(matcher).
				RespondWith().
				Body(`{"ok": true}`).
				Status(http.StatusOK).
				End()

			mockResponse, matchErrors := matches(req, []*Mock{testMock})

			assert.Equal(t, test.matchErrors, matchErrors)
			if test.mockResponse == nil {
				assert.Equal(t, true, mockResponse == nil)
			} else {
				assert.Equal(t, test.mockResponse.body, mockResponse.body)
				assert.Equal(t, test.mockResponse.statusCode, mockResponse.statusCode)
			}
		})
	}
}

func TestMocksAddMatcherKeepsDefaultMocks(t *testing.T) {
	testMock := NewMock()

	// Default matchers present on new mock
	assert.Equal(t, len(defaultMatchers), len(testMock.request.matchers))

	testMock.Get("/test/mock").
		AddMatcher(func(r *http.Request, mr *MockRequest) error {
			return nil
		}).
		RespondWith().
		Body(`{"ok": true}`).
		Status(http.StatusOK).
		End()

	// New matcher added successfully
	assert.Equal(t, len(defaultMatchers)+1, len(testMock.request.matchers))
}

func TestMocksPanicsIfURLInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected to panic")
		}
	}()

	NewMock().Get("http:// blah")
}

func TestMocksMatches(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/preferences/12345", nil)

	getPreferences := NewMock().
		Get("/preferences/12345").
		RespondWith().
		Body(`{"is_contactable": true}`).
		Status(http.StatusOK).
		End()
	getUser := NewMock().
		Get("/user/1234").
		RespondWith().
		Status(http.StatusOK).
		BodyFromFile("testdata/mock_response_body.json").
		End()

	mockResponse, matchErrors := matches(req, []*Mock{getUser, getPreferences})

	assert.Equal(t, true, matchErrors == nil)
	assert.Equal(t, true, mockResponse != nil)
	assert.Equal(t, `{"is_contactable": true}`, mockResponse.body)
}

func TestMocksMatchesErrors(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test/mock", nil)

	testMock := NewMock().
		Post("/test/mock").
		BodyFromFile("testdata/mock_request_body.json").
		Query("queryKey", "queryVal").
		QueryPresent("queryKey2").
		QueryParams(map[string]string{"queryKey": "queryVal"}).
		Header("headerKey", "headerVal").
		Headers(map[string]string{"headerKey": "headerVal"}).
		RespondWith().
		Header("responseHeaderKey", "responseHeaderVal").
		Body(`{"responseBodyKey": "responseBodyVal"}`).
		Status(http.StatusOK).
		End()

	mockResponse, matchErrors := matches(req, []*Mock{testMock})

	assert.Equal(t, true, mockResponse == nil)
	assert.Equal(t, &unmatchedMockError{errors: map[int][]error{
		1: {
			errors.New("received method GET did not match mock method POST"),
			errors.New("not all of received headers map[] matched expected mock headers map[Headerkey:[headerVal headerVal]]"),
			errors.New("not all of received query params map[] matched expected mock query params map[queryKey:[queryVal queryVal]]"),
			errors.New("expected query param queryKey2 not received"),
			errors.New("expected a body but received none"),
		},
	}}, matchErrors)
}

func TestMocksMatchesNilIfNoMatch(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/preferences/12345", nil)

	mockResponse, matchErrors := matches(req, []*Mock{})

	if mockResponse != nil {
		t.Fatal("Expected nil")
	}

	assert.Equal(t, true, matchErrors != nil)
	assert.Equal(t, newUnmatchedMockError(), matchErrors)
}

func TestMocksUnmatchedMockErrorOrderedMockKeys(t *testing.T) {
	unmatchedMockError := newUnmatchedMockError().
		addErrors(3, errors.New("oh no")).
		addErrors(1, errors.New("oh shoot")).
		addErrors(4, errors.New("gah"))

	assert.Equal(t,
		"received request did not match any mocks\n\nMock 1 mismatches:\n• oh shoot\n\nMock 3 mismatches:\n• oh no\n\nMock 4 mismatches:\n• gah\n\n",
		unmatchedMockError.Error())
}

func TestMocksMatchesErrorsMatchUnmatchedMocks(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/preferences/12345", nil)

	mockResponse, matchErrors := matches(req,
		[]*Mock{
			NewMock().
				Get("/preferences/123456").
				RespondWith().
				End()})

	if mockResponse != nil {
		t.Fatal("Expected nil")
	}

	assert.Equal(t, true, matchErrors != nil)
	assert.Equal(t, "received request did not match any mocks\n\nMock 1 mismatches:\n• received path /preferences/12345 did not match mock path /preferences/123456\n\n",
		matchErrors.Error())
}

func TestMocksMethodMatcher(t *testing.T) {
	tests := []struct {
		requestMethod string
		methodToMatch string
		expectedError error
	}{
		{http.MethodGet, http.MethodGet, nil},
		{http.MethodPost, http.MethodPost, nil},
		{http.MethodDelete, "", nil},
		{http.MethodPut, http.MethodGet, errors.New("received method PUT did not match mock method GET")},
		{"", http.MethodGet, nil},
		{"", "", nil},
		{http.MethodOptions, http.MethodGet, errors.New("received method OPTIONS did not match mock method GET")},
	}
	for _, test := range tests {
		t.Run(test.requestMethod, func(t *testing.T) {
			req := httptest.NewRequest(test.requestMethod, "/path", nil)
			matchError := methodMatcher(req, NewMock().Method(test.methodToMatch))
			if !reflect.DeepEqual(matchError, test.expectedError) {
				t.Fatalf("methodToMatch='%s' requestMethod='%s' actual=%v shouldMatch=%v",
					test.methodToMatch, test.requestMethod, matchError, test.expectedError)
			}
		})
	}
}

func TestMocksRequestSetsTheMethod(t *testing.T) {
	tests := []struct {
		expectedMethod string
		methodSetter   func(m *Mock)
	}{
		{http.MethodGet, func(m *Mock) { m.Get("/") }},
		{http.MethodPost, func(m *Mock) { m.Post("/") }},
		{http.MethodPut, func(m *Mock) { m.Put("/") }},
		{http.MethodDelete, func(m *Mock) { m.Delete("/") }},
		{http.MethodPatch, func(m *Mock) { m.Patch("/") }},
		{http.MethodHead, func(m *Mock) { m.Head("/") }},
	}
	for _, test := range tests {
		t.Run(test.expectedMethod, func(t *testing.T) {
			mock := NewMock()
			test.methodSetter(mock)
			assert.Equal(t, test.expectedMethod, mock.request.method)
		})
	}
}

func TestMocksURLFormatterSupport(t *testing.T) {
	t.Run("Getf", func(tc *testing.T) {
		req := NewMock().Getf("/user/%d", 1)
		assert.Equal(tc, "/user/1", req.url.Path)
		assert.Equal(tc, "GET", req.method)
	})

	t.Run("Putf", func(tc *testing.T) {
		req := NewMock().Putf("/user/%d", 1)
		assert.Equal(tc, "/user/1", req.url.Path)
		assert.Equal(tc, "PUT", req.method)
	})

	t.Run("Patchf", func(tc *testing.T) {
		req := NewMock().Patchf("/user/%d", 1)
		assert.Equal(tc, "/user/1", req.url.Path)
		assert.Equal(tc, "PATCH", req.method)
	})

	t.Run("Deletef", func(tc *testing.T) {
		req := NewMock().Deletef("/user/%d", 1)
		assert.Equal(tc, "/user/1", req.url.Path)
		assert.Equal(tc, "DELETE", req.method)
	})

	t.Run("Postf", func(tc *testing.T) {
		req := NewMock().Postf("/user/%d", 1)
		assert.Equal(tc, "/user/1", req.url.Path)
		assert.Equal(tc, "POST", req.method)
	})
}

func TestMocksBodyFormatterSupport(t *testing.T) {
	t.Run("request body", func(tc *testing.T) {
		req := NewMock().Post("/user/1").Bodyf(`{"name": "%s"}`, "Jan")
		assert.Equal(tc, `{"name": "Jan"}`, req.body)
	})

	t.Run("response body", func(tc *testing.T) {
		res := NewMock().Get("/user/1").RespondWith().Bodyf(`{"name": "%s"}`, "Den")
		assert.Equal(tc, `{"name": "Den"}`, res.body)
	})
}

func TestMocksResponseSetsTextPlainIfNoContentTypeSet(t *testing.T) {
	mockResponse := NewMock().
		Get("assert").
		RespondWith().
		Body("abcdef")

	response := buildResponseFromMock(mockResponse)

	bytes, _ := io.ReadAll(response.Body)
	assert.Equal(t, string(bytes), "abcdef")
	assert.Equal(t, "text/plain", response.Header.Get("Content-Type"))
}

func TestMocksResponseSetsTheBodyAsJSON(t *testing.T) {
	mockResponse := NewMock().
		Get("assert").
		RespondWith().
		Body(`{"a": 123}`)

	response := buildResponseFromMock(mockResponse)

	bytes, _ := io.ReadAll(response.Body)
	assert.Equal(t, string(bytes), `{"a": 123}`)
	assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
}

func TestMocksResponseJSON(t *testing.T) {
	mockResponse := NewMock().
		Get("assert").
		RespondWith().
		JSON(map[string]int{"a": 123})

	response := buildResponseFromMock(mockResponse)

	bytes, _ := io.ReadAll(response.Body)
	assert.Equal(t, string(bytes), `{"a":123}`)
	assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
}

func TestMocksResponseSetsTheBodyAsOther(t *testing.T) {
	mockResponse := NewMock().
		Get("assert").
		RespondWith().
		Body(`<html>123</html>`).
		Header("Content-Type", "text/html")

	response := buildResponseFromMock(mockResponse)

	bytes, _ := io.ReadAll(response.Body)
	assert.Equal(t, string(bytes), `<html>123</html>`)
	assert.Equal(t, "text/html", response.Header.Get("Content-Type"))
}

func TestMocksResponseHeadersWithNormalizedKeys(t *testing.T) {
	mockResponse := NewMock().
		Get("assert").
		RespondWith().
		Header("a", "1").
		Headers(map[string]string{"B": "2"}).
		Header("c", "3")

	response := buildResponseFromMock(mockResponse)

	assert.Equal(t, http.Header(map[string][]string{"A": {"1"}, "B": {"2"}, "C": {"3"}}), response.Header)
}

func TestMocksResponseCookies(t *testing.T) {
	mockResponse := NewMock().
		Get("test").
		RespondWith().
		Cookie("A", "1").
		Cookies(NewCookie("B").Value("2")).
		Cookie("C", "3")

	response := buildResponseFromMock(mockResponse)

	assert.Equal(t, []*http.Cookie{
		{Name: "A", Value: "1", Raw: "A=1"},
		{Name: "B", Value: "2", Raw: "B=2"},
		{Name: "C", Value: "3", Raw: "C=3"},
	}, response.Cookies())
}

func TestMocksStandalone(t *testing.T) {
	cli := http.Client{Timeout: 5}
	defer NewMock().
		Post("http://localhost:8080/path").
		Body(`{"a", 12345}`).
		RespondWith().
		Status(http.StatusCreated).
		EndStandalone()()

	resp, err := cli.Post("http://localhost:8080/path",
		"application/json",
		strings.NewReader(`{"a", 12345}`))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestMocksStandaloneWithContainer(t *testing.T) {
	cli := http.Client{Timeout: 5}
	reset := NewStandaloneMocks(
		NewMock().
			Post("http://localhost:8080/path").
			Body(`{"a": 12345}`).
			RespondWith().
			Status(http.StatusCreated).
			End(),
		NewMock().
			Get("http://localhost:8080/path").
			RespondWith().
			Body(`{"a": 12345}`).
			Status(http.StatusOK).
			End(),
	).
		End()
	defer reset()

	resp, err := cli.Post("http://localhost:8080/path",
		"application/json",
		strings.NewReader(`{"a": 12345}`))

	assert.NoError(t, err)

	getRes, err := cli.Get("http://localhost:8080/path")

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	data, err := io.ReadAll(getRes.Body)

	assert.NoError(t, err)
	assert.JSONEq(t, `{"a": 12345}`, string(data))
}

func TestMocksStandaloneWithCustomHTTPClient(t *testing.T) {
	httpClient := customCli
	defer NewMock().
		HTTPClient(httpClient).
		Post("http://localhost:8080/path").
		Body(`{"a", 12345}`).
		RespondWith().
		Status(http.StatusCreated).
		EndStandalone()()

	resp, err := httpClient.Post("http://localhost:8080/path",
		"application/json",
		strings.NewReader(`{"a", 12345}`))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestMocksWithHTTPTimeout(t *testing.T) {
	httpClient := customCli
	defer NewMock().
		HTTPClient(httpClient).
		Post("http://localhost:8080/path").
		Body(`{"a", 12345}`).
		RespondWith().
		Timeout().
		EndStandalone()()

	_, err := httpClient.Post("http://localhost:8080/path",
		"application/json",
		strings.NewReader(`{"a", 12345}`))

	assert.Equal(t, true, err != nil)
	var isTimeout bool
	if err, ok := err.(net.Error); ok && err.Timeout() {
		isTimeout = true
	}
	assert.Equal(t, true, isTimeout)
}

func TestMocksApiTestWithMocks(t *testing.T) {
	tests := []struct {
		name    string
		httpCli *http.Client
	}{
		{name: "custom http cli", httpCli: customCli},
		{name: "default http cli", httpCli: http.DefaultClient},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			getUser := NewMock().
				Get("/user").
				RespondWith().
				Body(`{"name": "jon", "id": "1234"}`).
				FixedDelay(5000).
				Status(http.StatusOK).
				End()

			getPreferences := NewMock().
				Get("/preferences").
				RespondWith().
				Body(`{"is_contactable": false}`).
				Status(http.StatusOK).
				End()

			New().
				HTTPClient(test.httpCli).
				Mocks(getUser, getPreferences).
				Handler(getUserHandler(NewHTTPGet(test.httpCli))).
				Get("/user").
				Expect(t).
				Status(http.StatusOK).
				Body(`{"name": "jon", "is_contactable": false}`).
				End()
		})
	}
}

func TestMocksApiTestSupportsObservingMocks(t *testing.T) {
	var observedMocks []*mockInteraction

	getUser := NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("1").
		Times(2).
		End()

	getPreferences := NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("2").
		End()

	New().
		ObserveMocks(func(res *http.Response, req *http.Request, a *SpecTest) {
			if res == nil || req == nil || a == nil {
				t.Fatal("expected request and response to be defined")
			}
			assert.Equal(t, true, res.Request != nil, "expected request to be set in response")
			observedMocks = append(observedMocks, &mockInteraction{response: res, request: req})
		}).
		Mocks(getUser, getPreferences).
		Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes1 := getUserData()
			bytes2 := getUserData()
			bytes3 := getUserData()

			_, err := w.Write(bytes1)
			assert.Equal(t, true, err == nil)
			_, err = w.Write(bytes2)
			assert.Equal(t, true, err == nil)
			_, err = w.Write(bytes3)
			assert.Equal(t, true, err == nil)
			w.WriteHeader(http.StatusOK)
		})).
		Get("/").
		Expect(t).
		Status(http.StatusOK).
		Body(`112`).
		End()

	assert.Equal(t, 3, len(observedMocks))
}

func TestMocksApiTestSupportsObservingMocksWithReport(t *testing.T) {
	var observedMocks []*mockInteraction
	reporter := &RecorderCaptor{}
	observeMocksCalled := false

	getUser := NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("1").
		Times(2).
		End()

	getPreferences := NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("2").
		FixedDelay(1000).
		End()

	New().
		Report(reporter).
		EnableMockResponseDelay().
		ObserveMocks(func(res *http.Response, req *http.Request, a *SpecTest) {
			observeMocksCalled = true
			if res == nil || req == nil || a == nil {
				t.Fatal("expected request and response to be defined")
			}
			observedMocks = append(observedMocks, &mockInteraction{response: res, request: req})
		}).
		Mocks(getUser, getPreferences).
		Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes1 := getUserData()
			bytes2 := getUserData()
			bytes3 := getUserData()

			_, err := w.Write(bytes1)
			assert.NoError(t, err)
			_, err = w.Write(bytes2)
			assert.NoError(t, err)
			_, err = w.Write(bytes3)
			assert.NoError(t, err)

			w.WriteHeader(http.StatusOK)
		})).
		Get("/").
		Expect(t).
		Status(http.StatusOK).
		Body(`112`).
		End()

	assert.Equal(t, 3, len(observedMocks))
	assert.True(t, observeMocksCalled)
	oneSecondInNanoSecs := int64(1000000000)
	assert.True(t, reporter.capturedRecorder.Meta.Duration > oneSecondInNanoSecs)
}

func TestMocksApiTestSupportsMultipleMocks(t *testing.T) {
	getUser := NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("1").
		Times(2).
		End()

	getPreferences := NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("2").
		End()

	New().
		Mocks(getUser, getPreferences).
		Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes1 := getUserData()
			bytes2 := getUserData()
			bytes3 := getUserData()

			_, err := w.Write(bytes1)
			assert.NoError(t, err)
			_, err = w.Write(bytes2)
			assert.NoError(t, err)
			_, err = w.Write(bytes3)
			assert.NoError(t, err)
			w.WriteHeader(http.StatusOK)
		})).
		Get("/").
		Expect(t).
		Status(http.StatusOK).
		Body(`112`).
		End()
}

func getUserData() []byte {
	res, err := http.Get("http://localhost:8080")
	if err != nil {
		panic(err)
	}
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	return bytes
}

func getUserHandler(get HTTPGet) *http.ServeMux {
	handler := http.NewServeMux()
	handler.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		var user User
		get("/user", &user)

		var contactPreferences ContactPreferences
		get("/preferences", &contactPreferences)

		response := UserResponse{
			Name:          user.Name,
			IsContactable: contactPreferences.IsContactable,
		}
		bytes, _ := json.Marshal(response)
		_, err := w.Write(bytes)
		if err != nil {
			panic(err)
		}
		w.WriteHeader(http.StatusOK)
	})
	return handler
}

type User struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type ContactPreferences struct {
	IsContactable bool `json:"is_contactable"`
}

type UserResponse struct {
	Name          string `json:"name"`
	IsContactable bool   `json:"is_contactable"`
}

var customCli = &http.Client{
	Transport: &http.Transport{},
}

type HTTPGet func(path string, response interface{})

func NewHTTPGet(cli *http.Client) HTTPGet {
	return func(path string, response interface{}) {
		res, err := cli.Get(fmt.Sprintf("http://localhost:8080%s", path))
		if err != nil {
			panic(err)
		}

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}

		err = json.Unmarshal(bytes, response)
		if err != nil {
			panic(err)
		}
	}
}

type RecorderCaptor struct {
	capturedRecorder Recorder
}

func (r *RecorderCaptor) Format(recorder *Recorder) {
	r.capturedRecorder = *recorder
}

var assert = DefaultVerifier{}

func TestMockMethodSetup(t *testing.T) {
	t.Run("success Mock.Headf()", func(t *testing.T) {
		mock := NewMock().Headf("/user/%d", 1)
		assert.Equal(t, "/user/1", mock.mock.request.url.Path)
		assert.Equal(t, "HEAD", mock.mock.request.method)
	})

	t.Run("success Mock.Connectf()", func(t *testing.T) {
		mock := NewMock().Connectf("/user/%d", 1)
		assert.Equal(t, "/user/1", mock.mock.request.url.Path)
		assert.Equal(t, "CONNECT", mock.mock.request.method)
	})

	t.Run("success Mock.Optionsf()", func(t *testing.T) {
		mock := NewMock().Optionsf("/user/%d", 1)
		assert.Equal(t, "/user/1", mock.mock.request.url.Path)
		assert.Equal(t, "OPTIONS", mock.mock.request.method)
	})

	t.Run("success Mock.Tracef()", func(t *testing.T) {
		mock := NewMock().Tracef("/user/%d", 1)
		assert.Equal(t, "/user/1", mock.mock.request.url.Path)
		assert.Equal(t, "TRACE", mock.mock.request.method)
	})
}
