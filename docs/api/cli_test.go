package api_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLIDocumentationFlags verifies that all CLI flags are documented
func TestCLIDocumentationFlags(t *testing.T) {
	cliDocPath := "../../docs/api/cli.md"
	content, err := os.ReadFile(cliDocPath)
	if err != nil {
		t.Fatalf("Failed to read CLI documentation: %v", err)
	}

	docText := string(content)

	requiredFlags := []string{
		"-size",
		"-strategy",
		"-overlap",
		"-encoding",
		"-format",
		"-pretty",
	}

	for _, flag := range requiredFlags {
		if !strings.Contains(docText, flag) {
			t.Errorf("Flag %s is not documented in CLI documentation", flag)
		}
	}

	requiredCommands := []string{"chunker serve", "chunker inspect", "chunker files"}
	for _, cmd := range requiredCommands {
		if !strings.Contains(docText, cmd) {
			t.Errorf("Command %s is not documented in CLI documentation", cmd)
		}
	}
}

// TestCLIDocumentationStdinPiping verifies stdin piping patterns are documented
func TestCLIDocumentationStdinPiping(t *testing.T) {
	cliDocPath := "../../docs/api/cli.md"
	content, err := os.ReadFile(cliDocPath)
	if err != nil {
		t.Fatalf("Failed to read CLI documentation: %v", err)
	}

	docText := string(content)

	// Check for stdin piping section
	if !strings.Contains(docText, "Stdin Piping Patterns") &&
		!strings.Contains(docText, "stdin") &&
		!strings.Contains(docText, "pipe") {
		t.Error("Stdin piping patterns section is missing from documentation")
	}

	// Verify piping examples exist
	requiredPatterns := []string{
		"cat document.txt | chunker",
		"chunker < document.txt",
		"cat",
		"|",
		"chunker",
	}

	for _, pattern := range requiredPatterns {
		if !strings.Contains(docText, pattern) {
			t.Errorf("Required piping pattern '%s' is missing from documentation", pattern)
		}
	}
}

// TestCLIDocumentationOutputFormats verifies JSON vs JSONL output formats are explained
func TestCLIDocumentationOutputFormats(t *testing.T) {
	cliDocPath := "../../docs/api/cli.md"
	content, err := os.ReadFile(cliDocPath)
	if err != nil {
		t.Fatalf("Failed to read CLI documentation: %v", err)
	}

	docText := string(content)

	// Check for output formats section
	if !strings.Contains(docText, "Output Formats") {
		t.Error("Output Formats section is missing from documentation")
	}

	// Verify JSON format is documented
	if !strings.Contains(docText, "JSON Format") && !strings.Contains(docText, "-format json") {
		t.Error("JSON format is not documented")
	}

	// Verify JSONL format is documented
	if !strings.Contains(docText, "JSONL Format") && !strings.Contains(docText, "-format jsonl") {
		t.Error("JSONL format is not documented")
	}

	// Verify explanation of differences
	if !strings.Contains(docText, "jsonl") || !strings.Contains(docText, "json") {
		t.Error("JSON and JSONL formats are not properly explained")
	}

	// Check for examples of both formats
	if !strings.Contains(docText, `"chunks"`) {
		t.Error("JSON format example is missing")
	}

	if !strings.Contains(docText, `"type": "chunk"`) && !strings.Contains(docText, `"type":"chunk"`) {
		t.Error("JSONL format example is missing")
	}
}

// TestCLIDocumentationExists verifies the CLI documentation file exists
func TestCLIDocumentationExists(t *testing.T) {
	cliDocPath := "../../docs/api/cli.md"

	// Get absolute path
	absPath, err := filepath.Abs(cliDocPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Errorf("CLI documentation file does not exist at: %s", absPath)
	}
}

// TestCLIDocumentationStructure verifies the documentation has proper structure
func TestCLIDocumentationStructure(t *testing.T) {
	cliDocPath := "../../docs/api/cli.md"
	content, err := os.ReadFile(cliDocPath)
	if err != nil {
		t.Fatalf("Failed to read CLI documentation: %v", err)
	}

	docText := string(content)

	// Check for required sections
	requiredSections := []string{
		"# CLI Reference",
		"## Overview",
		"## CLI Flags",
		"## Stdin Piping Patterns",
		"## Output Formats",
		"## Examples",
	}

	for _, section := range requiredSections {
		if !strings.Contains(docText, section) {
			t.Errorf("Required section '%s' is missing from documentation", section)
		}
	}

	// Verify it's markdown
	if !strings.HasSuffix(cliDocPath, ".md") {
		t.Error("Documentation file should be a markdown file (.md)")
	}
}
