package app

import "testing"

// TestDiffFormatConstants verifies DiffFormatUnified and DiffFormatSplit have
// the correct string values.
func TestDiffFormatConstants(t *testing.T) {
	if DiffFormatUnified != DiffFormat("unified") {
		t.Errorf("expected DiffFormatUnified == \"unified\", got %q", DiffFormatUnified)
	}
	if DiffFormatSplit != DiffFormat("split") {
		t.Errorf("expected DiffFormatSplit == \"split\", got %q", DiffFormatSplit)
	}
	// Ensure the two constants are distinct.
	if DiffFormatUnified == DiffFormatSplit {
		t.Error("DiffFormatUnified and DiffFormatSplit must be distinct")
	}
}

// TestDiffOldLineExists is a table-driven test for diffOldLineExists.
func TestDiffOldLineExists(t *testing.T) {
	files := []DiffFile{
		{
			Path: "a.go",
			Hunks: []DiffHunk{
				{
					OldStart: 1, OldCount: 3, NewStart: 1, NewCount: 3,
					Lines: []DiffLine{
						{Kind: DiffContext, OldLine: 1, NewLine: 1, Text: "ctx"},
						{Kind: DiffDel, OldLine: 2, NewLine: 0, Text: "removed"},
						{Kind: DiffAdd, OldLine: 0, NewLine: 2, Text: "added"},
						{Kind: DiffContext, OldLine: 3, NewLine: 3, Text: "ctx2"},
					},
				},
			},
		},
	}

	tests := []struct {
		name     string
		path     string
		oldLine  int
		expected bool
	}{
		{
			name:     "old line on a del line returns true",
			path:     "a.go",
			oldLine:  2,
			expected: true,
		},
		{
			name:     "old line on a context line returns true",
			path:     "a.go",
			oldLine:  1,
			expected: true,
		},
		{
			name:     "old line 0 on add line returns false",
			path:     "a.go",
			oldLine:  0,
			expected: false,
		},
		{
			name:     "line number not in any hunk returns false",
			path:     "a.go",
			oldLine:  99,
			expected: false,
		},
		{
			name:     "file not found returns false",
			path:     "notexist.go",
			oldLine:  1,
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := diffOldLineExists(files, tc.path, tc.oldLine)
			if got != tc.expected {
				t.Errorf("diffOldLineExists(%q, %d) = %v, want %v", tc.path, tc.oldLine, got, tc.expected)
			}
		})
	}
}

// TestReviewModelHasDiffFormat verifies that ReviewModel carries a DiffFormat
// field and that it can be set to the known constants.
func TestReviewModelHasDiffFormat(t *testing.T) {
	m := ReviewModel{}

	// Zero value — DiffFormat field must exist (compiler enforces this).
	if m.DiffFormat != DiffFormat("") {
		t.Errorf("zero value of DiffFormat should be empty string, got %q", m.DiffFormat)
	}

	m.DiffFormat = DiffFormatUnified
	if m.DiffFormat != DiffFormatUnified {
		t.Errorf("expected DiffFormatUnified after assignment, got %q", m.DiffFormat)
	}

	m.DiffFormat = DiffFormatSplit
	if m.DiffFormat != DiffFormatSplit {
		t.Errorf("expected DiffFormatSplit after assignment, got %q", m.DiffFormat)
	}
}
