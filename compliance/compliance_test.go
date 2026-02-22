package compliance

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/agentable/jsonpath"
	"github.com/stretchr/testify/require"
)

// The CTS (Compliance Test Suite) is maintained as a git submodule at:
// .references/jsonpath-compliance-test-suite
//
// To update the CTS to the latest version:
//   cd .references/jsonpath-compliance-test-suite
//   git pull origin main
//   cd ../..
//   cp .references/jsonpath-compliance-test-suite/cts.json compliance/testdata/cts.json
//   git add compliance/testdata/cts.json
//   git commit -m "chore: update JSONPath CTS to latest version"

//go:embed testdata/cts.json
var ctsJSON []byte

// ctsFile represents the structure of the CTS JSON file.
type ctsFile struct {
	Description string     `json:"description"`
	Tests       []testCase `json:"tests"`
}

// testCase represents a single test case from the CTS.
type testCase struct {
	Name            string     `json:"name"`
	Selector        string     `json:"selector"`
	Document        any        `json:"document"`
	Result          []any      `json:"result"`
	Results         [][]any    `json:"results"`
	ResultPaths     []string   `json:"result_paths"`
	ResultsPaths    [][]string `json:"results_paths"`
	InvalidSelector bool       `json:"invalid_selector"`
	Tags            []string   `json:"tags"`
}

func TestCompliance(t *testing.T) {
	var suite ctsFile
	require.NoError(t, json.Unmarshal(ctsJSON, &suite))

	for _, tc := range suite.Tests {
		t.Run(tc.Name, func(t *testing.T) {
			// Invalid selector tests
			if tc.InvalidSelector {
				_, err := jsonpath.Parse(tc.Selector)
				require.Error(t, err, "expected parse error for invalid selector")
				return
			}

			// Valid selector tests
			path, err := jsonpath.Parse(tc.Selector)
			require.NoError(t, err, "failed to parse valid selector")

			got := path.Select(tc.Document)

			// Handle single result vs multiple results
			if tc.Results != nil {
				// Multiple possible results (non-deterministic ordering)
				require.Contains(t, tc.Results, []any(got), "result not in expected results")
			} else {
				// Single expected result
				require.ElementsMatch(t, tc.Result, got, "result mismatch")
			}

			// Validate paths if SelectLocated is implemented
			if tc.ResultPaths != nil || tc.ResultsPaths != nil {
				located := path.SelectLocated(tc.Document)
				gotPaths := make([]string, len(located))
				for i, loc := range located {
					gotPaths[i] = loc.Path.String()
				}

				if tc.ResultsPaths != nil {
					// Multiple possible path orderings
					require.Contains(t, tc.ResultsPaths, gotPaths, "paths not in expected paths")
				} else if tc.ResultPaths != nil {
					// Single expected path ordering
					require.Equal(t, tc.ResultPaths, gotPaths, "paths mismatch")
				}
			}
		})
	}
}
