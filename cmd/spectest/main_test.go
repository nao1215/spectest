// Package main is a package that contains subcommands for the spectest CLI command.
package main

import (
	"os"
	"testing"
)

func Test_main(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		exitCode int
	}{
		{
			name:     "no args",
			args:     []string{},
			exitCode: 0,
		},
		{
			name:     "version",
			args:     []string{"version"},
			exitCode: 0,
		},
		{
			name:     "help",
			args:     []string{"help"},
			exitCode: 0,
		},
		{
			name:     "help version",
			args:     []string{"help", "version"},
			exitCode: 0,
		},
		{
			name:     "help help",
			args:     []string{"help", "help"},
			exitCode: 0,
		},
		{
			name:     "help unknown",
			args:     []string{"help", "unknown"},
			exitCode: 0,
		},
		{
			name:     "unknown",
			args:     []string{"unknown"},
			exitCode: 1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {

			osExit = func(code int) {
				if code != tt.exitCode {
					t.Errorf("osExit() = %v, want %v", code, tt.exitCode)
				}
			}

			os.Args = append([]string{"spectest"}, tt.args...)

			main()
		})
	}
}
