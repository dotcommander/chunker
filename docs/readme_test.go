package docs_test

import (
	"os"
	"strings"
	"testing"
)

func TestReadmeNavigation_HasAllSections(t *testing.T) {
	// Read docs/README.md from project root
	var content []byte
	var err error

	// If running from docs directory, adjust path
	projectRoot, _ := os.Getwd()
	if strings.HasSuffix(projectRoot, "/docs") {
		// Running from docs directory
		content, err = os.ReadFile("README.md")
	} else {
		// Running from project root
		content, err = os.ReadFile("docs/README.md")
	}
	if err != nil {
		t.Fatalf("Failed to read docs/README.md: %v", err)
	}

	doc := string(content)

	requiredSections := []struct {
		section string
		header  string
	}{
		{"Getting Started", "## Getting Started"},
		{"Concepts", "## Concepts"},
		{"Strategies", "## Strategies"},
		{"API Reference", "## API Reference"},
		{"Contributing", "## Contributing"},
	}

	for _, section := range requiredSections {
		t.Run(section.section, func(t *testing.T) {
			if !strings.Contains(doc, section.header) {
				t.Errorf("docs/README.md missing %q section (expected %q)",
					section.section, section.header)
			}
		})
	}
}

func TestReadmeNavigation_AllDocumentsLinked(t *testing.T) {
	// Read docs/README.md from project root
	var content []byte
	var err error

	// If running from docs directory, adjust path
	projectRoot, _ := os.Getwd()
	if strings.HasSuffix(projectRoot, "/docs") {
		// Running from docs directory
		content, err = os.ReadFile("README.md")
	} else {
		// Running from project root
		content, err = os.ReadFile("docs/README.md")
	}
	if err != nil {
		t.Fatalf("Failed to read docs/README.md: %v", err)
	}

	doc := string(content)

	requiredLinks := []struct {
		link        string
		description string
	}{
		// Concepts
		{"ARCHITECTURE.md", "Link to architecture documentation"},
		{"KNOWLEDGE.md", "Link to knowledge base"},

		// Strategies
		{"strategies/smart-boundary.md", "Link to smart boundary strategy"},
		{"strategies/sentence-boundary.md", "Link to sentence boundary strategy"},
		{"strategies/word-boundary.md", "Link to word boundary strategy"},

		// API Reference
		{"api/cli.md", "Link to CLI reference"},
		{"api/schemas.md", "Link to API schemas"},
		{"api/errors.md", "Link to error handling"},
		{"API.md", "Link to legacy API documentation"},

		// Contributing
		{"contributing/architecture.md", "Link to contributing architecture guide"},
		{"contributing/new-strategy.md", "Link to new strategy guide"},
	}

	for _, link := range requiredLinks {
		t.Run(link.description, func(t *testing.T) {
			if !strings.Contains(doc, link.link) {
				t.Errorf("docs/README.md missing link: %s (%s)",
					link.link, link.description)
			}
		})
	}
}

func TestReadmeNavigation_HasDescriptions(t *testing.T) {
	// Read docs/README.md from project root
	var content []byte
	var err error

	// If running from docs directory, adjust path
	projectRoot, _ := os.Getwd()
	if strings.HasSuffix(projectRoot, "/docs") {
		// Running from docs directory
		content, err = os.ReadFile("README.md")
	} else {
		// Running from project root
		content, err = os.ReadFile("docs/README.md")
	}
	if err != nil {
		t.Fatalf("Failed to read docs/README.md: %v", err)
	}

	doc := string(content)

	// Verify that links have descriptions (text between markdown link syntax)
	// A proper markdown link looks like [Description](path)

	linksWithDescriptions := []struct {
		pattern     string
		description string
	}{
		{"[Architecture Overview](ARCHITECTURE.md)", "Architecture overview link has description"},
		{"[Knowledge Base](KNOWLEDGE.md)", "Knowledge base link has description"},
		{"[Smart Boundary](strategies/smart-boundary.md)", "Smart boundary link has description"},
		{"[Sentence Boundary](strategies/sentence-boundary.md)", "Sentence boundary link has description"},
		{"[Word Boundary](strategies/word-boundary.md)", "Word boundary link has description"},
		{"[CLI Reference](api/cli.md)", "CLI reference link has description"},
		{"[API Schemas](api/schemas.md)", "API schemas link has description"},
		{"[Error Handling](api/errors.md)", "Error handling link has description"},
		{"[Architecture Overview](contributing/architecture.md)", "Contributing architecture link has description"},
		{"[Adding a New Strategy](contributing/new-strategy.md)", "New strategy link has description"},
	}

	for _, link := range linksWithDescriptions {
		t.Run(link.description, func(t *testing.T) {
			if !strings.Contains(doc, link.pattern) {
				t.Errorf("docs/README.md missing described link: %s", link.pattern)
			}
		})
	}
}

func TestReadmeNavigation_HasQuickLinksSection(t *testing.T) {
	// Read docs/README.md from project root
	var content []byte
	var err error

	// If running from docs directory, adjust path
	projectRoot, _ := os.Getwd()
	if strings.HasSuffix(projectRoot, "/docs") {
		// Running from docs directory
		content, err = os.ReadFile("README.md")
	} else {
		// Running from project root
		content, err = os.ReadFile("docs/README.md")
	}
	if err != nil {
		t.Fatalf("Failed to read docs/README.md: %v", err)
	}

	doc := string(content)

	// Check for Quick Links section
	if !strings.Contains(doc, "## Quick Links") {
		t.Error("docs/README.md missing Quick Links section")
	}

	// Verify Quick Links contains common task links
	commonTaskLinks := []struct {
		link        string
		description string
	}{
		{"#getting-started", "Quick link to Getting Started section"},
		{"#strategies", "Quick link to Strategies section"},
		{"#api-reference", "Quick link to API Reference section"},
		{"#contributing", "Quick link to Contributing section"},
	}

	for _, link := range commonTaskLinks {
		t.Run(link.description, func(t *testing.T) {
			if !strings.Contains(doc, link.link) {
				t.Errorf("docs/README.md Quick Links missing: %s", link.link)
			}
		})
	}
}

func TestReadmeNavigation_TableOfContents(t *testing.T) {
	// Read docs/README.md from project root
	var content []byte
	var err error

	// If running from docs directory, adjust path
	projectRoot, _ := os.Getwd()
	if strings.HasSuffix(projectRoot, "/docs") {
		// Running from docs directory
		content, err = os.ReadFile("README.md")
	} else {
		// Running from project root
		content, err = os.ReadFile("docs/README.md")
	}
	if err != nil {
		t.Fatalf("Failed to read docs/README.md: %v", err)
	}

	doc := string(content)

	// Check for Table of Contents section
	if !strings.Contains(doc, "## Table of Contents") {
		t.Error("docs/README.md missing Table of Contents section")
	}

	// Verify Table of Contents has internal links to main sections
	tocLinks := []struct {
		link        string
		section     string
		description string
	}{
		{"#getting-started", "Getting Started", "TOC link to Getting Started"},
		{"#concepts", "Concepts", "TOC link to Concepts"},
		{"#strategies", "Strategies", "TOC link to Strategies"},
		{"#api-reference", "API Reference", "TOC link to API Reference"},
		{"#contributing", "Contributing", "TOC link to Contributing"},
	}

	for _, link := range tocLinks {
		t.Run(link.description, func(t *testing.T) {
			// Check both the section header and the TOC link
			if !strings.Contains(doc, "- ["+link.section+"]("+link.link+")") &&
				!strings.Contains(doc, "- ["+link.section + "](" + link.link + ")") {
				t.Errorf("docs/README.md Table of Contents missing link to %s", link.section)
			}
		})
	}
}
