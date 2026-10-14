package features_test

import (
	"flag"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/cli"
	"github.com/PaddleHQ/claude-statusline/internal/featuretest"
)

var featureTags = flag.String("feature.tags", "", "run only the scenarios matching these tags: , is or, && is and, ~ is not (for example @hooks,@wrapping)")

// TestFeatures runs every scenario in this directory except the @screenshot ones, which tools/screenshots runs.
// Each scenario is a subtest named after it. See CONTRIBUTING.md for the step vocabulary.
func TestFeatures(t *testing.T) {
	tags := "~@screenshot"
	if *featureTags != "" {
		terms := strings.Split(*featureTags, ",")
		for i, term := range terms {
			terms[i] = term + " && ~@screenshot"
		}
		tags = strings.Join(terms, ",")
	}
	suite := featuretest.Suite{Execute: cli.New(cli.Build{Version: "dev"}).Execute, Paths: []string{"."}, Tags: tags}
	if err := suite.Run(t); err != nil {
		t.Fatal(err)
	}
}
