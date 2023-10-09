package spectest

import (
	"net/http"
	"testing"
)

func TestApiTestAssertStatusCodes(t *testing.T) {
	tests := []struct {
		responseStatus []int
		assertFunc     Assert
		isSuccess      bool
	}{
		{[]int{http.StatusOK, 312., 399}, IsSuccess, true},
		{[]int{http.StatusBadRequest, http.StatusNotFound, 499}, IsClientError, true},
		{[]int{http.StatusInternalServerError, http.StatusServiceUnavailable}, IsServerError, true},
		{[]int{http.StatusBadRequest, http.StatusInternalServerError}, IsSuccess, false},
		{[]int{http.StatusOK, http.StatusInternalServerError}, IsClientError, false},
		{[]int{http.StatusOK, http.StatusBadRequest}, IsServerError, false},
	}
	for _, test := range tests {
		for _, status := range test.responseStatus {
			response := &http.Response{StatusCode: status}
			err := test.assertFunc(response, nil)
			if test.isSuccess && err != nil {
				t.Fatalf("Expected nil but received %s", err)
			} else if !test.isSuccess && err == nil {
				t.Fatalf("Expected error but didn't receive one")
			}
		}
	}
}
