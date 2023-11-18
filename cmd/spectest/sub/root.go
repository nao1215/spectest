// Package sub is spectest sub-commands.
package sub

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/fatih/color"
	"github.com/go-spectest/spectest"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

// Execute runs the process.
func Execute() int {
	rootCmd := newRootCmd()

	// Workaround: The root command of spectest functions as a wrapper for 'go test'.
	// I want the arguments provided to spectest to be passed directly to the go command.
	// However, spf13/cobra parses the arguments and throws an error stating "unknown command"
	// if it is encountered. Therefore, if an unknown command is found, I want to forcibly
	// execute the root command.
	if _, _, err := rootCmd.Find(os.Args[1:]); err != nil {
		if strings.HasPrefix(err.Error(), "unknown command") && !strings.Contains(err.Error(), "help") && !strings.Contains(err.Error(), "version") {
			if err := root(rootCmd, os.Args[1:]); err != nil {
				fmt.Fprintf(os.Stderr, "%s", err.Error())
				return 1
			}
			return 0
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		return 1
	}
	return 0
}

// newRootCmd returns a root command.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spectest",
		Short: "spectest is a tool for unit test.",
		Long: `The spectest command provides utility for unit testing, not only API test.
By default, spectest command is a wrapper for 'go test' command.`,
	}
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.DisableFlagParsing = true

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newBugReportCmd())
	cmd.AddCommand(newIndexCmd())
	return cmd
}

// root is a root command.
func root(cmd *cobra.Command, args []string) error {
	return newSpectester(cmd, args).run()
}

// TestStats holds the test statistics.
type TestStats struct {
	// Pass is the number of passed tests.
	Pass int32
	// Fail is the number of failed tests.
	Fail int32
	// Skip is the number of skipped tests.
	Skip int32
	// Total is the number of total tests.
	Total int32
}

// spectester is a struct for spectest command.
type spectester struct {
	args            []string
	stats           TestStats
	allTestMessages []string
	interval        *spectest.Interval
}

// newSpectester returns a spectester.
func newSpectester(_ *cobra.Command, args []string) *spectester {
	return &spectester{
		args:            args,
		stats:           TestStats{},
		allTestMessages: []string{},
		interval:        spectest.NewInterval(),
	}
}

// run runs the spectest command.
func (s *spectester) run() error {
	if err := s.canUseGoCommand(); err != nil {
		return fmt.Errorf("spectest command requires go command. please install go command")
	}
	return s.runTest()
}

// canUseGoCommand returns true if go command is available.
func (s *spectester) canUseGoCommand() error {
	_, err := exec.LookPath("go")
	return err
}

// runTest runs the test command.
func (s *spectester) runTest() error {
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()

	r, w := io.Pipe()
	defer w.Close() //nolint

	args := append([]string{"test"}, s.args...)
	if !slices.Contains(args, "-v") {
		args = append(args, "-v") // This option is required to count the number of tests.
	}

	cmd := exec.Command("go", args...) //nolint
	cmd.Stderr = w
	cmd.Stdout = w
	cmd.Env = os.Environ()

	s.interval.Start()
	if err := cmd.Start(); err != nil {
		wg.Done()
		return err
	}

	go s.consume(&wg, r)
	defer func() {
		s.interval.End()
		s.testResult()
	}()

	sigc := make(chan os.Signal, 1)
	done := make(chan struct{})
	defer func() {
		done <- struct{}{}
	}()
	signal.Notify(sigc)

	go func() {
		for {
			select {
			case sig := <-sigc:
				if err := cmd.Process.Signal(sig); err != nil {
					if errors.Is(err, os.ErrProcessDone) {
						break
					}
					fmt.Fprintf(os.Stderr, "failed to send signal: %s", err.Error())
				}
			case <-done:
				return
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		if _, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			return nil
		}
		return err
	}

	return nil
}

// consume consumes the output of the test command.
func (s *spectester) consume(wg *sync.WaitGroup, r io.Reader) {
	defer wg.Done()
	reader := bufio.NewReader(r)
	for {
		l, _, err := reader.ReadLine()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Print(err)
			return
		}
		s.parse(string(l))
	}
}

// parse parses a line of test output. It updates the test statistics.
func (s *spectester) parse(line string) {
	trimmed := strings.TrimSpace(line)

	switch {
	case strings.HasPrefix(trimmed, "ok"):
		fallthrough
	case strings.HasPrefix(trimmed, "FAIL"):
		fallthrough
	case strings.HasPrefix(trimmed, "PASS"):
		fallthrough
	case strings.Contains(trimmed, "[no test files]"):
		return

	case strings.HasPrefix(trimmed, "=== RUN"):
		fallthrough
	case strings.HasPrefix(trimmed, "=== CONT"):
		fallthrough
	case strings.HasPrefix(trimmed, "=== PAUSE"):
		s.allTestMessages = append(s.allTestMessages, line)
		return

	// passed
	case strings.HasPrefix(trimmed, "--- PASS"):
		fmt.Fprint(os.Stdout, color.GreenString("."))
		atomic.AddInt32(&s.stats.Pass, 1)
		atomic.StoreInt32(&s.stats.Total, atomic.AddInt32(&s.stats.Total, 1))
		s.allTestMessages = append(s.allTestMessages, line)

	// skipped
	case strings.HasPrefix(trimmed, "--- SKIP"):
		fmt.Fprint(os.Stdout, color.BlueString("."))
		atomic.AddInt32(&s.stats.Skip, 1)
		atomic.StoreInt32(&s.stats.Total, atomic.AddInt32(&s.stats.Total, 1))
		s.allTestMessages = append(s.allTestMessages, line)

	// failed
	case strings.HasPrefix(trimmed, "--- FAIL"):
		fmt.Fprint(os.Stdout, color.RedString("."))
		atomic.AddInt32(&s.stats.Fail, 1)
		atomic.StoreInt32(&s.stats.Total, atomic.AddInt32(&s.stats.Total, 1))
		s.allTestMessages = append(s.allTestMessages, line)

	default:
		s.allTestMessages = append(s.allTestMessages, line)
		return
	}
}

// testResult prints the test result.
func (s *spectester) testResult() {
	if s.stats.Fail > 0 {
		fmt.Printf("\n[Error Messages]\n")
		for _, msg := range extractFailTestMessage(s.allTestMessages) {
			fmt.Printf(" %s\n", msg)
		}
	}

	fmt.Printf("\n[Test Results]\n")
	fmt.Printf(" - Execution Time: %s\n", s.interval.Duration())
	fmt.Printf(" - Total         : %d\n", s.stats.Total)
	fmt.Printf(" - Passed        : %s\n", color.GreenString("%d", s.stats.Pass))
	if s.stats.Fail == 0 {
		fmt.Printf(" - Failed        : %d\n", s.stats.Fail)
	} else {
		fmt.Printf(" - Failed        : %s\n", color.RedString("%d", s.stats.Fail))
	}
	if s.stats.Skip == 0 {
		fmt.Printf(" - Skipped       : %d\n", s.stats.Skip)
	} else {
		fmt.Printf(" - Skipped       : %s\n", color.BlueString("%d", s.stats.Skip))
	}
}

func extractFailTestMessage(testResultMsgs []string) []string {
	failTestMessages := []string{}
	beforeRunPos := 0
	lastFailPos := 0
	lastRunMsg := ""

	for i, msg := range testResultMsgs {
		switch {
		case strings.Contains(msg, "=== RUN"):
			if lastRunMsg != "" && strings.Contains(msg, fmt.Sprintf("%s/", lastRunMsg)) {
				continue
			}

			if beforeRunPos < lastFailPos {
				for _, v := range testResultMsgs[beforeRunPos:lastFailPos] {
					if !strings.Contains(v, "--- FAIL") &&
						!strings.Contains(v, "--- PASS") &&
						!strings.Contains(v, "--- SKIP") &&
						!strings.Contains(v, "=== RUN") &&
						!strings.Contains(v, "=== CONT") &&
						!strings.Contains(v, "=== PAUSE") {
						failTestMessages = append(failTestMessages, fmt.Sprintf("    %s", color.RedString(v)))
					}
				}
			}
			lastRunMsg = msg
			beforeRunPos = i
		case strings.Contains(msg, "--- FAIL"):
			lastFailPos = i
			failTestMessages = append(failTestMessages, msg)
		default:
		}
	}
	return failTestMessages
}
