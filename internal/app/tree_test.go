package app

import (
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// fileHasComments tests
// ---------------------------------------------------------------------------

// TestFileHasCommentsTrueWhenCommentExists verifies that fileHasComments
// returns true when at least one comment matches the given path.
//
// Scenario: Returns true when a comment exists for the given path
func TestFileHasCommentsTrueWhenCommentExists(t *testing.T) {
	comments := []Comment{
		{ID: 1, Path: "other.go", StartLine: 1, EndLine: 1, Text: "nope"},
		{ID: 2, Path: "a.go", StartLine: 5, EndLine: 5, Text: "yes"},
	}
	if !fileHasComments("a.go", comments) {
		t.Error("expected fileHasComments to return true when a matching comment exists")
	}
}

// TestFileHasCommentsFalseWhenNoMatch verifies that fileHasComments returns
// false when no comment matches the given path.
//
// Scenario: Returns false when no comment matches
func TestFileHasCommentsFalseWhenNoMatch(t *testing.T) {
	comments := []Comment{
		{ID: 1, Path: "other.go", StartLine: 1, EndLine: 1, Text: "nope"},
	}
	if fileHasComments("a.go", comments) {
		t.Error("expected fileHasComments to return false when no comment matches the path")
	}
}

// TestFileHasCommentsNilSafe verifies that fileHasComments returns false
// safely when comments is nil (no panic).
//
// Scenario: nil comments returns false safely
func TestFileHasCommentsNilSafe(t *testing.T) {
	if fileHasComments("a.go", nil) {
		t.Error("expected fileHasComments to return false for nil comments slice")
	}
}

// ---------------------------------------------------------------------------
// buildTree tests (new signature with viewed and comments)
// ---------------------------------------------------------------------------

// TestBuildTreeViewedSetOnItem verifies that when a path is in the viewed
// map, the corresponding TreeItem has Viewed=true.
//
// Scenario: Viewed=true on a file sets TreeItem.Viewed=true
func TestBuildTreeViewedSetOnItem(t *testing.T) {
	files := []File{
		{Path: "a.go", PathSlash: "a.go", Lines: []string{}},
		{Path: "b.go", PathSlash: "b.go", Lines: []string{}},
	}
	viewed := map[string]bool{"a.go": true}
	items := buildTree(files, "a.go", viewed, nil)

	foundA, foundB := false, false
	for _, item := range items {
		if item.Path == "a.go" {
			foundA = true
			if !item.Viewed {
				t.Errorf("a.go: expected Viewed=true, got false")
			}
		}
		if item.Path == "b.go" {
			foundB = true
			if item.Viewed {
				t.Errorf("b.go: expected Viewed=false, got true")
			}
		}
	}
	if !foundA {
		t.Error("tree item for a.go not found")
	}
	if !foundB {
		t.Error("tree item for b.go not found")
	}
}

// TestBuildTreeHasCommentsSetOnItem verifies that when a comment exists for
// a file path, the corresponding TreeItem has HasComments=true.
//
// Scenario: Comment exists for a file sets TreeItem.HasComments=true
func TestBuildTreeHasCommentsSetOnItem(t *testing.T) {
	files := []File{
		{Path: "commented.go", PathSlash: "commented.go", Lines: []string{}},
		{Path: "clean.go", PathSlash: "clean.go", Lines: []string{}},
	}
	comments := []Comment{
		{ID: 1, Path: "commented.go", StartLine: 1, EndLine: 1, Text: "note"},
	}
	items := buildTree(files, "", nil, comments)

	foundCommented, foundClean := false, false
	for _, item := range items {
		if item.Path == "commented.go" {
			foundCommented = true
			if !item.HasComments {
				t.Errorf("commented.go: expected HasComments=true, got false")
			}
		}
		if item.Path == "clean.go" {
			foundClean = true
			if item.HasComments {
				t.Errorf("clean.go: expected HasComments=false, got true")
			}
		}
	}
	if !foundCommented {
		t.Error("tree item for commented.go not found")
	}
	if !foundClean {
		t.Error("tree item for clean.go not found")
	}
}

// TestBuildTreeNilViewedSafe verifies that passing a nil viewed map does not
// panic and leaves Viewed=false on all items.
//
// Scenario: nil viewed map is safe (Viewed stays false)
func TestBuildTreeNilViewedSafe(t *testing.T) {
	files := []File{
		{Path: "a.go", PathSlash: "a.go", Lines: []string{}},
	}
	// Must not panic.
	items := buildTree(files, "a.go", nil, nil)
	for _, item := range items {
		if item.IsDir {
			continue
		}
		if item.Viewed {
			t.Errorf("item %q: expected Viewed=false with nil viewed map, got true", item.Path)
		}
	}
}

// TestBuildTreeNilCommentsSafe verifies that passing nil comments does not
// panic and leaves HasComments=false on all items.
//
// Scenario: nil comments is safe (HasComments stays false)
func TestBuildTreeNilCommentsSafe(t *testing.T) {
	files := []File{
		{Path: "a.go", PathSlash: "a.go", Lines: []string{}},
	}
	// Must not panic.
	items := buildTree(files, "", nil, nil)
	for _, item := range items {
		if item.IsDir {
			continue
		}
		if item.HasComments {
			t.Errorf("item %q: expected HasComments=false with nil comments, got true", item.Path)
		}
	}
}

// ---------------------------------------------------------------------------
// buildGroupedTree tests
// ---------------------------------------------------------------------------

func makeFile(path string) File {
	return File{Path: path, PathSlash: filepath.ToSlash(path), Lines: []string{}}
}

// TestBuildGroupedTreeGroupHeadersHaveIsGroupTrue verifies that group entries
// in the resulting flat list have IsGroup=true and Depth=0.
//
// Scenario: Group headers produced with IsGroup=true, Depth=0
func TestBuildGroupedTreeGroupHeadersHaveIsGroupTrue(t *testing.T) {
	groups := []Group{
		{Name: "Auth", Files: []string{"auth.go"}},
		{Name: "API", Files: []string{"handler.go"}},
	}
	files := []File{makeFile("auth.go"), makeFile("handler.go")}

	items := buildGroupedTree(groups, files, "", nil, nil)

	var headers []TreeItem
	for _, item := range items {
		if item.IsGroup {
			headers = append(headers, item)
		}
	}
	if len(headers) != 2 {
		t.Fatalf("expected 2 group headers, got %d", len(headers))
	}
	for _, h := range headers {
		if h.Depth != 0 {
			t.Errorf("group header %q: expected Depth=0, got %d", h.Name, h.Depth)
		}
	}
	if headers[0].Name != "Auth" {
		t.Errorf("first group header: expected Name=Auth, got %q", headers[0].Name)
	}
	if headers[1].Name != "API" {
		t.Errorf("second group header: expected Name=API, got %q", headers[1].Name)
	}
}

// TestBuildGroupedTreeFilesAtDepth1WithGroupName verifies that file items
// within a group have Depth=1 and the correct GroupName set.
//
// Scenario: Files within group at Depth=1 with correct GroupName
func TestBuildGroupedTreeFilesAtDepth1WithGroupName(t *testing.T) {
	groups := []Group{
		{Name: "Auth", Files: []string{"auth.go"}},
	}
	files := []File{makeFile("auth.go")}

	items := buildGroupedTree(groups, files, "", nil, nil)

	var fileItems []TreeItem
	for _, item := range items {
		if !item.IsGroup && !item.IsDir {
			fileItems = append(fileItems, item)
		}
	}
	if len(fileItems) == 0 {
		t.Fatal("expected at least one file item in grouped tree, got none")
	}
	authItem := fileItems[0]
	if authItem.Depth != 1 {
		t.Errorf("auth.go: expected Depth=1, got %d", authItem.Depth)
	}
	if authItem.GroupName != "Auth" {
		t.Errorf("auth.go: expected GroupName=Auth, got %q", authItem.GroupName)
	}
}

// TestBuildGroupedTreeOtherGroupForUngroupedFiles verifies that files not
// assigned to any group appear under an "Other" group at the bottom of the
// tree output.
//
// Scenario: "Other" group produced for ungrouped files at bottom
func TestBuildGroupedTreeOtherGroupForUngroupedFiles(t *testing.T) {
	groups := []Group{
		{Name: "Auth", Files: []string{"auth.go"}},
	}
	files := []File{makeFile("auth.go"), makeFile("utils.go")}

	items := buildGroupedTree(groups, files, "", nil, nil)

	// Find the "Other" group header and verify it comes after "Auth".
	authHeaderIdx := -1
	otherHeaderIdx := -1
	utilsIdx := -1
	for i, item := range items {
		if item.IsGroup && item.Name == "Auth" {
			authHeaderIdx = i
		}
		if item.IsGroup && item.Name == "Other" {
			otherHeaderIdx = i
		}
		if !item.IsGroup && item.Path == "utils.go" {
			utilsIdx = i
		}
	}

	if otherHeaderIdx == -1 {
		t.Fatal("expected an 'Other' group header for ungrouped files, not found")
	}
	if utilsIdx == -1 {
		t.Fatal("expected utils.go to appear in the tree, not found")
	}
	if authHeaderIdx != -1 && otherHeaderIdx < authHeaderIdx {
		t.Errorf("'Other' group (idx %d) should appear after 'Auth' group (idx %d)", otherHeaderIdx, authHeaderIdx)
	}
	if utilsIdx < otherHeaderIdx {
		t.Errorf("utils.go (idx %d) should appear after 'Other' header (idx %d)", utilsIdx, otherHeaderIdx)
	}
}

// TestBuildGroupedTreeGroupOrderingPreserved verifies that groups appear in
// the order they are specified in the groups slice.
//
// Scenario: Group ordering preserved (groups appear in order specified)
func TestBuildGroupedTreeGroupOrderingPreserved(t *testing.T) {
	groups := []Group{
		{Name: "Zebra", Files: []string{"z.go"}},
		{Name: "Alpha", Files: []string{"a.go"}},
		{Name: "Middle", Files: []string{"m.go"}},
	}
	files := []File{makeFile("a.go"), makeFile("m.go"), makeFile("z.go")}

	items := buildGroupedTree(groups, files, "", nil, nil)

	var headerNames []string
	for _, item := range items {
		if item.IsGroup {
			headerNames = append(headerNames, item.Name)
		}
	}

	want := []string{"Zebra", "Alpha", "Middle"}
	if len(headerNames) < len(want) {
		t.Fatalf("expected at least %d group headers, got %d: %v", len(want), len(headerNames), headerNames)
	}
	for i, name := range want {
		if headerNames[i] != name {
			t.Errorf("group header[%d]: expected %q, got %q", i, name, headerNames[i])
		}
	}
}

// TestBuildGroupedTreeSelectedFileHasSelectedTrue verifies that the tree item
// for the selected path has Selected=true and all others have Selected=false.
//
// Scenario: Selected file correct (Selected=true on matching file)
func TestBuildGroupedTreeSelectedFileHasSelectedTrue(t *testing.T) {
	groups := []Group{
		{Name: "Auth", Files: []string{"auth.go", "token.go"}},
	}
	files := []File{makeFile("auth.go"), makeFile("token.go")}

	items := buildGroupedTree(groups, files, "token.go", nil, nil)

	foundToken := false
	for _, item := range items {
		if item.IsGroup {
			continue
		}
		if item.Path == "token.go" {
			foundToken = true
			if !item.Selected {
				t.Errorf("token.go: expected Selected=true, got false")
			}
		} else if item.Selected {
			t.Errorf("item %q: expected Selected=false, got true", item.Path)
		}
	}
	if !foundToken {
		t.Error("token.go item not found in grouped tree")
	}
}

// TestBuildGroupedTreeGroupActiveSetWhenSelectedBelongsToGroup verifies that
// the group header item has GroupActive=true when the selectedPath belongs to
// that group, and GroupActive=false for other groups.
//
// Scenario: GroupActive set when selectedPath belongs to a group
func TestBuildGroupedTreeGroupActiveSetWhenSelectedBelongsToGroup(t *testing.T) {
	groups := []Group{
		{Name: "Auth", Files: []string{"auth.go"}},
		{Name: "API", Files: []string{"handler.go"}},
	}
	files := []File{makeFile("auth.go"), makeFile("handler.go")}

	items := buildGroupedTree(groups, files, "auth.go", nil, nil)

	var authHeader, apiHeader *TreeItem
	for i := range items {
		if items[i].IsGroup && items[i].Name == "Auth" {
			authHeader = &items[i]
		}
		if items[i].IsGroup && items[i].Name == "API" {
			apiHeader = &items[i]
		}
	}

	if authHeader == nil {
		t.Fatal("Auth group header not found")
	}
	if apiHeader == nil {
		t.Fatal("API group header not found")
	}
	if !authHeader.GroupActive {
		t.Errorf("Auth group: expected GroupActive=true when auth.go is selected, got false")
	}
	if apiHeader.GroupActive {
		t.Errorf("API group: expected GroupActive=false when auth.go is selected, got true")
	}
}
