package spectest

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	// ErrTimeout is an error that indicates a timeout.
	ErrTimeout = errors.New("deadline exceeded")
)

// errorOrNil returns nil if the statement is true, otherwise it returns an error with the given message.
func errorOrNil(statement bool, errorMessage func() string) error {
	if statement {
		return nil
	}
	return errors.New(errorMessage())
}

// unmatchedMockError is used to store errors when a request does not match any mocks.
// It implements the error interface.
type unmatchedMockError struct {
	// errors is a map of mock number to errors
	errors map[int][]error
}

// newUnmatchedMockError creates a new unmatchedMockError
func newUnmatchedMockError() *unmatchedMockError {
	return &unmatchedMockError{
		errors: map[int][]error{},
	}
}

// append adds errors to the unmatchedMockError
func (u *unmatchedMockError) append(mockNumber int, errors ...error) *unmatchedMockError {
	u.errors[mockNumber] = append(u.errors[mockNumber], errors...)
	return u
}

// Error implementation of in-built error human readable string function
func (u *unmatchedMockError) Error() string {
	var strBuilder strings.Builder
	strBuilder.WriteString("received request did not match any mocks\n\n")
	for _, mockNumber := range u.orderedMockKeys() {
		strBuilder.WriteString(fmt.Sprintf("Mock %d mismatches:\n", mockNumber))
		for _, err := range u.errors[mockNumber] {
			strBuilder.WriteString("• ")
			strBuilder.WriteString(err.Error())
			strBuilder.WriteString("\n")
		}
		strBuilder.WriteString("\n")
	}
	return strBuilder.String()
}

// orderedMockKeys returns the keys of the errors map in order.
func (u *unmatchedMockError) orderedMockKeys() []int {
	mockKeys := make([]int, 0, len(u.errors))
	for mockKey := range u.errors {
		mockKeys = append(mockKeys, mockKey)
	}
	sort.Ints(mockKeys)
	return mockKeys
}
