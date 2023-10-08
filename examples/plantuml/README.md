## Example

```go
apitest.New("search user").
    Handler(myHandler).
	Report(plantuml.NewFormatter(myWriter)).
	Mocks(getPreferencesMock, getUserMock).
	Post("/user/search").
	Body(`{"name": "jon"}`).
	Expect(t).
	Status(http.StatusOK).
	Header("Content-Type", "application/json").
	Body(`{"name": "jon", "is_contactable": true}`).
	End()
```

![Diagram](./testdata/plantuml.png?raw=true "Sequence Diagram")
