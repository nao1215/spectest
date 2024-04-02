# css-selector

Assertions for [spectest](https://github.com/nao1215/spectest) using css selectors.

## Examples

### `selector.TextExists`

```go
spectest.New().
	HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
		<head>
			<title>My document</title>
		</head>
		<body>
		<h1>Header</h1>
		<p>Some text to match on</p>
		</body>
		</html>`,
		))
		w.WriteHeader(http.StatusOK)
	}).
	Get("/").
	Expect(t).
	Status(http.StatusOK).
	Assert(selector.TextExists("Some text to match on")).
	End()
```

### `selector.ContainsTextValue`

If you are selecting a data test id, a convenience method is provided to simplify the query.

```go
spectest.New().
	HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
	<head>
		<title>My document</title>
	</head>
	<body>
	<div data-test-id="some-test-id">
		<div>some content</div>
	</div>
	</body>
	</html>`,
		))
		w.WriteHeader(http.StatusOK)
	}).
	Get("/").
	Expect(t).
	Status(http.StatusOK).
	Assert(selector.ContainsTextValue(selector.DataTestID("some-test-id"), "some content")).
	End()
```

### `selector.FirstTextValue`

```go
spectest.New().
	Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<div class="myClass">content</div>`))
		w.WriteHeader(http.StatusOK)
	})).
	Get("/").
	Expect(t).
	Status(http.StatusOK).
	Assert(selector.FirstTextValue(`.myClass`, "content")).
	End()
```

see also `selector.NthTextValue` and `selector.ContainsTextValue`

### `selector.Exists` `selector.NotExists`

```go
spectest.New().
	Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(
	`<div>
		<div class="myClass">text</div>
		<div id="myId">text</div>
		<div data-test-id="product-5">text</div>
	</div>`))
		w.WriteHeader(http.StatusOK)
	})).
	Get("/").
	Expect(t).
	Status(http.StatusOK).
	Assert(selector.Exists(".myClass", `div[data-test-id^="product-"]`, "#myId")).
	Assert(selector.NotExists("#notExists")).
	End()
```

### `selector.Selection`

This exposes `goquery`'s Selection api and offers more flexibility over the previous methods

```go
Assert(selector.Selection(".outerClass", func(selection *goquery.Selection) error {
	if test.expectedText != selection.Find(".innerClass").Text() {
	    return fmt.Errorf("text did not match")
	}
	return nil
})).
```

## LICENSE
MIT LICESE

This repository is forked from [steinfletcher/apitest-css-selector](https://github.com/steinfletcher/apitest-css-selector).
