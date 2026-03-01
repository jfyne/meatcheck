package app

import (
	"strings"
	"testing"
)

// TestRenderMarkdownFrontmatterTable verifies that a document beginning with
// YAML frontmatter is rendered with a <table class="frontmatter"> block
// prepended to the body content.
//
// Scenario: YAML frontmatter rendered as metadata table
func TestRenderMarkdownFrontmatterTable(t *testing.T) {
	input := "---\ntitle: Test\ndate: 2024-01-01\n---\n# Hello"
	got := string(renderMarkdown(input))

	if !strings.Contains(got, `<table`) {
		t.Fatalf("expected <table in output, got: %q", got)
	}
	if !strings.Contains(got, `frontmatter`) {
		t.Fatalf("expected frontmatter class on table, got: %q", got)
	}
	if !strings.Contains(got, "title") {
		t.Fatalf("expected key 'title' in frontmatter table, got: %q", got)
	}
	if !strings.Contains(got, "Test") {
		t.Fatalf("expected value 'Test' in frontmatter table, got: %q", got)
	}
	if !strings.Contains(got, "date") {
		t.Fatalf("expected key 'date' in frontmatter table, got: %q", got)
	}
	if !strings.Contains(got, "2024-01-01") {
		t.Fatalf("expected value '2024-01-01' in frontmatter table, got: %q", got)
	}
	if !strings.Contains(got, `<h1`) {
		t.Fatalf("expected <h1 from body content after frontmatter, got: %q", got)
	}
}

// TestRenderMarkdownNoFrontmatter verifies that a document without frontmatter
// is rendered as normal markdown with no frontmatter table injected.
//
// Scenario: Document without frontmatter renders unchanged
func TestRenderMarkdownNoFrontmatter(t *testing.T) {
	input := "# Hello\nworld"
	got := string(renderMarkdown(input))

	if strings.Contains(got, "frontmatter") {
		t.Fatalf("expected no frontmatter class in output, got: %q", got)
	}
	if !strings.Contains(got, `<h1`) {
		t.Fatalf("expected <h1 from rendered heading, got: %q", got)
	}
	if !strings.Contains(got, "world") {
		t.Fatalf("expected body text 'world' in output, got: %q", got)
	}
}

// TestRenderMarkdownFrontmatterOnly verifies that a document containing only
// frontmatter (no body) still renders the frontmatter table.
//
// Scenario: YAML frontmatter rendered as metadata table (frontmatter-only case)
func TestRenderMarkdownFrontmatterOnly(t *testing.T) {
	input := "---\ntitle: Only\n---\n"
	got := string(renderMarkdown(input))

	if !strings.Contains(got, `frontmatter`) {
		t.Fatalf("expected frontmatter class in output even with no body, got: %q", got)
	}
	if !strings.Contains(got, "Only") {
		t.Fatalf("expected value 'Only' from frontmatter, got: %q", got)
	}
}

// TestRenderMarkdownEmptyFrontmatter verifies that a frontmatter block with no
// valid key-value pairs does not inject an empty table.
func TestRenderMarkdownEmptyFrontmatter(t *testing.T) {
	input := "---\n\n---\n# Hello"
	got := string(renderMarkdown(input))

	if strings.Contains(got, "frontmatter") {
		t.Fatalf("expected no frontmatter table for empty frontmatter block, got: %q", got)
	}
	if !strings.Contains(got, `<h1`) {
		t.Fatalf("expected <h1 from body content, got: %q", got)
	}
}

// TestRenderMarkdownNotFrontmatter verifies that a "---" separator that does
// NOT appear at line 0 (e.g., used as a horizontal rule) is NOT treated as
// frontmatter and does not produce a frontmatter table.
//
// Scenario: Document without frontmatter renders unchanged (HR case)
func TestRenderMarkdownNotFrontmatter(t *testing.T) {
	input := "Some content\n\n---\n\nMore content"
	got := string(renderMarkdown(input))

	if strings.Contains(got, "frontmatter") {
		t.Fatalf("expected no frontmatter table when --- is not at line 0, got: %q", got)
	}
	if !strings.Contains(got, "Some content") {
		t.Fatalf("expected 'Some content' in rendered output, got: %q", got)
	}
	if !strings.Contains(got, "More content") {
		t.Fatalf("expected 'More content' in rendered output, got: %q", got)
	}
}
