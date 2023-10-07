## jsonschema validates the http response body against the provided json schema.
```go

const schema = `{
	"$id": "https://example.com/person.schema.json",
	"$schema": "https://json-schema.org/draft/2020-12/schema",
	"title": "Person",
	"type": "object",
	"required": [ "firstName", "lastName", "age" ],
	"properties": {
	  "firstName": {
		"type": "string",
		"description": "The person's first name."
	  },
	  "lastName": {
		"type": "string",
		"description": "The person's last name."
	  },
	  "age": {
		"description": "Age in years which must be equal to or greater than zero.",
		"type": "integer",
		"minimum": 0
	  }
	}
  }`

func TestValidateMatchesSchema(t *testing.T) {
	spectest.New().
		HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			_, _ = writer.Write([]byte(`{
			  "firstName": "John",
			  "lastName": "Doe",
			  "age": 21
			}`))
			writer.WriteHeader(http.StatusOK)
		}).
		Get("/").
		Expect(t).
		Status(http.StatusOK).
		Assert(jsonschema.Validate(schema)).
		End()
}
```

## Supported OS
- Linux
- Mac
- Windows

## LICENSE
MIT License

This repository is forked from [steinfletcher/apitest-jsonschema](https://github.com/steinfletcher/apitest-jsonschema).
