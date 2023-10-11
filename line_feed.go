package spectest

import "runtime"

// lineFeed return line feed for current OS.
func lineFeed() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}
