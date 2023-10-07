package selector_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-spectest/spectest"

	selector "github.com/go-spectest/css-selector"
)

func TestTextExists(t *testing.T) {
	spectest.New().
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`<html>
			<head>
				<title>My document</title>
			</head>
			<body>
			<h1>Header</h1>
			<p>Some text to match on</p>
			</body>
			</html>`,
			))
			w.WriteHeader(http.StatusOK)
		}).
		Get("/").
		Expect(t).
		Status(http.StatusOK).
		Assert(selector.TextExists("Some text to match on")).
		End()
}

func TestWithDataTestID(t *testing.T) {
	spectest.New().
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`<html>
		<head>
			<title>My document</title>
		</head>
		<body>
		<h1>Header</h1>
		<div data-test-id="some-test-id">
			<div>some content</div>
		</div>
		</body>
		</html>`,
			))
			w.WriteHeader(http.StatusOK)
		}).
		Get("/").
		Expect(t).
		Status(http.StatusOK).
		Assert(selector.ContainsTextValue(selector.DataTestID("some-test-id"), "some content")).
		End()
}

func TestSelectorFirstTextValue(t *testing.T) {
	tests := map[string]struct {
		selector     string
		responseBody string
		expected     string
	}{
		"first text value": {
			selector: "h1",
			responseBody: `<html>
				<head>
					<title>My document</title>
				</head>
				<body>
					<h1>Header</h1>
				</body>
			</html>`,
			expected: "Header",
		},
		"first text value with class": {
			selector:     ".myClass",
			responseBody: `<div class="myClass">content</div>`,
			expected:     "content",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			spectest.New().
				HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte(test.responseBody))
					w.WriteHeader(http.StatusOK)
				}).
				Get("/").
				Expect(t).
				Status(http.StatusOK).
				Assert(selector.FirstTextValue(test.selector, test.expected)).
				End()
		})
	}
}

func TestSelectorspectestFirstTextValueNoMatch(t *testing.T) {
	verifier := &mockVerifier{
		EqualMock: func(t spectest.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool {
			expectedError := "did not find expected value for selector '.myClass'"
			if actual.(error).Error() != expectedError {
				t.Fatalf("actual was unexpected: %v", actual)
			}
			return true
		},
		NoErrorMock: func(t spectest.TestingT, err error, msgAndArgs ...interface{}) bool {
			return true
		},
	}

	spectest.New().
		Verifier(verifier).
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`<div class="myClass">content</div>`))
			w.WriteHeader(http.StatusOK)
		}).
		Get("/").
		Expect(t).
		Assert(selector.FirstTextValue(".myClass", "notContent")).
		End()
}

func TestSelectorNthTextValue(t *testing.T) {
	tests := map[string]struct {
		selector     string
		responseBody string
		expected     string
		n            int
	}{
		"second text value": {
			selector: ".myClass",
			responseBody: `<div>
				<div class="myClass">first</div>
				<div class="myClass">second</div>
			</div>`, expected: "first",
			n: 0,
		},
		"last text value": {
			selector: ".myClass",
			responseBody: `<div>
				<div class="myClass">first</div>
				<div class="myClass">second</div>
			</div>`, expected: "second",
			n: 1,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			spectest.New().
				HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte(test.responseBody))
					w.WriteHeader(http.StatusOK)
				}).
				Get("/").
				Expect(t).
				Status(http.StatusOK).
				Assert(selector.NthTextValue(test.n, test.selector, test.expected)).
				End()
		})
	}
}

func TestSelectorsTextValueContains(t *testing.T) {
	tests := map[string]struct {
		selector     string
		responseBody string
		expected     string
	}{
		"text value contains": {
			selector: ".myClass",
			responseBody: `<div>
				<div class="myClass">first</div>
				<div class="myClass">something second</div>
			</div>`,
			expected: "second",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			spectest.New().
				HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte(test.responseBody))
					w.WriteHeader(http.StatusOK)
				}).
				Get("/").
				Expect(t).
				Status(http.StatusOK).
				Assert(selector.ContainsTextValue(test.selector, test.expected)).
				End()
		})
	}
}

func TestSelectorExistsNoMatch(t *testing.T) {
	verifier := &mockVerifier{
		EqualMock: func(t spectest.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool {
			expectedError := "expected found='true' for selector '.myClass'"
			if actual.(error).Error() != expectedError {
				t.Fatal()
			}
			return true
		},
		NoErrorMock: func(t spectest.TestingT, err error, msgAndArgs ...interface{}) bool {
			return true
		},
	}

	spectest.New().
		Verifier(verifier).
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`<div class="someClass">content</div>`))
			w.WriteHeader(http.StatusOK)
		}).
		Get("/").
		Expect(t).
		Assert(selector.Exists(".myClass")).
		End()
}

func TestSelectorExists(t *testing.T) {
	tests := map[string]struct {
		exists       bool
		selector     []string
		responseBody string
	}{
		"exists": {
			exists:   true,
			selector: []string{`div[data-test-id^="product-"]`},
		},
		"multiple exists": {
			exists:   true,
			selector: []string{`div[data-test-id^="product-"]`, `.otherClass`},
		},
		"not exists": {
			exists:   false,
			selector: []string{`div[data-test-id^="product-4"]`},
		},
		"multiple not exists": {
			exists:   false,
			selector: []string{`div[data-test-id^="product-4"]`, `.notExistClass`},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			sel := selector.NotExists(test.selector...)
			if test.exists {
				sel = selector.Exists(test.selector...)
			}
			spectest.New().
				HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte(`<div>
					<div class="myClass">first</div>
					<div class="otherClass">something second</div>
					<div data-test-id="product-5">first</div>
				</div>`))
					w.WriteHeader(http.StatusOK)
				}).
				Get("/").
				Expect(t).
				Assert(sel).
				End()
		})
	}
}

func TestSelectorSelection(t *testing.T) {
	tests := map[string]struct {
		selector      string
		selectionFunc func(*goquery.Selection) error
		responseBody  string
		expectedText  string
	}{
		"with selection": {
			selector: `div[data-test-id^="product-"]`,
			responseBody: `<div>
				<div class="otherClass">something second</div>
				<div data-test-id="product-5">
					<div class="myClass">expectedText</div>
				</div>
			</div>`,
			expectedText: "expectedText",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			spectest.New().
				HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte(test.responseBody))
					w.WriteHeader(http.StatusOK)
				}).
				Get("/").
				Expect(t).
				Status(http.StatusOK).
				Assert(selector.Selection(test.selector, func(selection *goquery.Selection) error {
					if test.expectedText != selection.Find(".myClass").Text() {
						return fmt.Errorf("text did not match")
					}
					return nil
				})).
				End()
		})
	}
}

func TestSelectorSelectionNotMatch(t *testing.T) {
	verifier := &mockVerifier{
		EqualMock: func(t spectest.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool {
			expectedError := "text did not match"
			if actual.(error).Error() != expectedError {
				t.Fatalf("actual was unexpected: %v", actual)
			}
			return true
		},
		NoErrorMock: func(t spectest.TestingT, err error, msgAndArgs ...interface{}) bool {
			return true
		},
	}

	tests := map[string]struct {
		selector      string
		selectionFunc func(*goquery.Selection) error
		responseBody  string
		expectedText  string
	}{
		"with selection": {
			selector: `div[data-test-id^="product-"]`,
			responseBody: `<div>
				<div class="otherClass">something second</div>
				<div data-test-id="product-5">
					<div class="myClass">notExpectedText</div>
				</div>
			</div>`,
			expectedText: "expectedText",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			spectest.New().
				Verifier(verifier).
				HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte(test.responseBody))
					w.WriteHeader(http.StatusOK)
				}).
				Get("/").
				Expect(t).
				Assert(selector.Selection(test.selector, func(selection *goquery.Selection) error {
					if test.expectedText != selection.Find(".myClass").Text() {
						return fmt.Errorf("text did not match")
					}
					return nil
				})).
				End()
		})
	}
}

func TestSelectorMultipleExistsNoMatch(t *testing.T) {
	verifier := &mockVerifier{
		EqualMock: func(t spectest.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool {
			expectedError := "expected found='true' for selector '.myClass'"
			if actual.(error).Error() != expectedError {
				t.Fatalf("actual was unexpected: %v", actual)
			}
			return true
		},
		NoErrorMock: func(t spectest.TestingT, err error, msgAndArgs ...interface{}) bool {
			return true
		},
	}

	spectest.New().
		Verifier(verifier).
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`<div class="someClass">content</div>`))
			w.WriteHeader(http.StatusOK)
		}).
		Get("/").
		Expect(t).
		Assert(selector.Exists(".someClass", ".myClass")).
		End()
}

type mockVerifier struct {
	EqualInvoked bool
	EqualMock    func(spectest.TestingT, interface{}, interface{}, ...interface{}) bool

	TrueInvoked bool
	TrueMock    func(t spectest.TestingT, value bool, msgAndArgs ...interface{}) bool

	JSONEqInvoked bool
	JSONEqMock    func(t spectest.TestingT, expected string, actual string, msgAndArgs ...interface{}) bool

	FailInvoked bool
	FailMock    func(t spectest.TestingT, failureMessage string, msgAndArgs ...interface{}) bool

	NoErrorInvoked bool
	NoErrorMock    func(t spectest.TestingT, err error, msgAndArgs ...interface{}) bool
}

func (m *mockVerifier) Equal(t spectest.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool {
	m.EqualInvoked = true
	return m.EqualMock(t, expected, actual, msgAndArgs)
}

func (m *mockVerifier) True(t spectest.TestingT, value bool, msgAndArgs ...interface{}) bool {
	m.TrueInvoked = true
	return m.TrueMock(t, value, msgAndArgs)
}

func (m *mockVerifier) JSONEq(t spectest.TestingT, expected string, actual string, msgAndArgs ...interface{}) bool {
	m.JSONEqInvoked = true
	return m.JSONEqMock(t, expected, actual, msgAndArgs)
}

func (m *mockVerifier) Fail(t spectest.TestingT, failureMessage string, msgAndArgs ...interface{}) bool {
	m.FailInvoked = true
	return m.FailMock(t, failureMessage, msgAndArgs)
}

func (m *mockVerifier) NoError(t spectest.TestingT, err error, msgAndArgs ...interface{}) bool {
	m.NoErrorInvoked = true
	return m.NoErrorMock(t, err, msgAndArgs)
}
