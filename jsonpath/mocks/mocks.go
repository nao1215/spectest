// Package mocks provides convenience functions for asserting jsonpath expressions
package mocks

import (
	"net/http"

	"github.com/go-spectest/spectest"
	httputil "github.com/go-spectest/spectest/jsonpath/http"
	"github.com/go-spectest/spectest/jsonpath/jsonpath"
)

// Contains is a convenience function to assert that a jsonpath expression extracts a value in an array
func Contains(expression string, expected interface{}) spectest.Matcher {
	return func(req *http.Request, mockReq *spectest.MockRequest) error {
		return jsonpath.Contains(expression, expected, httputil.CopyRequest(req).Body)
	}
}

// Equal is a convenience function to assert that a jsonpath expression matches the given value
func Equal(expression string, expected interface{}) spectest.Matcher {
	return func(req *http.Request, mockReq *spectest.MockRequest) error {
		return jsonpath.Equal(expression, expected, httputil.CopyRequest(req).Body)
	}
}

// NotEqual is a function to check json path expression value is not equal to given value
func NotEqual(expression string, expected interface{}) spectest.Matcher {
	return func(req *http.Request, mockReq *spectest.MockRequest) error {
		return jsonpath.NotEqual(expression, expected, httputil.CopyRequest(req).Body)
	}
}

// Len asserts that value is the expected length, determined by reflect.Len
func Len(expression string, expectedLength int) spectest.Matcher {
	return func(req *http.Request, mockReq *spectest.MockRequest) error {
		return jsonpath.Length(expression, expectedLength, httputil.CopyRequest(req).Body)
	}
}

// GreaterThan asserts that value is greater than the given length, determined by reflect.Len
func GreaterThan(expression string, minimumLength int) spectest.Matcher {
	return func(req *http.Request, mockReq *spectest.MockRequest) error {
		return jsonpath.GreaterThan(expression, minimumLength, httputil.CopyRequest(req).Body)
	}
}
