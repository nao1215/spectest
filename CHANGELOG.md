# Changelog

## [0.0.2, Unreleased] - 2023-XX-XX

### Added
- GitHub Actions
  - Unit Test for Linux/macOS/windows, reviewdog, golangci-lint
- Dependabot and patch/minor version auto update.
- Import side project; jsonpath, jsonschema, plantuml, css-selector
  - aws package is not imported. It's broken.
- `APITest.CustomReportName()`
  - Output sequence diagram with custom name instead of hash.
- Add help target to Makefile.

### Fixed
- Refactoring
  - Refactoring for internal code. For example, meta information is now stored in a struct instead of a map.

### Changed
- Rename 'Http' to 'HTTP'. So, some function names are changed.
  - `Http` -> `HTTP`
  - `HttpHandler` -> `HTTPHandler`
- deprecated method in io/ioutil.

- Export difflib. 
  - I exported difflib as an external package (diff) because it is an independent package. Additionally, I made the diff package compatible with Windows. Specifically, it now displays differences in color even when indicating diffs in a Windows environment.

### Removed
- `APITest.Meta()` method.
  - The precise information of the meta (map[string]interface{}) was not being exposed to users. Consequently, users found it challenging to effectively utilize APITest.Meta(). Instead of providing an alternative method, APITest.Meta() was removed.