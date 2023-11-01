// Package main is a package that contains subcommands for the spectest CLI command.
package main

import (
	"os"

	"github.com/go-spectest/spectest/cmd/lion"
)

// osExit is wrapper for  os.Exit(). It's for unit test.
var osExit = os.Exit //nolint

func main() {
	osExit(lion.Execute())
}
