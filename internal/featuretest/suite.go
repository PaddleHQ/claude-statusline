// Package featuretest runs the Gherkin feature files under features/ against the status line. It holds the
// whole step vocabulary, so a new scenario is written in a .feature file and needs no Go code. The status
// line's tests and the documentation image generator share it.
package featuretest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/cucumber/godog"
)

// ErrScenariosFailed is returned when a scenario fails, is undefined or is pending.
var ErrScenariosFailed = errors.New("feature scenarios failed")

// ErrNoScenarios is returned when no scenario was selected, which would otherwise pass without checking anything.
var ErrNoScenarios = errors.New("no scenarios were run: check the paths and tags")

// Execute runs the claude-statusline command with the given arguments, standard input and environment.
type Execute func(args []string, stdin io.Reader, stdout io.Writer, getenv func(string) string)

// Suite is a set of feature files and what runs them.
type Suite struct {
	// Execute runs the command a When step names.
	Execute Execute
	// Paths are the feature files or directories to read.
	Paths []string
	// Tags selects scenarios the way godog does: a comma is or, && is and and ~ is not, as in "@hooks && ~@screenshot".
	// Empty runs all.
	Tags string

	ran int
}

// Run runs the selected scenarios in strict mode, so an undefined or pending step fails. With a testing.T each
// scenario is a subtest and failures are reported through it; with nil the failures come back in the error.
func (s *Suite) Run(t *testing.T) error {
	s.ran = 0
	var log bytes.Buffer
	status := godog.TestSuite{
		Name:                "claude-statusline",
		ScenarioInitializer: s.initialize,
		Options: &godog.Options{
			Format:   "pretty",
			Output:   &log,
			NoColors: true,
			Strict:   true,
			Paths:    s.Paths,
			Tags:     s.Tags,
			TestingT: t,
		},
	}.Run()
	if status == 0 && s.ran == 0 {
		return ErrNoScenarios
	}
	if status == 0 {
		return nil
	}
	if t != nil {
		return ErrScenariosFailed
	}
	return fmt.Errorf("%w:\n%s", ErrScenariosFailed, log.String())
}

func (s *Suite) initialize(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		s.ran++
		w, err := newWorld(s)
		if err != nil {
			return ctx, err
		}
		return context.WithValue(ctx, worldKey{}, w), nil
	})
	sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		return ctx, worldOf(ctx).close()
	})

	sc.Step(`^the session:$`, func(ctx context.Context, doc *godog.DocString) { worldOf(ctx).setSession(doc.Content) })
	sc.Step(`^the config:$`, func(ctx context.Context, doc *godog.DocString) error { return worldOf(ctx).writeConfig(doc.Content) })
	sc.Step(`^the environment variable "([^"]*)" is "([^"]*)"$`, func(ctx context.Context, name, value string) {
		worldOf(ctx).setEnv(name, value)
	})
	sc.Step(`^the file "([^"]+)":$`, func(ctx context.Context, path string, doc *godog.DocString) error {
		return worldOf(ctx).writeFile(path, doc.Content, filePerm)
	})
	sc.Step(`^the directory "([^"]+)"$`, func(ctx context.Context, path string) error { return worldOf(ctx).makeDir(path) })

	sc.Step(`^the status line renders$`, func(ctx context.Context) { worldOf(ctx).renderStatusLine(ctx) })
	sc.Step(`^the status line renders at (\d+) columns$`, func(ctx context.Context, columns string) {
		worldOf(ctx).setEnv("COLUMNS", columns)
		worldOf(ctx).renderStatusLine(ctx)
	})

	sc.Step(`^the output is:$`, func(ctx context.Context, doc *godog.DocString) error {
		return worldOf(ctx).checkOutput(doc.Content)
	})
	sc.Step(`^the text is:$`, func(ctx context.Context, doc *godog.DocString) error {
		return worldOf(ctx).checkText(doc.Content)
	})
	sc.Step(`^the raw output is:$`, func(ctx context.Context, doc *godog.DocString) error {
		return worldOf(ctx).checkRawOutput(doc.Content)
	})
}
