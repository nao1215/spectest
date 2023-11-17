[![Go Reference](https://pkg.go.dev/badge/github.com/go-spectest/spectest.svg)](https://pkg.go.dev/github.com/go-spectest/spectest)
[![LinuxUnitTest](https://github.com/go-spectest/spectest/actions/workflows/linux_test.yml/badge.svg)](https://github.com/go-spectest/spectest/actions/workflows/linux_test.yml)
[![MacUnitTest](https://github.com/go-spectest/spectest/actions/workflows/mac_test.yml/badge.svg)](https://github.com/go-spectest/spectest/actions/workflows/mac_test.yml)
[![WindowsUnitTest](https://github.com/go-spectest/spectest/actions/workflows/windows_test.yml/badge.svg)](https://github.com/go-spectest/spectest/actions/workflows/windows_test.yml)
[![UnitTestExampleCodes](https://github.com/go-spectest/spectest/actions/workflows/test-examples.yml/badge.svg)](https://github.com/go-spectest/spectest/actions/workflows/test-examples.yml)
[![reviewdog](https://github.com/go-spectest/spectest/actions/workflows/reviewdog.yml/badge.svg)](https://github.com/go-spectest/spectest/actions/workflows/reviewdog.yml)
[![Gosec](https://github.com/go-spectest/spectest/actions/workflows/gosec.yml/badge.svg)](https://github.com/go-spectest/spectest/actions/workflows/gosec.yml)
![Coverage](https://github.com/go-spectest/octocovs-central-repo/blob/main//badges/go-spectest/spectest/coverage.svg?raw=true)

## What is spectest?

A simple and extensible behavioral testing library. Supports mocking external http calls and renders sequence diagrams on completion. In behavioral tests the internal structure of the app is not known by the tests. Data is input to the system and the outputs are expected to meet certain conditions.

**This project is forked from [steinfletcher/apitest](https://github.com/steinfletcher/apitest)** apitest was functionally complete. However, I wanted more features, so I decided to fork it to actively develop it further. I will mainly enhance document generation and integration with AWS. There are no plans for compatibility between apitest and spectest. Therefore, future development will include BREAKING CHANGES. 

The spectest has its own unique use cases, [Use Cases of spectest](./doc/spectest.md). Please refer to this document for more information.

## Supported OS
- Linux
- Mac
- Windows (Original [apitest](https://github.com/steinfletcher/apitest) does not support Windows)

## Installation

```bash
go get -u github.com/go-spectest/spectest
```

## Demo

![animated gif](./spectest.gif)

## Examples

### Framework and library integration examples

| Example                                                                                              | Comment                                                                                                    |
| ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| [gin](https://github.com/go-spectest/spectest/tree/main/examples/gin)                             | popular martini-like web framework                                                                         |
| [graphql](https://github.com/go-spectest/spectest/tree/main/examples/graphql)                     | using gqlgen.com to generate a graphql server                                                              |
| [gorilla](https://github.com/go-spectest/spectest/tree/main/examples/gorilla)                     | the gorilla web toolkit                                                                                    |
| [iris (broken)](https://github.com/go-spectest/spectest/tree/main/examples/iris)                           | iris web framework                                                                                         |
| [echo](https://github.com/go-spectest/spectest/tree/main/examples/echo)                           | High performance, extensible, minimalist Go web framework                                                  |
| [fiber](https://github.com/go-spectest/spectest/tree/main/examples/fiber)                         | Express inspired web framework written in Go                                                               |
| [httprouter](https://github.com/go-spectest/spectest/tree/main/examples/httprouter)               | High performance HTTP request router that scales well                                                      |
| [mocks](https://github.com/go-spectest/spectest/tree/main/examples/mocks)                         | example mocking out external http calls                                                                    |
| [sequence diagrams](https://github.com/go-spectest/spectest/tree/main/examples/sequence-diagrams) | generate sequence diagrams from tests |
| [Ginkgo](https://github.com/go-spectest/spectest/tree/main/examples/ginkgo) | Ginkgo BDD test framework|
| [plantuml](https://github.com/go-spectest/spectest/tree/main/examples/plantuml) | wxample generating plantuml|

### Companion libraries (Side projects)
In the original apitest repository, side projects were managed in separate repositories. However, in spectest, these side projects are managed within the same repository. However, the difflib, which operates independently from spectest, and the malfunctioning aws package, are managed in separate repositories.
  
| Library                                                                 | Comment                                        |
| ----------------------------------------------------------------------- | -----------------------------------------------|
| [JSON Path](https://github.com/go-spectest/spectest/tree/main/jsonpath)           | JSON Path assertion addons                      |
| [JOSN Schema](https://github.com/go-spectest/spectest/tree/main/jsonschema)               | JSON Schema assertion addons |
| [CSS Selectors](https://github.com/go-spectest/spectest/tree/main/css-selector)  | CSS selector assertion addons                  |
| [PlantUML](https://github.com/go-spectest/spectest/tree/main/plantuml)           | Export sequence diagrams as plantUML           |
| [DynamoDB (broken)](https://github.com/go-spectest/tree/main/aws)           | Add DynamoDB interactions to sequence diagrams |

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
	spectest.New().
		Handler(handler).
		Get("/user/1234").
		Expect(t).
		Body(`{"id": "1234", "name": "Tate"}`).
		Status(http.StatusOK).
		End()
}
```

#### JSONPath

For asserting on parts of the response body JSONPath may be used. A separate module must be installed which provides these assertions - `go get -u github.com/go-spectest/spectest/jsonpath`. This is packaged separately to keep this library dependency free.

Given the response is `{"a": 12345, "b": [{"key": "c", "value": "result"}]}`

```go
func TestApi(t *testing.T) {
	spectest.Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.Contains(`$.b[? @.key=="c"].value`, "result")).
		End()
}
```

and `jsonpath.Equals` checks for value equality

```go
func TestApi(t *testing.T) {
	spectest.Handler(handler).
		Get("/hello").
		Expect(t).
		Assert(jsonpath.Equal(`$.a`, float64(12345))).
		End()
}
```

#### Custom assert functions

```go
func TestApi(t *testing.T) {
	spectest.Handler(handler).
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
	spectest.Handler(handler).
		Patch("/hello").
		Expect(t).
		Status(http.StatusOK).
		Cookies(spectest.Cookie("ABC").Value("12345")).
		CookiePresent("Session-Token").
		CookieNotPresent("XXX").
		Cookies(
			spectest.Cookie("ABC").Value("12345"),
			spectest.Cookie("DEF").Value("67890"),
		).
		End()
}
```

#### Assert headers

```go
func TestApi(t *testing.T) {
	spectest.Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		Headers(map[string]string{"ABC": "12345"}).
		End()
}
```

#### Mocking external http calls

```go
var getUser = spectest.NewMock().
	Get("/user/12345").
	RespondWith().
	Body(`{"name": "jon", "id": "1234"}`).
	Status(http.StatusOK).
	End()

var getPreferences = spectest.NewMock().
	Get("/preferences/12345").
	RespondWith().
	Body(`{"is_contactable": true}`).
	Status(http.StatusOK).
	End()

func TestApi(t *testing.T) {
	spectest.New().
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
	spectest.New().
		Report(spectest.SequenceDiagram()).
		Mocks(getUser, getPreferences).
		Handler(handler).
		Get("/hello").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"name": "jon", "id": "1234"}`).
		End()
}
```

It is possible to override the default storage location by passing the formatter instance `Report(spectest.NewSequenceDiagramFormatter(".sequence-diagrams"))`. If you want to change the report file name , you use `CustomReportName("file name is here")` . By default, the hash value becomes the report file name. 

You can bring your own formatter too if you want to produce custom output. By default a sequence diagram is rendered on a html page.

The spectest checks the Content Type of the response. If it's an image-related MIME type, the image will be displayed in the report. In  the apitest, binary data was being displayed.

![DiagramSample](doc/image/sample_diagram.png)


One feature that does not exist in the apitest fork is the ability to output reports in markdown format. The below code snippet is an example of how to output a markdown report.

```go

func TestApi(t *testing.T) {
	spectest.New().
		CustomReportName("markdow_report").
		Report(spectest.SequenceReport(spectest.ReportFormatterConfig{
			Path: "doc",
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
}
```

![MarkdownReportSample](doc/image/markdown_report.png)

#### Debugging http requests and responses generated by api test and any mocks

```go
func TestApi(t *testing.T) {
	spectest.New().
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
	spectest.Handler(handler).
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
	spectest.Handler(handler).
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
	spectest.Handler(handler).
		Get("/hello").
		Cookies(spectest.Cookie("ABC").Value("12345")).
		Expect(t).
		Status(http.StatusOK).
		End()
}
```

#### Provide headers in the request

```go
func TestApi(t *testing.T) {
	spectest.Handler(handler).
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
	spectest.Handler(handler).
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
	spectest.Handler(handler).
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
	spectest.Handler(handler).
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
	spectest.Handler(handler).
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
	spectest.New().
		Observe(func(res *http.Response, req *http.Request, specTest *spectest.SpecTest) {
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
	spectest.Handler(handler).
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


