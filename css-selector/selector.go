// Package selector provides a set of functions for css selector based assertions
package selector

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type selectionMatcher func(i int, selection *goquery.Selection) bool

// DataTestID returns a css selector for a data-test-id attribute
func DataTestID(value string) string {
	return fmt.Sprintf(`[data-test-id="%s"]`, value)
}

// FirstTextValue returns a function that asserts the first element matching the selection has the expected text value
func FirstTextValue(selection string, expectedTextValue string) func(*http.Response, *http.Request) error {
	return newAssertSelection(selection, func(i int, selection *goquery.Selection) bool {
		if i == 0 {
			if selection.Text() == expectedTextValue {
				return true
			}
		}
		return false
	})
}

// NthTextValue returns a function that asserts the nth element matching the selection has the expected text value
func NthTextValue(n int, selection string, expectedTextValue string) func(*http.Response, *http.Request) error {
	return newAssertSelection(selection, func(i int, selection *goquery.Selection) bool {
		if i == n {
			if selection.Text() == expectedTextValue {
				return true
			}
		}
		return false
	})
}

// ContainsTextValue returns a function that asserts the first element matching the selection contains the expected text value
func ContainsTextValue(selection string, expectedTextValue string) func(*http.Response, *http.Request) error {
	return newAssertSelection(selection, func(i int, selection *goquery.Selection) bool {
		if strings.Contains(selection.Text(), expectedTextValue) {
			return true
		}
		return false
	})
}

// Selection returns
func Selection(selection string, selectionFunc func(*goquery.Selection) error) func(*http.Response, *http.Request) error {
	return func(response *http.Response, request *http.Request) error {
		doc, err := goquery.NewDocumentFromReader(response.Body)
		if err != nil {
			return err
		}
		return selectionFunc(doc.Find(selection))
	}
}

// Exists returns a function that asserts the selection exists
func Exists(selections ...string) func(*http.Response, *http.Request) error {
	return expectExists(true, selections...)
}

// NotExists returns a function that asserts the selection does not exist
func NotExists(selections ...string) func(*http.Response, *http.Request) error {
	return expectExists(false, selections...)
}

// TextExists returns a function that asserts the document contains the expected text
func TextExists(text string) func(*http.Response, *http.Request) error {
	return func(response *http.Response, request *http.Request) error {
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}

		if !strings.Contains(string(bodyBytes), text) {
			return fmt.Errorf("document did not contain '%v'", text)
		}

		return nil
	}
}

func expectExists(exists bool, selections ...string) func(*http.Response, *http.Request) error {
	return func(response *http.Response, request *http.Request) error {
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}

		for _, selection := range selections {
			doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyBytes))
			if err != nil {
				return err
			}

			var found bool
			doc.Find(selection).Each(func(i int, selection *goquery.Selection) {
				found = true
			})

			if found != exists {
				return fmt.Errorf("expected found='%v' for selector '%s'", exists, selection)
			}
		}

		return nil
	}
}

func newAssertSelection(selection string, matcher selectionMatcher) func(*http.Response, *http.Request) error {
	return func(response *http.Response, request *http.Request) error {
		doc, err := goquery.NewDocumentFromReader(response.Body)
		if err != nil {
			return err
		}

		var found bool
		doc.Find(selection).Each(func(i int, selection *goquery.Selection) {
			if matcher(i, selection) {
				found = true
			}
		})

		if !found {
			return fmt.Errorf("did not find expected value for selector '%s'", selection)
		}

		return nil
	}
}
