package spectest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-spectest/spectest"
	"github.com/go-spectest/spectest/mocks"
	"github.com/google/go-cmp/cmp"
	"github.com/nao1215/gorky/file"
)

func TestApiTestResponseBody(t *testing.T) {
	spectest.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id": "1234", "name": "Andy"}`))
		w.WriteHeader(http.StatusOK)
	}).
		Get("/user/1234").
		Expect(t).
		Body(`{"id": "1234", "name": "Andy"}`).
		Status(http.StatusOK).
		End()
}

func TestApiTestHttpRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		if string(data) != `hello` {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		if r.Header.Get("key") != "val" {
			t.Fatal("expected header key=val")
		}
	})

	request := httptest.NewRequest(http.MethodGet, "/hello", strings.NewReader("hello"))
	request.Header.Set("key", "val")

	spectest.Handler(handler).
		HTTPRequest(request).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestAddsJSONBodyToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		if string(data) != `{"a": 12345}` {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	spectest.Handler(handler).
		Post("/hello").
		Body(`{"a": 12345}`).
		Header("Content-Type", "application/json").
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestAddsJSONBodyToRequestSupportsFormatter(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		if string(data) != `{"a": 12345}` {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Post("/hello").
		Bodyf(`{"a": %d}`, 12345).
		Header("Content-Type", "application/json").
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestRequestURLFormat(t *testing.T) {
	spectest.New().
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			spectest.DefaultVerifier{}.Equal(t, "/user/1234", r.URL.Path)
		}).
		Getf("/user/%s", "1234").
		Expect(t).
		Status(http.StatusOK).
		End()

	spectest.New().
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			spectest.DefaultVerifier{}.Equal(t, "/user/1234", r.URL.Path)
		}).
		Putf("/user/%s", "1234").
		Expect(t).
		Status(http.StatusOK).
		End()

	spectest.New().
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			spectest.DefaultVerifier{}.Equal(t, "/user/1234", r.URL.Path)
		}).
		Patchf("/user/%s", "1234").
		Expect(t).
		Status(http.StatusOK).
		End()

	spectest.New().
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			spectest.DefaultVerifier{}.Equal(t, "/user/1234", r.URL.Path)
		}).
		Postf("/user/%s", "1234").
		Expect(t).
		Status(http.StatusOK).
		End()

	spectest.New().
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			spectest.DefaultVerifier{}.Equal(t, "/user/1234", r.URL.Path)
		}).
		Deletef("/user/%s", "1234").
		Expect(t).
		Status(http.StatusOK).
		End()

	spectest.New().
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			spectest.DefaultVerifier{}.Equal(t, "/user/1234", r.URL.Path)
		}).
		Method(http.MethodGet).
		URLf("/user/%d", 1234).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestJSONBody(t *testing.T) {
	type bodyStruct struct {
		A int `json:"a"`
	}

	tests := map[string]struct {
		body interface{}
	}{
		"string": {
			body: `{"a": 12345}`,
		},
		"[]byte": {
			body: []byte(`{"a": 12345}`),
		},
		"struct": {
			body: bodyStruct{A: 12345},
		},
		"map": {
			body: map[string]interface{}{"a": 12345},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			handler := http.NewServeMux()
			handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
				data, _ := io.ReadAll(r.Body)
				spectest.DefaultVerifier{}.JSONEq(t, `{"a": 12345}`, string(data))
				if r.Header.Get("Content-Type") != "application/json" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusOK)
			})

			spectest.New().
				Handler(handler).
				Post("/hello").
				JSON(test.body).
				Expect(t).
				Status(http.StatusOK).
				End()
		})
	}
}

func TestApiTestAddsJSONBodyToRequestUsingJSON(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		if string(data) != `{"a": 12345}` {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Post("/hello").
		JSON(`{"a": 12345}`).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestAddsTextBodyToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		if string(data) != `hello` {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Put("/hello").
		Body(`hello`).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestAddsQueryParamsToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("a") != "b" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		QueryParams(map[string]string{"a": "b"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestAddsQueryParamCollectionToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "a=b&a=c&a=d&e=f" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		QueryCollection(map[string][]string{"a": {"b", "c", "d"}}).
		QueryParams(map[string]string{"e": "f"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestAddsQueryParamCollectionToRequestHandlesEmpty(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "e=f" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		QueryCollection(map[string][]string{}).
		QueryParams(map[string]string{"e": "f"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestCanCombineQueryParamMethods(t *testing.T) {
	expectedQueryString := "a=1&a=2&a=9&a=22&b=2"
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if expectedQueryString != r.URL.RawQuery {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Query("a", "9").
		Query("a", "22").
		QueryCollection(map[string][]string{"a": {"1", "2"}}).
		QueryParams(map[string]string{"b": "2"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestAddsHeadersToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		header := r.Header["Authorization"]
		if len(header) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Delete("/hello").
		Headers(map[string]string{"Authorization": "12345"}).
		Header("Authorization", "098765").
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestAddsContentTypeHeaderToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.Header["Content-Type"][0] != "application/x-www-form-urlencoded" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Post("/hello").
		ContentType("application/x-www-form-urlencoded").
		Body(`name=John`).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestAddsCookiesToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("Cookie1"); err != nil || cookie.Value != "Yummy" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if cookie, err := r.Cookie("Cookie"); err != nil || cookie.Value != "Nom" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Method(http.MethodGet).
		URL("/hello").
		Cookie("Cookie", "Nom").
		Cookies(spectest.NewCookie("Cookie1").Value("Yummy")).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestAddsBasicAuthToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if username != "username" || password != "password" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	spectest.New("some test name").
		Handler(handler).
		Get("/hello").
		BasicAuth("username", "password").
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestAddsTimedOutContextToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Sleep time is temporarily extended because the sleep time is not timed out
		// unless the sleep time is significantly extended in a windows environment
		time.Sleep(time.Second * 3)
		if r.Context().Err() == context.DeadlineExceeded {
			w.WriteHeader(http.StatusRequestTimeout)
		}
		w.WriteHeader(http.StatusOK)
	})

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	spectest.New("test with timed out context").
		Handler(handler).
		Get("/hello").
		WithContext(timeoutCtx).
		Expect(t).
		Status(http.StatusRequestTimeout).
		End()
}

func TestApiTestAddsCancelledContextToRequest(t *testing.T) {

	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Err() == context.Canceled {
			w.WriteHeader(http.StatusNoContent)
		}
		w.WriteHeader(http.StatusOK)
	})

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	spectest.New("test with canceled context").
		Handler(handler).
		Get("/hello").
		WithContext(cancelCtx).
		Expect(t).
		Status(http.StatusNoContent).
		End()
}

func TestApiTestGraphQLQuery(t *testing.T) {
	spectest.New().
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			var req spectest.GraphQLRequestBody
			if err := json.Unmarshal(bodyBytes, &req); err != nil {
				t.Fatal(err)
			}

			spectest.DefaultVerifier{}.Equal(t, spectest.GraphQLRequestBody{
				Query: `query { todos { text } }`,
			}, req)

			w.WriteHeader(http.StatusOK)
		}).
		Post("/query").
		GraphQLQuery(`query { todos { text } }`).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestGraphQLRequest(t *testing.T) {
	spectest.New().
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			var req spectest.GraphQLRequestBody
			if err := json.Unmarshal(bodyBytes, &req); err != nil {
				t.Fatal(err)
			}

			expected := spectest.GraphQLRequestBody{
				Query:         `query { todos { text } }`,
				OperationName: "myOperation",
				Variables: map[string]interface{}{
					"a": float64(1),
					"b": "2",
				},
			}

			spectest.DefaultVerifier{}.Equal(t, expected, req)

			w.WriteHeader(http.StatusOK)
		}).
		Post("/query").
		GraphQLRequest(spectest.GraphQLRequestBody{
			Query: "query { todos { text } }",
			Variables: map[string]interface{}{
				"a": 1,
				"b": "2",
			},
			OperationName: "myOperation",
		}).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestMatchesJSONResponseBody(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": 12345}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Body(`{"a": 12345}`).
		Status(http.StatusCreated).
		End()
}

func TestApiTestMatchesJSONResponseBodyWithFormatter(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": 12345}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Bodyf(`{"a": %d}`, 12345).
		Status(http.StatusCreated).
		End()
}

func TestApiTestMatchesJSONBodyFromFile(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		spectest.DefaultVerifier{}.JSONEq(t, `{"a": 12345}`, string(data))

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": 12345}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Post("/hello").
		JSONFromFile("testdata/request_body.json").
		Expect(t).
		BodyFromFile("testdata/response_body.json").
		Status(http.StatusCreated).
		End()
}

func TestApiTestMatchesBodyFromFile(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		spectest.DefaultVerifier{}.JSONEq(t, `{"a": 12345}`, string(data))

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": 12345}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Post("/hello").
		BodyFromFile("testdata/request_body.json").
		Header("ContentType", "application/json").
		Expect(t).
		BodyFromFile("testdata/response_body.json").
		Status(http.StatusCreated).
		End()
}

func TestApiTestMatchesJSONResponseBodyWithWhitespace(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": 12345, "b": "hi"}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Body(`{
			"a": 12345,
			"b": "hi"
		}`).
		Status(http.StatusCreated).
		End()
}

func TestApiTestMatchesTextResponseBody(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte(`hello`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Body(`hello`).
		Status(http.StatusOK).
		End()
}

func TestApiTestMatchesResponseCookies(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-ExpectedCookie", "ABC=12345; DEF=67890; XXX=1fsadg235; VVV=9ig32g34g")
		http.SetCookie(w, &http.Cookie{
			Name:  "ABC",
			Value: "12345",
		})
		http.SetCookie(w, &http.Cookie{
			Name:  "DEF",
			Value: "67890",
		})
		http.SetCookie(w, &http.Cookie{
			Name:  "XXX",
			Value: "1fsadg235",
		})
		http.SetCookie(w, &http.Cookie{
			Name:  "VVV",
			Value: "9ig32g34g",
		})
		http.SetCookie(w, &http.Cookie{
			Name:  "YYY",
			Value: "kfiufhtne",
		})

		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Patch("/hello").
		Expect(t).
		Status(http.StatusOK).
		Cookies(
			spectest.NewCookie("ABC").Value("12345"),
			spectest.NewCookie("DEF").Value("67890")).
		Cookie("YYY", "kfiufhtne").
		CookiePresent("XXX").
		CookiePresent("VVV").
		CookieNotPresent("ZZZ").
		CookieNotPresent("TomBeer").
		End()
}

func TestApiTestMatchesResponseHttpCookies(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:  "ABC",
			Value: "12345",
		})
		http.SetCookie(w, &http.Cookie{
			Name:  "DEF",
			Value: "67890",
		})
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Cookies(
			spectest.NewCookie("ABC").Value("12345"),
			spectest.NewCookie("DEF").Value("67890")).
		End()
}

func TestApiTestMatchesResponseHttpCookiesOnlySuppliedFields(t *testing.T) {
	parsedDateTime, err := time.Parse(time.RFC3339, "2019-01-26T23:19:02Z")
	if err != nil {
		t.Fatalf("%s", err)
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    "pdsanjdna_8e8922",
			Path:     "/",
			Expires:  parsedDateTime,
			Secure:   true,
			HttpOnly: true,
		})
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Cookies(
			spectest.NewCookie("session_id").
				Value("pdsanjdna_8e8922").
				Path("/").
				Expires(parsedDateTime).
				Secure(true).
				HTTPOnly(true)).
		End()
}

func TestApiTestMatchesResponseHeadersWithMixedKeyCase(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ABC", "12345")
		w.Header().Set("DEF", "67890")
		w.Header().Set("Authorization", "12345")
		w.Header().Add("authorizATION", "00000")
		w.Header().Add("Authorization", "98765")
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		Headers(map[string]string{
			"Abc": "12345",
			"Def": "67890",
		}).
		Header("Authorization", "12345").
		Header("Authorization", "00000").
		Header("authorization", "98765").
		HeaderPresent("Def").
		HeaderPresent("Authorization").
		HeaderNotPresent("XYZ").
		End()
}

func TestApiTestEndReturnsTheResult(t *testing.T) {
	type resBody struct {
		B string `json:"b"`
	}

	var r resBody
	spectest.New().
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"a": 12345, "b": "hi"}`)); err != nil {
				t.Fatal(err)
			}
		}).
		Get("/hello").
		Expect(t).
		Body(`{
			"a": 12345,
			"b": "hi"
		}`).
		Status(http.StatusCreated).
		End().
		JSON(&r)

	spectest.DefaultVerifier{}.Equal(t, "hi", r.B)
}

func TestApiTestCustomAssert(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-ExpectedCookie", "ABC=12345; DEF=67890; XXX=1fsadg235; VVV=9ig32g34g")
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Patch("/hello").
		Expect(t).
		Assert(spectest.IsSuccess).
		End()
}

func TestApiTestVerifierCapturesTheTestMessage(t *testing.T) {
	verifier := mocks.NewVerifier()
	verifier.EqualFn = func(t spectest.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool {
		if expected == http.StatusOK {
			return true
		}
		args := msgAndArgs[0].([]interface{})
		spectest.DefaultVerifier{}.Equal(t, 2, len(args))
		spectest.DefaultVerifier{}.Equal(t, "expected header 'Abc' not present in response", args[0].(string))
		return true
	}

	spectest.New("the test case name").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"id": "1234", "name": "Andy"}`))
			w.WriteHeader(http.StatusOK)
		}).
		Verifier(verifier).
		Get("/user/1234").
		Expect(t).
		Status(http.StatusOK).
		Header("Abc", "1234").
		End()
}

func TestApiTestReport(t *testing.T) {
	getUser := spectest.NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("1").
		Times(1).
		End()

	reporter := &RecorderCaptor{}

	spectest.New("some test").
		Debug().
		CustomHost("abc.com").
		Report(reporter).
		Mocks(getUser).
		Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Millisecond * 100)
			getUserData()
			w.WriteHeader(http.StatusOK)
		})).
		Post("/hello").
		Body(`{"a": 12345}`).
		Headers(map[string]string{"Content-Type": "application/json"}).
		Expect(t).
		Status(http.StatusOK).
		End()

	r := reporter.capturedRecorder
	spectest.DefaultVerifier{}.Equal(t, "POST /hello", r.Title)
	spectest.DefaultVerifier{}.Equal(t, "some test", r.SubTitle)
	spectest.DefaultVerifier{}.Equal(t, 4, len(r.Events))
	spectest.DefaultVerifier{}.Equal(t, http.StatusOK, r.Meta.StatusCode)
	spectest.DefaultVerifier{}.Equal(t, "/hello", r.Meta.Path)
	spectest.DefaultVerifier{}.Equal(t, http.MethodPost, r.Meta.Method)
	spectest.DefaultVerifier{}.Equal(t, "some test", r.Meta.Name)
	spectest.DefaultVerifier{}.Equal(t, "abc.com", r.Meta.Host)
	spectest.DefaultVerifier{}.Equal(t, true, r.Meta.Duration != 0)
}

func TestApiTestRecorder(t *testing.T) {
	getUser := spectest.NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("1").
		Times(1).
		End()

	reporter := &RecorderCaptor{}
	messageRequest := spectest.MessageRequest{
		Source:    "Source",
		Target:    "Target",
		Header:    "Header",
		Body:      "Body",
		Timestamp: time.Now().UTC(),
	}
	messageResponse := spectest.MessageResponse{
		Source:    "Source",
		Target:    "Target",
		Header:    "Header",
		Body:      "Body",
		Timestamp: time.Now().UTC(),
	}
	recorder := spectest.NewTestRecorder()
	recorder.AddMessageRequest(messageRequest)
	recorder.AddMessageResponse(messageResponse)

	spectest.New("some test").
		CustomHost("abc.com").
		Report(reporter).
		Recorder(recorder).
		Mocks(getUser).
		Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			getUserData()
			w.WriteHeader(http.StatusOK)
		})).
		Post("/hello").
		Body(`{"a": 12345}`).
		Headers(map[string]string{"Content-Type": "application/json"}).
		Expect(t).
		Status(http.StatusOK).
		End()

	r := reporter.capturedRecorder
	spectest.DefaultVerifier{}.Equal(t, 6, len(r.Events))
	spectest.DefaultVerifier{}.Equal(t, messageRequest, r.Events[0])
	spectest.DefaultVerifier{}.Equal(t, messageResponse, r.Events[1])
}

func TestApiTestObserve(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	observeCalled := false

	spectest.New("observe test").
		Observe(func(res *http.Response, req *http.Request, apiTest *spectest.SpecTest) {
			observeCalled = true
			spectest.DefaultVerifier{}.Equal(t, http.StatusOK, res.StatusCode)
			spectest.DefaultVerifier{}.Equal(t, "/hello", req.URL.Path)
		}).
		Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		End()

	spectest.DefaultVerifier{}.Equal(t, true, observeCalled)
}

func TestApiTestObserveDumpsTheHttpRequestAndResponse(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": 12345}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Post("/hello").
		Body(`{"a": 12345}`).
		Headers(map[string]string{"Content-Type": "application/json"}).
		Expect(t).
		Status(http.StatusCreated).
		End()
}

func TestApiTestObserveWithReport(t *testing.T) {
	reporter := &RecorderCaptor{}
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	observeCalled := false

	spectest.New("observe test").
		Report(reporter).
		Observe(func(res *http.Response, req *http.Request, apiTest *spectest.SpecTest) {
			observeCalled = true
			spectest.DefaultVerifier{}.Equal(t, http.StatusOK, res.StatusCode)
			spectest.DefaultVerifier{}.Equal(t, "/hello", req.URL.Path)
		}).
		Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		End()

	spectest.DefaultVerifier{}.Equal(t, true, observeCalled)
}

func TestApiTestIntercept(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "a[]=xxx&a[]=yyy" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Header.Get("Auth-Token") != "12345" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Intercept(func(req *http.Request) {
			req.URL.RawQuery = "a[]=xxx&a[]=yyy"
			req.Header.Set("Auth-Token", req.Header.Get("authtoken"))
		}).
		Get("/hello").
		Headers(map[string]string{"authtoken": "12345"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestExposesRequestAndResponse(t *testing.T) {
	apiTest := spectest.New()

	spectest.DefaultVerifier{}.Equal(t, true, apiTest.Request() != nil)
	spectest.DefaultVerifier{}.Equal(t, true, apiTest.Response() != nil)
}

func TestApiTestRequestContextIsPreserved(t *testing.T) {
	ctxKey := struct{}{}
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		value := r.Context().Value(ctxKey).([]byte)
		w.Write(value)
	})

	interceptor := func(r *http.Request) {
		*r = *r.WithContext(context.WithValue(r.Context(), ctxKey, []byte("world")))
	}

	spectest.New().
		Handler(handler).
		Intercept(interceptor).
		Get("/hello").
		Expect(t).
		Body("world").
		End()
}

func TestApiTestNoopVerifier(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": 12345}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Verifier(spectest.NoopVerifier{}).
		Get("/hello").
		Expect(t).
		Body(`{"a": 123456}`).
		Status(http.StatusBadGateway).
		End()
}

// TestRealNetworking creates a server with two endpoints, /login sets a token via a cookie and /authenticated_resource
// validates the token. A cookie jar is used to verify session persistence across multiple spectest instances
func TestRealNetworking(t *testing.T) {
	srv := &http.Server{Addr: "localhost:9876"}
	tokenValue := "ABCDEF"
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "Token", Value: tokenValue})
		w.WriteHeader(203)
	})
	http.HandleFunc("/authenticated_resource", func(w http.ResponseWriter, r *http.Request) {
		token, err := r.Cookie("Token")
		if err == http.ErrNoCookie {
			w.WriteHeader(400)
			return
		}
		if err != nil {
			w.WriteHeader(500)
			return
		}

		if token.Value != tokenValue {
			t.Fatalf("token did not equal %s", tokenValue)
		}
		w.WriteHeader(204)
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			panic(err)
		}
	}()
	time.Sleep(time.Millisecond * 100) // TODO: find a better way to wait for the server to start

	finish := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered in f", r)
			}
		}()

		cookieJar, _ := cookiejar.New(nil)
		cli := &http.Client{
			Timeout: time.Second * 1,
			Jar:     cookieJar,
		}

		spectest.New().
			EnableNetworking(cli).
			Get("http://localhost:9876/login").
			Expect(t).
			Status(203).
			End()

		spectest.New().
			EnableNetworking(cli).
			Get("http://localhost:9876/authenticated_resource").
			Expect(t).
			Status(204).
			End()

		finish <- struct{}{}
	}()
	<-finish
}

func TestApiTestAddsUrlEncodedFormBody(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.Header["Content-Type"][0] != "application/x-www-form-urlencoded" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		expectedPostFormData := map[string][]string{
			"name":     {"John"},
			"age":      {"99"},
			"children": {"Jack", "Ann"},
			"pets":     {"Toby", "Henry", "Alice"},
		}

		for key := range expectedPostFormData {
			if !reflect.DeepEqual(expectedPostFormData[key], r.PostForm[key]) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Post("/hello").
		FormData("name", "John").
		FormData("age", "99").
		FormData("children", "Jack").
		FormData("children", "Ann").
		FormData("pets", "Toby", "Henry", "Alice").
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestAddsMultipartFormData(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header["Content-Type"][0], "multipart/form-data") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err := r.ParseMultipartForm(2 << 32)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		expectedPostFormData := map[string][]string{
			"name":     {"John"},
			"age":      {"99"},
			"children": {"Jack", "Ann"},
			"pets":     {"Toby", "Henry", "Alice"},
		}

		for key := range expectedPostFormData {
			if !reflect.DeepEqual(expectedPostFormData[key], r.MultipartForm.Value[key]) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		for _, exp := range []struct {
			filename string
			data     string
		}{
			{
				filename: "response_body",
				data:     `{"a": 12345}`,
			},
			{
				filename: "mock_request_body",
				data:     `{"bodyKey": "bodyVal"}`,
			},
		} {
			for _, file := range r.MultipartForm.File[exp.filename] {
				spectest.DefaultVerifier{}.Equal(t, exp.filename+".json", file.Filename)

				f, err := file.Open()
				if err != nil {
					t.Fatal(err)
				}
				data, err := io.ReadAll(f)
				if err != nil {
					t.Fatal(err)
				}
				spectest.DefaultVerifier{}.JSONEq(t, exp.data, string(data))
			}
		}

		w.WriteHeader(http.StatusOK)
	})

	spectest.New().
		Handler(handler).
		Post("/hello").
		MultipartFormData("name", "John").
		MultipartFormData("age", "99").
		MultipartFormData("children", "Jack").
		MultipartFormData("children", "Ann").
		MultipartFormData("pets", "Toby", "Henry", "Alice").
		MultipartFile("request_body", "testdata/request_body.json", "testdata/request_body.json").
		MultipartFile("mock_request_body", "testdata/mock_request_body.json").
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTestCombineFormDataWithMultipart(t *testing.T) {
	if os.Getenv("RUN_FATAL_TEST") == "FormData" {
		spectest.New().
			Post("/hello").
			MultipartFormData("name", "John").
			FormData("name", "John")
		return
	}
	if os.Getenv("RUN_FATAL_TEST") == "File" {
		spectest.New().
			Post("/hello").
			MultipartFile("file", "testdata/request_body.json").
			FormData("name", "John")
		return
	}

	tests := map[string]string{
		"formdata_with_multiple_formdata": "FormData",
		"formdata_with_multiple_file":     "File",
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			cmd := exec.Command(os.Args[0], "-test.run=TestApiTestCombineFormDataWithMultipart")
			cmd.Env = append(os.Environ(), "RUN_FATAL_TEST="+tt)
			err := cmd.Run()
			if e, ok := err.(*exec.ExitError); ok && !e.Success() {
				return
			}
			t.Fatalf("process ran with err %v, want exit status 1", err)
		})
	}
}

func TestApiTestErrorIfMockInvocationsDoNotMatchTimes(t *testing.T) {
	getUser := spectest.NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Times(2).
		End()

	verifier := mocks.NewVerifier()
	verifier.FailFn = func(t spectest.TestingT, failureMessage string, msgAndArgs ...interface{}) bool {
		spectest.DefaultVerifier{}.Equal(t, "mock was not invoked expected times", failureMessage)
		return true
	}

	res := spectest.New().
		Mocks(getUser).
		Verifier(verifier).
		Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = getUserData()
			w.WriteHeader(http.StatusOK)
		})).
		Get("/").
		Expect(t).
		Status(http.StatusOK).
		End()

	unmatchedMocks := res.UnmatchedMocks()
	spectest.DefaultVerifier{}.Equal(t, true, len(unmatchedMocks) > 0)
	spectest.DefaultVerifier{}.Equal(t, "http://localhost:8080", unmatchedMocks[0].URL.String())
}

func TestApiTestMatchesTimes(t *testing.T) {
	getUser := spectest.NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Times(1).
		End()

	res := spectest.New().
		Mocks(getUser).
		Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = getUserData()
			w.WriteHeader(http.StatusOK)
		})).
		Get("/").
		Expect(t).
		Status(http.StatusOK).
		End()

	spectest.DefaultVerifier{}.Equal(t, 0, len(res.UnmatchedMocks()))
}

type RecorderCaptor struct {
	capturedRecorder spectest.Recorder
}

func (r *RecorderCaptor) Format(recorder *spectest.Recorder) {
	r.capturedRecorder = *recorder
}

func getUserData() []byte {
	res, err := http.Get("http://localhost:8080")
	if err != nil {
		panic(err)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	return data
}

func TestHTTPMethodHEAD(t *testing.T) {
	t.Run("success case: test Head()", func(t *testing.T) {
		handler := http.NewServeMux()
		handler.HandleFunc("/head", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodHead {
				t.Fatalf("expected method to be HEAD, got %s", r.Method)
			}
			w.Header().Set("Content-Length", "10")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
		})

		spectest.New().
			Handler(handler).
			Head("/head").
			Expect(t).
			Status(http.StatusOK).
			Header("Content-Length", "10").
			Header("Content-Type", "text/plain; charset=utf-8").
			Body(""). // Body is empty for HEAD requests
			End()
	})

	t.Run("success case: test Headf()", func(t *testing.T) {
		handler := http.NewServeMux()
		handler.HandleFunc("/head/123", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodHead {
				t.Fatalf("expected method to be HEAD, got %s", r.Method)
			}
			w.Header().Set("Content-Length", "10")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
		})

		spectest.New().
			Handler(handler).
			Headf("/head/%d", 123).
			Expect(t).
			Status(http.StatusOK).
			Header("Content-Length", "10").
			Header("Content-Type", "text/plain; charset=utf-8").
			Body(""). // Body is empty for HEAD requests
			End()
	})
}

func TestHTTPMethodConnect(t *testing.T) {
	t.Run("success case: test Connect()", func(t *testing.T) {
		handler := http.NewServeMux()
		handler.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodConnect {
				t.Fatalf("expected method to be CONNECT, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		})

		spectest.New().
			Handler(handler).
			Connect("/connect").
			Expect(t).
			Status(http.StatusOK).
			End()
	})

	t.Run("success case: test Connectf()", func(t *testing.T) {
		handler := http.NewServeMux()
		handler.HandleFunc("/connect/123", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodConnect {
				t.Fatalf("expected method to be CONNECT, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		})

		spectest.New().
			Handler(handler).
			Connectf("/connect/%d", 123).
			Expect(t).
			Status(http.StatusOK).
			End()
	})
}

func TestHTTPMethodOptions(t *testing.T) {
	t.Run("success case: test Options()", func(t *testing.T) {
		handler := http.NewServeMux()
		handler.HandleFunc("/options", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodOptions {
				t.Fatalf("expected method to be OPTIONS, got %s", r.Method)
			}
			w.Header().Set("Allow", "GET, HEAD, OPTIONS")
			w.WriteHeader(http.StatusOK)
		})

		spectest.New().
			Handler(handler).
			Options("/options").
			Expect(t).
			Header("Allow", "GET, HEAD, OPTIONS").
			Status(http.StatusOK).
			End()
	})

	t.Run("success case: test Optionsf()", func(t *testing.T) {
		handler := http.NewServeMux()
		handler.HandleFunc("/options/123", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodOptions {
				t.Fatalf("expected method to be OPTIONS, got %s", r.Method)
			}
			w.Header().Set("Allow", "GET, HEAD, OPTIONS")
			w.WriteHeader(http.StatusOK)
		})

		spectest.New().
			Handler(handler).
			Optionsf("/options/%d", 123).
			Expect(t).
			Header("Allow", "GET, HEAD, OPTIONS").
			Status(http.StatusOK).
			End()
	})
}

func TestHTTPMethodTrace(t *testing.T) {
	t.Run("success  case: test Trace()", func(t *testing.T) {
		handler := http.NewServeMux()
		handler.HandleFunc("/trace123", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodTrace {
				t.Fatalf("expected method to be TRACE, got %s", r.Method)
			}

			// write response header to response body after sorting the header keys
			keys := make([]string, 0, len(r.Header))
			for k := range r.Header {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				for _, v := range r.Header[k] {
					fmt.Fprintf(w, "%s: %s\n", k, v)
				}
			}
			w.WriteHeader(http.StatusOK)
		})

		spectest.New().
			Handler(handler).
			Tracef("/trace%s", "123").
			Header("User-Agent", "Go-http-client/1.1").
			Header("Keep-Alive", "timeout=5, max=1000").
			Expect(t).
			Body("Keep-Alive: timeout=5, max=1000\nUser-Agent: Go-http-client/1.1\n").
			Status(http.StatusOK).
			End()
	})
}

func TestReportWithImage(t *testing.T) {
	imagePath := filepath.Join("testdata", "sample.png")
	imageFile, err := os.Open(filepath.Clean(imagePath))
	if err != nil {
		t.Fatal(err)
	}
	defer imageFile.Close() //nolint

	imageInfo, err := imageFile.Stat()
	if err != nil {
		t.Fatal(err)
	}

	body, err := io.ReadAll(imageFile)
	if err != nil {
		t.Fatal(err)
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected method to be GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", fmt.Sprint(imageInfo.Size()))

		_, err = io.Copy(w, bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	tmpDir, err := os.MkdirTemp("", "spectest")
	if err != nil {
		t.Fatal(err)
	}

	spectest.New().
		CustomReportName("sample").
		Report(spectest.SequenceDiagram(tmpDir)).
		Handler(handler).
		Get("/image").
		Expect(t).
		Body(string(body)).
		Header("Content-Type", "image/png").
		Header("Content-Length", fmt.Sprint(imageInfo.Size())).
		Status(http.StatusOK).
		End()

	if !file.Exists(filepath.Join(tmpDir, "sample_1.png")) {
		t.Errorf("image file should exist")
	}
}

func TestMarkdownReportWithImage(t *testing.T) {
	imagePath := filepath.Join("testdata", "sample.png")
	imageFile, err := os.Open(filepath.Clean(imagePath))
	if err != nil {
		t.Fatal(err)
	}
	defer imageFile.Close() //nolint

	imageInfo, err := imageFile.Stat()
	if err != nil {
		t.Fatal(err)
	}

	body, err := io.ReadAll(imageFile)
	if err != nil {
		t.Fatal(err)
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected method to be GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", fmt.Sprint(imageInfo.Size()))

		_, err = io.Copy(w, bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	tmpDir, err := os.MkdirTemp("", "spectest")
	if err != nil {
		t.Fatal(err)
	}

	spectest.New().
		CustomReportName("sample").
		Report(spectest.SequenceReport(spectest.ReportFormatterConfig{
			Path: tmpDir,
			Kind: spectest.ReportKindMarkdown,
		})).
		Handler(handler).
		Get("/image").
		Expect(t).
		Body(string(body)).
		Header("Content-Type", "image/png").
		Header("Content-Length", fmt.Sprint(imageInfo.Size())).
		Status(http.StatusOK).
		End()

	if !file.Exists(filepath.Join(tmpDir, "sample.md")) {
		t.Errorf("markdown file should exist")
	}
	if !file.Exists(filepath.Join(tmpDir, "sample_1.png")) {
		t.Errorf("image file should exist")
	}
}

func TestMarkdownReportResponseJSON(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"a": 12345}`)); err != nil {
			t.Fatal(err)
		}
	})

	tmpDir, err := os.MkdirTemp("", "spectest")
	if err != nil {
		t.Fatal(err)
	}

	spectest.New().
		CustomReportName("sample").
		Report(spectest.SequenceReport(spectest.ReportFormatterConfig{
			Path: tmpDir,
			Kind: spectest.ReportKindMarkdown,
		})).
		Handler(handler).
		Post("/hello").
		Expect(t).
		Header("Content-Type", "application/json").
		BodyFromFile(filepath.Join("testdata", "request_body.json")).
		Status(http.StatusOK).
		End()

	if !file.Exists(filepath.Join(tmpDir, "sample.md")) {
		t.Errorf("markdown file should exist")
	}

	path := filepath.Join("testdata", "sample.md")
	if runtime.GOOS == "windows" {
		path = filepath.Join("testdata", "sample_windows.md")
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Clean(filepath.Join(tmpDir, "sample.md")))
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("markdown file mismatch (-want +got):\n%s", diff)
	}
}
