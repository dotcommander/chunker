package chunking

import (
	"os"
	"strings"
	"testing"
)

func readSmartBoundaryDoc(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	path := "docs/strategies/smart-boundary.md"
	if strings.HasSuffix(wd, "/docs/strategies") {
		path = "smart-boundary.md"
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read documentation: %v", err)
	}

	return string(content)
}

func TestSmartBoundaryDocumentation_AcceptanceCriteria(t *testing.T) {
	doc := readSmartBoundaryDoc(t)

	tests := []struct {
		name           string
		requiredString string
		description    string
	}{
		{
			name:           "abbreviation-aware sentence detection explanation",
			requiredString: "Abbreviation-Aware Sentence Detection",
			description:    "Documentation must explain abbreviation-aware sentence detection",
		},
		{
			name:           "reliquary library reference",
			requiredString: "reliquary",
			description:    "Documentation must reference the reliquary library",
		},
		{
			name:           "abbreviation handling section",
			requiredString: "Abbreviation Handling",
			description:    "Documentation must explain abbreviation handling",
		},
		{
			name:           "Dr. example",
			requiredString: "Dr.",
			description:    "Documentation must show Dr. abbreviation example",
		},
		{
			name:           "U.S.A. example",
			requiredString: "U.S.A.",
			description:    "Documentation must show U.S.A. abbreviation example",
		},
		{
			name:           "fallback behavior section",
			requiredString: "Fallback Behavior",
			description:    "Documentation must document fallback to sentence_boundary",
		},
		{
			name:           "fallback to sentence_boundary",
			requiredString: "sentence_boundary",
			description:    "Documentation must mention sentence_boundary fallback",
		},
		{
			name:           "fallback splitter handling",
			requiredString: "fallback splitter",
			description:    "Documentation must explain what happens when the fallback splitter is used",
		},
		{
			name:           "usage examples",
			requiredString: "Usage Examples",
			description:    "Documentation must include usage examples",
		},
		{
			name:           "CLI example",
			requiredString: "cat document.txt",
			description:    "Documentation must show CLI usage",
		},
		{
			name:           "API example",
			requiredString: "curl",
			description:    "Documentation must show API usage",
		},
		{
			name:           "edge cases table",
			requiredString: "Edge Cases",
			description:    "Documentation must include edge cases",
		},
		{
			name:           "performance section",
			requiredString: "Performance",
			description:    "Documentation must discuss performance",
		},
		{
			name:           "when to use section",
			requiredString: "When to Use",
			description:    "Documentation must include usage recommendations",
		},
		{
			name:           "comparison to sentence_boundary",
			requiredString: "Comparison to sentence_boundary",
			description:    "Documentation must compare strategies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(doc, tt.requiredString) {
				t.Errorf("%s: documentation missing required content: %q\n",
					tt.description, tt.requiredString)
			}
		})
	}
}

func TestSmartBoundaryDocumentation_AbbreviationExamples(t *testing.T) {
	doc := readSmartBoundaryDoc(t)

	// Test that specific abbreviations are documented with examples
	abbreviationExamples := []string{
		"Dr. Smith",
		"U.S.A.",
		"Mr.",
		"version is 3.14",
	}

	for _, example := range abbreviationExamples {
		t.Run("example_contains_"+example, func(t *testing.T) {
			if !strings.Contains(doc, example) {
				t.Errorf("Documentation missing abbreviation example: %q", example)
			}
		})
	}
}

func TestSmartBoundaryDocumentation_FallbackDetails(t *testing.T) {
	doc := readSmartBoundaryDoc(t)

	// Test that fallback scenarios are documented
	fallbackScenarios := []string{
		"Malformed Unicode",
		"Symbol-only",
		"fallback splitter",
		"zero sentences",
	}

	for _, scenario := range fallbackScenarios {
		t.Run("fallback_scenario_"+scenario, func(t *testing.T) {
			if !strings.Contains(doc, scenario) {
				t.Errorf("Documentation missing fallback scenario: %q", scenario)
			}
		})
	}
}
