// Package version manage spectest command version
package version

const (
	// CommandName is cli command name
	CommandName = "spectest" //nolint
)

var (
	// TagVersion value is set by ldflags
	TagVersion string //nolint
	// Revision value is set by ldflagss
	Revision string //nolint
)
