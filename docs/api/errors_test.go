package api_test

import (
	"os"
	"strings"
	"testing"
)

// TestErrorsDocumentation_AllValidationErrorsListed verifies that all validation errors
// from the codebase are documented in errors.md
func TestErrorsDocumentation_AllValidationErrorsListed(t *testing.T) {
	content, err := os.ReadFile("errors.md")
	if err != nil {
		t.Fatalf("Failed to read errors.md: %v", err)
	}
	fileContent := string(content)

	// Validation errors from models.go
	expectedErrors := []string{
		"text is required",
		"chunk_size must be greater than 0",
		"overlap must be greater than or equal to 0",
		"overlap must be less than chunk_size",
	}

	// Unknown/invalid strategy error from domain.ChunkRequest.Validate() (calls Strategy.IsValid())
	expectedErrors = append(expectedErrors, "failed to create chunker: unknown strategy")

	// Handler errors from chunk_handler.go
	expectedErrors = append(expectedErrors, "Invalid request body")
	expectedErrors = append(expectedErrors, "Method not allowed")

	for _, expected := range expectedErrors {
		if !strings.Contains(fileContent, expected) {
			t.Errorf("Expected error message not found in errors.md: %q", expected)
		}
	}
}

// TestErrorsDocumentation_ResponseStructureDocumented verifies that the error
// response JSON structure is documented
func TestErrorsDocumentation_ResponseStructureDocumented(t *testing.T) {
	content, err := os.ReadFile("errors.md")
	if err != nil {
		t.Fatalf("Failed to read errors.md: %v", err)
	}
	fileContent := string(content)

	// Check for error response structure documentation
	requiredSections := []string{
		"## Error Response Structure",
		"`error`",
		"`code`",
		"Human-readable error message",
		"Machine-readable error code",
	}

	for _, section := range requiredSections {
		if !strings.Contains(fileContent, section) {
			t.Errorf("Missing expected section in errors.md: %q", section)
		}
	}

	// Verify JSON example is present
	expectedJSONExample := `{
  "error": "Human-readable error message",
  "code": "machine_readable_error_code"
}`
	if !strings.Contains(fileContent, expectedJSONExample) {
		t.Error("Error response JSON example not found in errors.md")
	}
}

// TestErrorsDocumentation_RecoverySuggestionsExist verifies that each error type
// has recovery suggestions
func TestErrorsDocumentation_RecoverySuggestionsExist(t *testing.T) {
	content, err := os.ReadFile("errors.md")
	if err != nil {
		t.Fatalf("Failed to read errors.md: %v", err)
	}
	fileContent := string(content)

	// Map of error patterns to their expected recovery keywords
	errorRecoveryMap := map[string][]string{
		"text is required": {
			"Provide the `text` field",
		},
		"chunk_size must be greater than 0": {
			"Set `chunk_size` to a positive integer",
		},
		"overlap must be greater than or equal to 0": {
			"Set `overlap` to 0",
		},
		"overlap must be less than chunk_size": {
			"Reduce overlap",
			"increase chunk_size",
		},
		"unknown strategy": {
			"Use a valid strategy",
			"smart_boundary",
			"sentence_boundary",
		},
		"Invalid request body": {
			"Ensure request body is valid JSON",
		},
		"Method not allowed": {
			"Use POST to `/chunk` endpoint",
		},
		"no input text provided": {
			"Pipe text to stdin",
		},
	}

	for errorPattern, recoveryKeywords := range errorRecoveryMap {
		if !strings.Contains(fileContent, errorPattern) {
			t.Errorf("Error message not found in errors.md: %q", errorPattern)
			continue
		}

		// Find the section containing this error
		errorIndex := strings.Index(fileContent, errorPattern)
		if errorIndex == -1 {
			continue
		}

		// Look ahead for recovery suggestions (within next 500 chars)
		contextEnd := errorIndex + 500
		if contextEnd > len(fileContent) {
			contextEnd = len(fileContent)
		}
		context := fileContent[errorIndex:contextEnd]

		for _, keyword := range recoveryKeywords {
			if !strings.Contains(context, keyword) {
				t.Errorf("No recovery suggestion found for error %q (expected keyword: %q)", errorPattern, keyword)
			}
		}
	}
}

// TestErrorsDocumentation_RecoverySection verifies the presence of a
// dedicated recovery section
func TestErrorsDocumentation_RecoverySection(t *testing.T) {
	content, err := os.ReadFile("errors.md")
	if err != nil {
		t.Fatalf("Failed to read errors.md: %v", err)
	}
	fileContent := string(content)

	// Check for dedicated recovery section
	if !strings.Contains(fileContent, "## Error Handling Best Practices") {
		t.Error("Missing 'Error Handling Best Practices' section")
	}

	// Check for client-side validation example
	if !strings.Contains(fileContent, "Client-Side Validation") {
		t.Error("Missing 'Client-Side Validation' subsection")
	}

	// Check for retry strategy
	if !strings.Contains(fileContent, "Retry Strategy") {
		t.Error("Missing 'Retry Strategy' subsection")
	}
}
