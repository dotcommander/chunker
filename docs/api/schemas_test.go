package api_test

import (
	"os"
	"strings"
	"testing"
)

func readSchemasDoc(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	path := "docs/api/schemas.md"
	if strings.HasSuffix(wd, "/docs/api") {
		path = "schemas.md"
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read schemas documentation: %v", err)
	}

	return string(content)
}

func TestSchemasDocumentation_ChunkRequestSchema(t *testing.T) {
	doc := readSchemasDoc(t)

	tests := []struct {
		name           string
		requiredString string
		description    string
	}{
		{
			name:           "ChunkRequest section header",
			requiredString: "## ChunkRequest",
			description:    "Documentation must have ChunkRequest section",
		},
		{
			name:           "text field",
			requiredString: "`text`",
			description:    "Documentation must document text field",
		},
		{
			name:           "text field type string",
			requiredString: "| `text`         | `string`",
			description:    "Documentation must show text field type",
		},
		{
			name:           "text field required",
			requiredString: "| Yes",
			description:    "Documentation must mark text as required",
		},
		{
			name:           "chunk_size field",
			requiredString: "`chunk_size`",
			description:    "Documentation must document chunk_size field",
		},
		{
			name:           "chunk_size type int",
			requiredString: "| `chunk_size`   | `int`",
			description:    "Documentation must show chunk_size type",
		},
		{
			name:           "strategy field",
			requiredString: "`strategy`",
			description:    "Documentation must document strategy field",
		},
		{
			name:           "strategy optional",
			requiredString: "| `strategy`     | `string`       | No",
			description:    "Documentation must mark strategy as optional",
		},
		{
			name:           "overlap field",
			requiredString: "`overlap`",
			description:    "Documentation must document overlap field",
		},
		{
			name:           "token_encoding field",
			requiredString: "`token_encoding`",
			description:    "Documentation must document token_encoding field",
		},
		{
			name:           "validation rules section",
			requiredString: "### Validation Rules",
			description:    "Documentation must include validation rules",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(doc, tt.requiredString) {
				t.Errorf("%s: documentation missing required content: %q",
					tt.description, tt.requiredString)
			}
		})
	}
}

func TestSchemasDocumentation_ChunkResponseSchema(t *testing.T) {
	doc := readSchemasDoc(t)

	tests := []struct {
		name           string
		requiredString string
		description    string
	}{
		{
			name:           "ChunkResponse section header",
			requiredString: "## ChunkResponse",
			description:    "Documentation must have ChunkResponse section",
		},
		{
			name:           "chunks field",
			requiredString: "`chunks`",
			description:    "Documentation must document chunks field",
		},
		{
			name:           "chunks array type",
			requiredString: "| `chunks`  | `array[Chunk]`",
			description:    "Documentation must show chunks array type",
		},
		{
			name:           "metadata field",
			requiredString: "`metadata`",
			description:    "Documentation must document metadata field",
		},
		{
			name:           "metadata type",
			requiredString: "| `metadata`| `Metadata`",
			description:    "Documentation must show Metadata type",
		},
		{
			name:           "nested Chunk section",
			requiredString: "## Chunk",
			description:    "Documentation must document nested Chunk object",
		},
		{
			name:           "nested Metadata section",
			requiredString: "## Metadata",
			description:    "Documentation must document nested Metadata object",
		},
		{
			name:           "Chunk id field",
			requiredString: "| `id`          | `int`",
			description:    "Documentation must document Chunk id field",
		},
		{
			name:           "Chunk text field",
			requiredString: "| `text`        | `string`",
			description:    "Documentation must document Chunk text field",
		},
		{
			name:           "Chunk char_count field",
			requiredString: "| `char_count`",
			description:    "Documentation must document Chunk char_count",
		},
		{
			name:           "Chunk word_count field",
			requiredString: "| `word_count`",
			description:    "Documentation must document Chunk word_count",
		},
		{
			name:           "Chunk token_count field",
			requiredString: "| `token_count` | `int`    | No",
			description:    "Documentation must document token_count as optional",
		},
		{
			name:           "Metadata total_chunks",
			requiredString: "| `total_chunks`   | `int`",
			description:    "Documentation must document total_chunks",
		},
		{
			name:           "Metadata total_chars",
			requiredString: "| `total_chars`    | `int`",
			description:    "Documentation must document total_chars",
		},
		{
			name:           "Metadata total_tokens optional",
			requiredString: "| `total_tokens`   | `int`    | No",
			description:    "Documentation must mark total_tokens as optional",
		},
		{
			name:           "Metadata strategy_used",
			requiredString: "| `strategy_used`",
			description:    "Documentation must document strategy_used",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(doc, tt.requiredString) {
				t.Errorf("%s: documentation missing required content: %q",
					tt.description, tt.requiredString)
			}
		})
	}
}

func TestSchemasDocumentation_OptionalVsRequired(t *testing.T) {
	doc := readSchemasDoc(t)

	// Verify the table has a Required column that distinguishes optional/required
	if !strings.Contains(doc, "| Required |") {
		t.Error("Documentation table must have Required column")
	}

	// Check for clear marking of required fields
	requiredIndicators := []string{
		"| Yes", // Required marker
		"| No",  // Optional marker
	}

	for _, indicator := range requiredIndicators {
		if !strings.Contains(doc, indicator) {
			t.Errorf("Documentation must use %q to mark field requirements", indicator)
		}
	}

	// Verify specific fields are correctly marked
	requiredFields := []struct {
		field    string
		required bool
	}{
		{"text", true},
		{"chunk_size", true},
		{"strategy", false},
		{"overlap", false},
		{"token_encoding", false},
	}

	for _, field := range requiredFields {
		t.Run("field_"+field.field, func(t *testing.T) {
			// Find the table row for this field
			lines := strings.Split(doc, "\n")
			found := false
			for _, line := range lines {
				if strings.Contains(line, "`"+field.field+"`") {
					found = true
					expected := "| Yes"
					if !field.required {
						expected = "| No"
					}
					if !strings.Contains(line, expected) {
						t.Errorf("Field %s should be marked as %v", field.field, field.required)
					}
					break
				}
			}
			if !found {
				t.Errorf("Field %s not found in documentation table", field.field)
			}
		})
	}
}
