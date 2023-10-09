package spectest

import (
	"fmt"
	"strings"
)

func debugLog(prefix, header, msg string) {
	fmt.Printf("\n%s %s\n%s\n", prefix, header, msg)
}

func requestDebugPrefix() string {
	return fmt.Sprintf("%s>", strings.Repeat("-", 10))
}

func responseDebugPrefix() string {
	return fmt.Sprintf("<%s", strings.Repeat("-", 10))
}
