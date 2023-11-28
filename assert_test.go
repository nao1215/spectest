package spectest

import (
	"net/http"
	"testing"
)

type mockTestingT struct{}

func (m *mockTestingT) Errorf(format string, args ...interface{}) {}
func (m *mockTestingT) Fatal(args ...interface{})                 {}
func (m *mockTestingT) Fatalf(format string, args ...interface{}) {}

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

func Test_DefaultVerifier_True(t *testing.T) {
	t.Parallel()
	verifier := &DefaultVerifier{}
	mock := &mockTestingT{}
	tests := []struct {
		name string
		args bool
		want bool
	}{
		{
			name: "should return true",
			args: true,
			want: true,
		},
		{
			name: "should return false",
			args: false,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := verifier.True(mock, tt.args)
			if actual != tt.want {
				t.Fatalf("Expected %t but received %t", actual, tt.want)
			}
		})
	}
}

func Test_DefaultVerifier_JSONEq(t *testing.T) {
	t.Parallel()

	verifier := &DefaultVerifier{}
	mock := &mockTestingT{}

	type args struct {
		expected string
		actual   string
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should return true",
			args: args{
				expected: `{"name":"John","age":30,"car":null}`,
				actual:   `{"name":"John","age":30,"car":null}`,
			},
			want: true,
		},
		{
			name: "should failure with different values",
			args: args{
				expected: `{"name":"John","age":30,"car":null}`,
				actual:   `{"name":"John","age":31,"car":null}`,
			},
			want: false,
		},
		{
			name: "should failure to parse expected",
			args: args{
				expected: `{"name":"John","age":30,"car":null`,
				actual:   `{"name":"John","age":30,"car":null}`,
			},
			want: false,
		},
		{
			name: "should failure to parse actual",
			args: args{
				expected: `{"name":"John","age":30,"car":null}`,
				actual:   `{"name":"John","age":30,"car":null`,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := verifier.JSONEq(mock, tt.args.expected, tt.args.actual)
			if actual != tt.want {
				t.Fatalf("Expected %t but received %t", actual, tt.want)
			}
		})
	}

}
