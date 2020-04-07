package duration_test

import (
	"testing"

	"github.com/gostaticanalysis/duration"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAnalyzer is a test for Analyzer.
func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, duration.Analyzer, "a")
}
