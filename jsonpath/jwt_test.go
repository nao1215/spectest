package jsonpath_test

import (
	"net/http"
	"testing"

	"github.com/nao1215/spectest"

	"github.com/nao1215/spectest/jsonpath"
)

const jwt = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

func TestApiTestJWT(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Authorization", jwt)
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.JWTPayloadEqual(fromAuthHeader, `$.name`, "John Doe")).
		Assert(jsonpath.JWTPayloadEqual(fromAuthHeader, `$.sub`, "1234567890")).
		Assert(jsonpath.JWTPayloadEqual(fromAuthHeader, `$.iat`, float64(1516239022))).
		Assert(jsonpath.JWTHeaderEqual(fromAuthHeader, `$.alg`, "HS256")).
		Assert(jsonpath.JWTHeaderEqual(fromAuthHeader, `$.typ`, "JWT")).
		End()
}

func fromAuthHeader(response *http.Response) (string, error) {
	return response.Header.Get("Authorization"), nil
}
