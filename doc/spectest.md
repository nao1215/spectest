## Use Cases of spectest
The spectest offers numerous features that are not available in its forked counterpart, [steinfletcher/apitest](https://github.com/steinfletcher/apitest). While apitest was a simple library, spectest provides functionalities both as a library and a CLI (Command Line Interface).
  
The current version of spectest is gradually upgrading its documentation generation capabilities. Instead of generating HTML documents, it aims to preserve End-to-End (E2E) test results as documents in Markdown format for developers working on GitHub.
  
### Simple usecase for generating test results
1. You create unit tests for your API endpoints with spectest.
2. You run the tests and auto generate Markdown documents with the test results.
3. When you run `spectest index`, spectest generates an index markdown for a directory full of markdown file.
  
#### Test code example
For example, let's consider an API that returns the health check of a server (GET /v1/health). The code to test the status code and body of this API is as follows:
  
```go
func TestHealthCheck(t *testing.T) {
	spectest.New().
        Report(spectest.SequenceReport(spectest.ReportFormatterConfig{
			Path: filepath.Join("docs", "health"),
			Kind: spectest.ReportKindMarkdown,
		})).
		CustomReportName("health_success").
		Handler(api).
		Get("/v1/health").
		Expect(t).
		Body(`{"name": "naraku", "revision": "revision-0.0.1", "version":"v0.0.1"}`).
		Status(http.StatusOK).
		End()
}
```

In spectest.SequenceReport, you specify the "output destination for E2E test results" and indicate that the results should be output in Markdown format. With CustomReportName, you specify the filename for the Markdown file. The above code will generate a Markdown file named "docs/health/health_success.md".

#### Markdown file example
![health_result](./image/health_result.png)

#### Index file example
When you write numerous unit tests, it results in the generation of multiple Markdown files containing E2E test results. It can become challenging for you to reference all these Markdown files individually. Therefore, you can use the spectest CLI to generate an index file with links to the Markdown files.

You execute the following command to generate an index file:
```shell
spectest index docs --title "naraku api result" 
```

[Output](https://github.com/go-spectest/naraku/blob/main/docs/index.md):  
![index_result](./image/index.png)

## Installation of spectest cli
```shell
go install github.com/go-spectest/spectest/cmd/spectest@latest
```

