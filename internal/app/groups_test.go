package app

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseGroupsFileValidJSON verifies that a well-formed JSON file with
// multiple groups is parsed into the expected []Group slice with paths
// normalized via filepath.ToSlash(filepath.Clean).
//
// Scenario: Valid JSON array with multiple groups parses correctly
func TestParseGroupsFileValidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "groups.json")

	content := `[
		{"name": "backend", "files": ["src/server.go", "src/db.go"]},
		{"name": "frontend", "files": ["web/app.js"]}
	]`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	groups, err := ParseGroupsFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	if groups[0].Name != "backend" {
		t.Errorf("group[0].Name: got %q, want %q", groups[0].Name, "backend")
	}
	if len(groups[0].Files) != 2 {
		t.Fatalf("group[0].Files: expected 2 files, got %d", len(groups[0].Files))
	}
	if groups[0].Files[0] != "src/server.go" {
		t.Errorf("group[0].Files[0]: got %q, want %q", groups[0].Files[0], "src/server.go")
	}
	if groups[0].Files[1] != "src/db.go" {
		t.Errorf("group[0].Files[1]: got %q, want %q", groups[0].Files[1], "src/db.go")
	}

	if groups[1].Name != "frontend" {
		t.Errorf("group[1].Name: got %q, want %q", groups[1].Name, "frontend")
	}
	if len(groups[1].Files) != 1 {
		t.Fatalf("group[1].Files: expected 1 file, got %d", len(groups[1].Files))
	}
	if groups[1].Files[0] != "web/app.js" {
		t.Errorf("group[1].Files[0]: got %q, want %q", groups[1].Files[0], "web/app.js")
	}
}

// TestParseGroupsFilePathNormalization verifies that file paths within groups
// are normalized using filepath.ToSlash(filepath.Clean).
//
// Scenario: Valid JSON array with multiple groups — paths are normalized
func TestParseGroupsFilePathNormalization(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "groups.json")

	// Use paths with redundant separators that Clean will normalize.
	content := `[{"name": "misc", "files": ["src//util.go", "src/./helper.go"]}]`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	groups, err := ParseGroupsFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Files[0] != "src/util.go" {
		t.Errorf("expected normalized path %q, got %q", "src/util.go", groups[0].Files[0])
	}
	if groups[0].Files[1] != "src/helper.go" {
		t.Errorf("expected normalized path %q, got %q", "src/helper.go", groups[0].Files[1])
	}
}

// TestParseGroupsFileInvalidJSON verifies that a file with invalid JSON
// content returns an error.
//
// Scenario: Invalid JSON returns error
func TestParseGroupsFileInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "groups.json")

	if err := os.WriteFile(path, []byte(`not valid json`), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := ParseGroupsFile(path)
	if err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
}

// TestParseGroupsFileEmptyNameReturnsError verifies that a group with an empty
// name field returns a validation error.
//
// Scenario: Group with empty name returns error
func TestParseGroupsFileEmptyNameReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "groups.json")

	content := `[{"name": "", "files": ["src/server.go"]}]`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := ParseGroupsFile(path)
	if err == nil {
		t.Fatal("expected an error for group with empty name, got nil")
	}
}

// TestParseGroupsFileEmptyFilesReturnsError verifies that a group with an
// empty files list returns a validation error.
//
// Scenario: Group with empty files list returns error
func TestParseGroupsFileEmptyFilesReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "groups.json")

	content := `[{"name": "backend", "files": []}]`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := ParseGroupsFile(path)
	if err == nil {
		t.Fatal("expected an error for group with empty files list, got nil")
	}
}

// TestParseGroupsFileNonExistentPath verifies that providing a path to a file
// that does not exist returns an error.
//
// Scenario: Non-existent file path returns error
func TestParseGroupsFileNonExistentPath(t *testing.T) {
	_, err := ParseGroupsFile("/nonexistent/path/groups.json")
	if err == nil {
		t.Fatal("expected an error for non-existent file path, got nil")
	}
}
