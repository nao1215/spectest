package jsonpath_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/go-spectest/spectest"
	"github.com/stretchr/testify/assert"

	"github.com/go-spectest/spectest/jsonpath"
)

func TestApiTestContains(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": 12345, "b": [{"key": "c", "value": "result"}], "d": null}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.Contains(`$.b[? @.key=="c"].value`, "result")).
		End()
}

func TestApiTestEqualNumeric(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": 12345, "b": [{"key": "c", "value": "result"}]}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.Equal(`$.a`, float64(12345))).
		End()
}

func TestApiTestEqualString(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": "12345", "b": [{"key": "c", "value": "result"}]}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.Equal(`$.a`, "12345")).
		End()
}

func TestApiTestEqualMap(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": "hello", "b": 12345}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.Equal(`$`, map[string]interface{}{"a": "hello", "b": float64(12345)})).
		End()
}

func TestApiTestNotEqualNumeric(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": 12345, "b": [{"key": "c", "value": "result"}]}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.NotEqual(`$.a`, float64(1))).
		End()
}

func TestApiTestNotEqualString(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": "12345", "b": [{"key": "c", "value": "result"}]}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.NotEqual(`$.a`, "1")).
		End()
}

func TestApiTestNotEqualMap(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": "hello", "b": 12345}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.NotEqual(`$`, map[string]interface{}{"a": "hello", "b": float64(1)})).
		End()
}

func TestApiTestLen(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": [1, 2, 3], "b": "c", "d": null}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.Len(`$.a`, 3)).
		Assert(jsonpath.Len(`$.b`, 1)).
		Assert(func(r1 *http.Response, r2 *http.Request) error {
			err := jsonpath.Len(`$.d`, 0)(r1, r2)

			if err == nil {
				return errors.New("jsonpath.Len was expected to fail on null value but it didn't")
			}

			assert.EqualError(t, err, "value is null")

			return nil
		}).
		End()
}

func TestApiTestGreaterThan(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": [1, 2, 3], "b": "c", "d": null}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.GreaterThan(`$.a`, 2)).
		Assert(jsonpath.GreaterThan(`$.b`, 0)).
		Assert(func(r1 *http.Response, r2 *http.Request) error {
			err := jsonpath.GreaterThan(`$.d`, 5)(r1, r2)

			if err == nil {
				return errors.New("jsonpath.GreaterThan was expected to fail on null value but it didn't")
			}

			assert.EqualError(t, err, "value is null")

			return nil
		}).
		End()
}

func TestApiTestLessThan(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": [1, 2, 3], "b": "c", "d": null}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.LessThan(`$.a`, 4)).
		Assert(jsonpath.LessThan(`$.b`, 2)).
		Assert(func(r1 *http.Response, r2 *http.Request) error {
			err := jsonpath.LessThan(`$.d`, 5)(r1, r2)

			if err == nil {
				return errors.New("jsonpath.LessThan was expected to fail on null value but it didn't")
			}

			assert.EqualError(t, err, "value is null")

			return nil
		}).
		End()
}

func TestApiTestPresent(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"a": 22}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.Present(`$.a`)).
		Assert(jsonpath.NotPresent(`$.password`)).
		End()
}

func TestApiTestMatches(t *testing.T) {
	testCases := [][]string{
		{`$.aString`, `^[mot]{3}<3[AB][re]{3}$`},
		{`$.aNumber`, `^\d$`},
		{`$.anObject.aNumber`, `^\d\.\d{3}$`},
		{`$.aNumberSlice[1]`, `^[80]$`},
		{`$.anObject.aBool`, `^true$`},
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"anObject":{"aString":"tom<3Beer","aNumber":7.212,"aBool":true},"aString":"tom<3Beer","aNumber":7,"aNumberSlice":[7,8,9],"aStringSlice":["7","8","9"]}`)); err != nil {
			t.Fatal(err)
		}
	})

	for testNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("match test %d", testNumber), func(t *testing.T) {
			spectest.New().
				Handler(handler).
				Get("/hello").
				Expect(t).
				Assert(jsonpath.Matches(testCase[0], testCase[1])).
				End()
		})
	}
}

func TestApiTestChain(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{
			"a": {
			"b": {
				"c": {
				"d": 1,
				"e": "2",
				"f": [3, 4, 5]
				}
			}
			}
		}`)); err != nil {
			t.Fatal(err)
		}
	})

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(
			jsonpath.Root("$.a.b.c").
				Equal("d", float64(1)).
				Equal("e", "2").
				Contains("f", float64(5)).
				End(),
		).
		End()

	spectest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(
			jsonpath.Chain().
				Equal("a.b.c.d", float64(1)).
				Equal("a.b.c.e", "2").
				Contains("a.b.c.f", float64(5)).
				End(),
		).
		End()
}

func TestApiTestMatchesFailCompile(t *testing.T) {
	willFailToCompile := jsonpath.Matches(`$.b[? @.key=="c"].value`, `\`)
	err := willFailToCompile(nil, nil)

	assert.EqualError(t, err, `invalid pattern: '\'`)
}

func TestApiTestMatchesFailForObject(t *testing.T) {
	matcher := jsonpath.Matches(`$.anObject`, `.+`)

	err := matcher(&http.Response{
		Body: io.NopCloser(bytes.NewBuffer([]byte(`{"anObject":{"aString":"lol"}}`))),
	}, nil)

	assert.EqualError(t, err, "unable to match using type: map")
}

func TestApiTestMatchesFailForArray(t *testing.T) {
	matcher := jsonpath.Matches(`$.aSlice`, `.+`)

	err := matcher(&http.Response{
		Body: io.NopCloser(bytes.NewBuffer([]byte(`{"aSlice":[1,2,3]}`))),
	}, nil)

	assert.EqualError(t, err, "unable to match using type: slice")
}

func TestApiTestMatchesFailForNilValue(t *testing.T) {
	matcher := jsonpath.Matches(`$.nothingHere`, `.+`)

	err := matcher(&http.Response{
		Body: io.NopCloser(bytes.NewBuffer([]byte(`{"aSlice":[1,2,3]}`))),
	}, nil)

	assert.EqualError(t, err, "no match for pattern: '$.nothingHere'")
}
