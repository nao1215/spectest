package apitest_test

import (
	"fmt"
	"github.com/steinfletcher/apitest"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestApiTest_AddsJSONBodyToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
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

	apitest.New().
		Handler(handler).
		Post("/hello").
		Body(`{"a": 12345}`).
		Header("Content-Type", "application/json").
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsJSONBodyToRequestUsingJSON(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
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

	apitest.New().
		Handler(handler).
		Post("/hello").
		JSON(`{"a": 12345}`).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsTextBodyToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		if string(data) != `hello` {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	apitest.New().
		Handler(handler).
		Put("/hello").
		Body(`hello`).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsQueryParamsToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if "b" != r.URL.Query().Get("a") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	apitest.New().
		Handler(handler).
		Get("/hello").
		QueryParams(map[string]string{"a": "b"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsQueryParamCollectionToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if "a=b&a=c&a=d&e=f" != r.URL.RawQuery {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	apitest.New().
		Handler(handler).
		Get("/hello").
		QueryCollection(map[string][]string{"a": {"b", "c", "d"}}).
		QueryParams(map[string]string{"e": "f"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsQueryParamCollectionToRequest_HandlesEmpty(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if "e=f" != r.URL.RawQuery {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	apitest.New().
		Handler(handler).
		Get("/hello").
		QueryCollection(map[string][]string{}).
		QueryParams(map[string]string{"e": "f"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_CanCombineQueryParamMethods(t *testing.T) {
	expectedQueryString := "a=1&a=2&a=9&a=22&b=2"
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if expectedQueryString != r.URL.RawQuery {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	apitest.New().
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

func TestApiTest_AddsHeadersToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		header := r.Header["Authorization"]
		if len(header) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	apitest.New().
		Handler(handler).
		Delete("/hello").
		Headers(map[string]string{"Authorization": "12345"}).
		Header("Authorization", "098765").
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsContentTypeHeaderToRequest(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.Header["Content-Type"][0] != "application/x-www-form-urlencoded" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	apitest.New().
		Handler(handler).
		Post("/hello").
		ContentType("application/x-www-form-urlencoded").
		Body(`name=John`).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsCookiesToRequest(t *testing.T) {
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

	apitest.New().
		Handler(handler).
		Method(http.MethodGet).
		URL("/hello").
		Cookie("Cookie", "Nom").
		Cookies(apitest.NewCookie("Cookie1").Value("Yummy")).
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_AddsBasicAuthToRequest(t *testing.T) {
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

	apitest.New("some test name").
		Handler(handler).
		Get("/hello").
		BasicAuth("username", "password").
		Expect(t).
		Status(http.StatusOK).
		End()
}

func TestApiTest_MatchesJSONResponseBody(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"a": 12345}`))
		if err != nil {
			panic(err)
		}
	})

	apitest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Body(`{"a": 12345}`).
		Status(http.StatusCreated).
		End()
}

func TestApiTest_MatchesJSONResponseBodyWithWhitespace(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"a": 12345, "b": "hi"}`))
		if err != nil {
			panic(err)
		}
	})

	apitest.New().
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

func TestApiTest_MatchesTextResponseBody(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		_, err := w.Write([]byte(`hello`))
		if err != nil {
			panic(err)
		}
	})

	apitest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Body(`hello`).
		Status(http.StatusOK).
		End()
}

func TestApiTest_MatchesResponseCookies(t *testing.T) {
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

	apitest.New().
		Handler(handler).
		Patch("/hello").
		Expect(t).
		Status(http.StatusOK).
		Cookies(
			apitest.NewCookie("ABC").Value("12345"),
			apitest.NewCookie("DEF").Value("67890")).
		Cookie("YYY", "kfiufhtne").
		CookiePresent("XXX").
		CookiePresent("VVV").
		CookieNotPresent("ZZZ").
		CookieNotPresent("TomBeer").
		End()
}

func TestApiTest_MatchesResponseHttpCookies(t *testing.T) {
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

	apitest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Cookies(
			apitest.NewCookie("ABC").Value("12345"),
			apitest.NewCookie("DEF").Value("67890")).
		End()
}

func TestApiTest_MatchesResponseHttpCookies_OnlySuppliedFields(t *testing.T) {
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

	apitest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Cookies(
			apitest.NewCookie("session_id").
				Value("pdsanjdna_8e8922").
				Path("/").
				Expires(parsedDateTime).
				Secure(true).
				HttpOnly(true)).
		End()
}

func TestApiTest_MatchesResponseHeaders_WithMixedKeyCase(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ABC", "12345")
		w.Header().Set("DEF", "67890")
		w.Header().Set("Authorization", "12345")
		w.Header().Add("authorizATION", "00000")
		w.Header().Add("Authorization", "98765")
		w.WriteHeader(http.StatusOK)
	})

	apitest.New().
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

func TestApiTest_EndReturnsTheResult(t *testing.T) {
	type resBody struct {
		B string `json:"b"`
	}
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"a": 12345, "b": "hi"}`))
		if err != nil {
			panic(err)
		}
	})

	var r resBody
	apitest.New().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Body(`{
			"a": 12345,
			"b": "hi"
		}`).
		Status(http.StatusCreated).
		End().
		JSON(&r)

	assert.Equal(t, "hi", r.B)
}

func TestApiTest_CustomAssert(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-ExpectedCookie", "ABC=12345; DEF=67890; XXX=1fsadg235; VVV=9ig32g34g")
		w.WriteHeader(http.StatusOK)
	})

	apitest.New().
		Handler(handler).
		Patch("/hello").
		Expect(t).
		Assert(apitest.IsSuccess).
		End()
}

func TestApiTest_Report(t *testing.T) {
	getUser := apitest.NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("1").
		Times(2).
		End()

	reporter := &RecorderCaptor{}

	apitest.New("some test").
		Debug().
		Meta(map[string]interface{}{"host": "abc.com"}).
		Report(reporter).
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
	assert.Equal(t, "POST /hello", r.Title)
	assert.Equal(t, "some test", r.SubTitle)
	assert.Len(t, r.Events, 4)
	assert.Equal(t, 200, r.Meta["status_code"])
	assert.Equal(t, "/hello", r.Meta["path"])
	assert.Equal(t, "POST", r.Meta["method"])
	assert.Equal(t, "some test", r.Meta["name"])
	assert.Equal(t, "abc.com", r.Meta["host"])
	assert.NotEmpty(t, r.Meta["duration"])
}

func TestApiTest_Recorder(t *testing.T) {
	getUser := apitest.NewMock().
		Get("http://localhost:8080").
		RespondWith().
		Status(http.StatusOK).
		Body("1").
		Times(2).
		End()

	reporter := &RecorderCaptor{}
	messageRequest := apitest.MessageRequest{
		Source:    "Source",
		Target:    "Target",
		Header:    "Header",
		Body:      "Body",
		Timestamp: time.Now().UTC(),
	}
	messageResponse := apitest.MessageResponse{
		Source:    "Source",
		Target:    "Target",
		Header:    "Header",
		Body:      "Body",
		Timestamp: time.Now().UTC(),
	}
	recorder := apitest.NewTestRecorder()
	recorder.AddMessageRequest(messageRequest)
	recorder.AddMessageResponse(messageResponse)

	apitest.New("some test").
		Meta(map[string]interface{}{"host": "abc.com"}).
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
	assert.Len(t, r.Events, 6)
	assert.Equal(t, messageRequest, r.Events[0])
	assert.Equal(t, messageResponse, r.Events[1])
}

func TestApiTest_Observe(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	observeCalled := false

	apitest.New("observe test").
		Observe(func(res *http.Response, req *http.Request, apiTest *apitest.APITest) {
			observeCalled = true
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "/hello", req.URL.Path)
		}).
		Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		End()

	assert.True(t, observeCalled)
}

func TestApiTest_Observe_DumpsTheHttpRequestAndResponse(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"a": 12345}`))
		if err != nil {
			panic(err)
		}
	})

	apitest.New().
		Handler(handler).
		Post("/hello").
		Body(`{"a": 12345}`).
		Headers(map[string]string{"Content-Type": "application/json"}).
		Expect(t).
		Status(http.StatusCreated).
		End()
}

func TestApiTest_ObserveWithReport(t *testing.T) {
	reporter := &RecorderCaptor{}
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	observeCalled := false

	apitest.New("observe test").
		Report(reporter).
		Observe(func(res *http.Response, req *http.Request, apiTest *apitest.APITest) {
			observeCalled = true
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "/hello", req.URL.Path)
		}).
		Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		End()

	assert.True(t, observeCalled)
}

func TestApiTest_Intercept(t *testing.T) {
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

	apitest.New().
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

func TestApiTest_ExposesRequestAndResponse(t *testing.T) {
	apiTest := apitest.New()

	assert.NotNil(t, apiTest.Request())
	assert.NotNil(t, apiTest.Response())
}

// TestRealNetworking creates a server with two endpoints, /login sets a token via a cookie and /authenticated_resource
// validates the token. A cookie jar is used to verify session persistence across multiple apitest instances
func TestRealNetworking(t *testing.T) {
	srv := &http.Server{Addr: ":9876"}
	finish := make(chan struct{})
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

		apitest.New().
			EnableNetworking(cli).
			Get("http://localhost:9876/login").
			Expect(t).
			Status(203).
			End()

		apitest.New().
			EnableNetworking(cli).
			Get("http://localhost:9876/authenticated_resource").
			Expect(t).
			Status(204).
			End()

		finish <- struct{}{}
	}()
	<-finish
}

func TestApiTest_AddsUrlEncodedFormBody(t *testing.T) {
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

	apitest.New().
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

type RecorderCaptor struct {
	capturedRecorder apitest.Recorder
}

func (r *RecorderCaptor) Format(recorder *apitest.Recorder) {
	r.capturedRecorder = *recorder
}

func getUserData() []byte {
	res, err := http.Get("http://localhost:8080")
	if err != nil {
		panic(err)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	return data
}
