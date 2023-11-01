// Package version manage command version
package version

const (
	// CommandName is cli command name
	CommandName = "lion" //nolint
)

var (
	// TagVersion value is set by ldflags
	TagVersion string //nolint
	// Revision value is set by ldflagss
	Revision string //nolint
)
