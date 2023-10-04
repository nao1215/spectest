[![LinuxUnitTest](https://github.com/nao1215/spectest/actions/workflows/linux_test.yml/badge.svg)](https://github.com/nao1215/spectest/actions/workflows/linux_test.yml)
[![MacUnitTest](https://github.com/nao1215/spectest/actions/workflows/mac_test.yml/badge.svg)](https://github.com/nao1215/spectest/actions/workflows/mac_test.yml)
[![Vuluncheck](https://github.com/nao1215/spectest/actions/workflows/govulncheck.yml/badge.svg)](https://github.com/nao1215/spectest/actions/workflows/govulncheck.yml)
[![Gosec](https://github.com/nao1215/spectest/actions/workflows/security.yml/badge.svg)](https://github.com/nao1215/spectest/actions/workflows/security.yml)
[![reviewdog](https://github.com/nao1215/spectest/actions/workflows/reviewdog.yml/badge.svg)](https://github.com/nao1215/spectest/actions/workflows/reviewdog.yml)
![Coverage](https://raw.githubusercontent.com/nao1215/octocovs-central-repo/main/badges/nao1215/spectest/coverage.svg)

## What is spectest?

**This project is forked from [steinfletcher/apitest](https://github.com/steinfletcher/apitest)** apitest was functionally complete. However, I wanted more features, so I decided to fork it to actively develop it further. I will mainly enhance document generation and integration with AWS."

A simple and extensible behavioral testing library. Supports mocking external http calls and renders sequence diagrams on completion.

In behavioral tests the internal structure of the app is not known by the tests. Data is input to the system and the outputs are expected to meet certain conditions.

## Supported OS
- Linux
- Mac

## Installation

```bash
go get -u github.com/nao1215/spectest
```

## Demo

![animated gif](./spectest.gif)

## Examples

### Framework and library integration examples

| Example                                                                                              | Comment                                                                                                    |
| ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| [gin](https://github.com/nao1215/apitest/tree/master/examples/gin)                             | popular martini-like web framework                                                                         |
| [graphql](https://github.com/nao1215/apitest/tree/master/examples/graphql)                     | using gqlgen.com to generate a graphql server                                                              |
| [gorilla](https://github.com/nao1215/apitest/tree/master/examples/gorilla)                     | the gorilla web toolkit                                                                                    |
| [iris](https://github.com/nao1215/apitest/tree/master/examples/iris)                           | iris web framework                                                                                         |
| [echo](https://github.com/nao1215/apitest/tree/master/examples/echo)                           | High performance, extensible, minimalist Go web framework                                                  |
| [fiber](https://github.com/nao1215/nao1215/tree/master/examples/fiber)                         | Express inspired web framework written in Go                                                               |
| [httprouter](https://github.com/nao1215/apitest/tree/master/examples/httprouter)               | High performance HTTP request router that scales well                                                      |
| [mocks](https://github.com/nao1215/apitest/tree/master/examples/mocks)                         | example mocking out external http calls                                                                    |
| [sequence diagrams](https://github.com/nao1215/apitest/tree/master/examples/sequence-diagrams) | generate sequence diagrams from tests |
| [Ginkgo](https://github.com/nao1215/apitest/tree/master/examples/ginkgo) | Ginkgo BDD test framework|

### Companion libraries

| Library                                                                 | Comment                                        |
| ----------------------------------------------------------------------- | -----------------------------------------------|
| [JSONPath](https://github.com/steinfletcher/apitest-jsonpath)           | JSONPath assertion addons                      |
| [CSS Selectors](https://github.com/steinfletcher/apitest-css-selector)  | CSS selector assertion addons                  |
| [PlantUML](https://github.com/steinfletcher/apitest-plantuml)           | Export sequence diagrams as plantUML           |
| [DynamoDB](https://github.com/steinfletcher/apitest-dynamodb)           | Add DynamoDB interactions to sequence diagrams |

### Credits

This library was influenced by the following software packages:

* [YatSpec](https://github.com/bodar/yatspec) for creating sequence diagrams from tests
* [MockMVC](https://spring.io) and [superagent](https://github.com/visionmedia/superagent) for the concept and behavioral testing approach
* [Gock](https://github.com/h2non/gock) for the approach to mocking HTTP services in Go
* [Baloo](https://github.com/h2non/baloo) for API design

### Code snippets

#### JSON body matcher

```go
func TestApi(t *testing.T) {
	apitest.New().
		Handler(handler).
		Get("/user/1234").
		Expect(t).
		Body(`{"id": "1234", "name": "Tate"}`).
		Status(http.StatusOK).
		End()
}
```

#### JSONPath

For asserting on parts of the response body JSONPath may be used. A separate module must be installed which provides these assertions - `go get -u github.com/steinfletcher/apitest-jsonpath`. This is packaged separately to keep this library dependency free.

Given the response is `{"a": 12345, "b": [{"key": "c", "value": "result"}]}`

```go
func TestApi(t *testing.T) {
	apitest.Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.Contains(`$.b[? @.key=="c"].value`, "result")).
		End()
}
```

and `jsonpath.Equals` checks for value equality

```go
func TestApi(t *testing.T) {
	apitest.Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.Equal(`$.a`, float64(12345))).
		End()
}
```

#### Custom assert functions

```go
func TestApi(t *testing.T) {
	apitest.Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(func(res *http.Response, req *http.Request) error {
			assert.Equal(t, http.StatusOK, res.StatusCode)
			return nil
		}).
		End()
}
```

#### Assert cookies

```go
func TestApi(t *testing.T) {
	apitest.Handler(handler).
		Patch("/hello").
		Expect(t).
		Status(http.StatusOK).
		Cookies(apitest.Cookie("ABC").Value("12345")).
		CookiePresent("Session-Token").
		CookieNotPresent("XXX").
		Cookies(
			apitest.Cookie("ABC").Value("12345"),
			apitest.Cookie("DEF").Value("67890"),
		).
		End()
}
```

#### Assert headers

```go
func TestApi(t *testing.T) {
	apitest.Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		Headers(map[string]string{"ABC": "12345"}).
		End()
}
```

#### Mocking external http calls

```go
var getUser = apitest.NewMock().
	Get("/user/12345").
	RespondWith().
	Body(`{"name": "jon", "id": "1234"}`).
	Status(http.StatusOK).
	End()

var getPreferences = apitest.NewMock().
	Get("/preferences/12345").
	RespondWith().
	Body(`{"is_contactable": true}`).
	Status(http.StatusOK).
	End()

func TestApi(t *testing.T) {
	apitest.New().
		Mocks(getUser, getPreferences).
		Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"name": "jon", "id": "1234"}`).
		End()
}
```

#### Generating sequence diagrams from tests

```go

func TestApi(t *testing.T) {
	apitest.New().
		Report(apitest.SequenceDiagram()).
		Mocks(getUser, getPreferences).
		Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"name": "jon", "id": "1234"}`).
		End()
}
```

It is possible to override the default storage location by passing the formatter instance `Report(apitest.NewSequenceDiagramFormatter(".sequence-diagrams"))`.
You can bring your own formatter too if you want to produce custom output. By default a sequence diagram is rendered on a html page. See the [demo](http://demo-html.apitest.dev.s3-website-eu-west-1.amazonaws.com/)

#### Debugging http requests and responses generated by api test and any mocks

```go
func TestApi(t *testing.T) {
	apitest.New().
		Debug().
		Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

#### Provide basic auth in the request

```go
func TestApi(t *testing.T) {
	apitest.Handler(handler).
		Get("/hello").
		BasicAuth("username", "password").
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

#### Pass a custom context to the request

```go
func TestApi(t *testing.T) {
	apitest.Handler(handler).
		Get("/hello").
		WithContext(context.TODO()).
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

#### Provide cookies in the request

```go
func TestApi(t *testing.T) {
	apitest.Handler(handler).
		Get("/hello").
		Cookies(apitest.Cookie("ABC").Value("12345")).
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

#### Provide headers in the request

```go
func TestApi(t *testing.T) {
	apitest.Handler(handler).
		Delete("/hello").
		Headers(map[string]string{"My-Header": "12345"}).
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

#### Provide query parameters in the request

`Query`, `QueryParams` and `QueryCollection` can all be used in combination 

```go
func TestApi(t *testing.T) {
	apitest.Handler(handler).
		Get("/hello").
		QueryParams(map[string]string{"a": "1", "b": "2"}).
		Query("c", "d").
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

Providing `{"a": {"b", "c", "d"}` results in parameters encoded as `a=b&a=c&a=d`.
`QueryCollection` can be used in combination with `Query`

```go
func TestApi(t *testing.T) {
	apitest.Handler(handler).
		Get("/hello").
		QueryCollection(map[string][]string{"a": {"b", "c", "d"}}).
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

#### Provide a url encoded form body in the request

```go
func TestApi(t *testing.T) {
	apitest.Handler(handler).
		Post("/hello").
		FormData("a", "1").
		FormData("b", "2").
		FormData("b", "3").
		FormData("c", "4", "5", "6").
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

#### Provide a multipart/form-data

```go
func TestApi(t *testing.T) {
	apitest.Handler(handler).
		Post("/hello").
		MultipartFormData("a", "1", "2").
		MultipartFile("file", "path/to/some.file1", "path/to/some.file2").
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

#### Capture the request and response data

```go
func TestApi(t *testing.T) {
	apitest.New().
		Observe(func(res *http.Response, req *http.Request, apiTest *apitest.APITest) {
			// do something with res and req
		}).
		Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

#### Intercept the request

This is useful for mutating the request before it is sent to the system under test.

```go
func TestApi(t *testing.T) {
	apitest.Handler(handler).
		Intercept(func(req *http.Request) {
			req.URL.RawQuery = "a[]=xxx&a[]=yyy"
		}).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

## Contributing

View the [contributing guide](CONTRIBUTING.md).
