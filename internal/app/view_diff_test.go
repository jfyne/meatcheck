package app

import (
	"strings"
	"testing"
)

func TestBuildViewDiffCommentsNewLinesOnly(t *testing.T) {
	df := &DiffFile{Path: "x.go", Hunks: []DiffHunk{{
		OldStart: 1,
		OldCount: 1,
		NewStart: 1,
		NewCount: 1,
		Lines: []DiffLine{
			{Kind: DiffDel, OldLine: 1, NewLine: 0, Text: "old"},
			{Kind: DiffAdd, OldLine: 0, NewLine: 1, Text: "new"},
		},
	}}}
	comments := []Comment{{ID: 1, Path: "x.go", StartLine: 1, EndLine: 1, Text: "hi"}}
	view := buildViewDiff(df, comments, 1, 1, false, 0, "")
	if len(view.Hunks) != 1 {
		t.Fatalf("expected 1 hunk")
	}
	lines := view.Hunks[0].Lines
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines")
	}
	if lines[0].Commented {
		t.Fatal("deleted line should not be commented")
	}
	if !lines[1].Commented {
		t.Fatal("added line should be commented")
	}
	if len(lines[1].Comments) != 1 {
		t.Fatalf("expected 1 comment on added line")
	}
}

// TestBuildViewDiffOldLineSelection verifies that when selectionSide == "old",
// a deleted line whose OldLine falls in [start, end] is marked Selected.
//
// Scenario: selectionSide "old" selects deleted line by OldLine number
func TestBuildViewDiffOldLineSelection(t *testing.T) {
	df := &DiffFile{Path: "f.go", Hunks: []DiffHunk{{
		OldStart: 5,
		OldCount: 1,
		NewStart: 5,
		NewCount: 1,
		Lines: []DiffLine{
			{Kind: DiffDel, OldLine: 5, NewLine: 0, Text: "deleted text"},
			{Kind: DiffAdd, OldLine: 0, NewLine: 5, Text: "added text"},
		},
	}}}

	// selectionSide="old", select range [5,5]: only the del line (OldLine==5) should be selected.
	view := buildViewDiff(df, nil, 5, 5, false, 0, "old")

	if len(view.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(view.Hunks))
	}
	lines := view.Hunks[0].Lines
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	delLine := lines[0]
	if delLine.Kind != DiffDel {
		t.Fatalf("lines[0] should be DiffDel, got %q", delLine.Kind)
	}
	if !delLine.Selected {
		t.Errorf("deleted line with OldLine=5 should be Selected when selectionSide='old' and range=[5,5]")
	}

	addLine := lines[1]
	if addLine.Kind != DiffAdd {
		t.Fatalf("lines[1] should be DiffAdd, got %q", addLine.Kind)
	}
	// The add line has OldLine=0, so with selectionSide="old" it must NOT be selected.
	if addLine.Selected {
		t.Errorf("added line (OldLine=0) should NOT be selected when selectionSide='old'")
	}
}

// TestBuildViewDiffOldLineComments verifies that comments with Side "old" appear
// on deleted lines and comments with Side "" appear on context/new lines.
//
// Scenario: old-side comment projects onto deleted line via OldLine
// Scenario: new-side comment projects onto context line via NewLine
func TestBuildViewDiffOldLineComments(t *testing.T) {
	// del line: OldLine=3, NewLine=0
	// context line: OldLine=3, NewLine=4  (same old number as del, different new number)
	df := &DiffFile{Path: "g.go", Hunks: []DiffHunk{{
		OldStart: 3,
		OldCount: 2,
		NewStart: 3,
		NewCount: 2,
		Lines: []DiffLine{
			{Kind: DiffDel, OldLine: 3, NewLine: 0, Text: "removed line"},
			{Kind: DiffContext, OldLine: 4, NewLine: 4, Text: "context line"},
		},
	}}}

	oldSideComment := Comment{ID: 10, Path: "g.go", StartLine: 3, EndLine: 3, Text: "old comment", Side: "old"}
	newSideComment := Comment{ID: 11, Path: "g.go", StartLine: 4, EndLine: 4, Text: "new comment", Side: ""}
	comments := []Comment{oldSideComment, newSideComment}

	view := buildViewDiff(df, comments, 0, 0, false, 0, "")

	if len(view.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(view.Hunks))
	}
	lines := view.Hunks[0].Lines
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	delLine := lines[0]
	if delLine.Kind != DiffDel {
		t.Fatalf("lines[0] should be DiffDel")
	}
	// Deleted line should show the old-side comment (Side="old", OldLine=3).
	if !delLine.Commented {
		t.Errorf("deleted line (OldLine=3) should be Commented via the old-side comment")
	}
	if len(delLine.Comments) != 1 {
		t.Errorf("deleted line should have 1 comment, got %d", len(delLine.Comments))
	} else if delLine.Comments[0].ID != 10 {
		t.Errorf("deleted line comment should have ID=10, got %d", delLine.Comments[0].ID)
	}

	ctxLine := lines[1]
	if ctxLine.Kind != DiffContext {
		t.Fatalf("lines[1] should be DiffContext")
	}
	// Context line should show the new-side comment (Side="", NewLine=4).
	if !ctxLine.Commented {
		t.Errorf("context line (NewLine=4) should be Commented via the new-side comment")
	}
	if len(ctxLine.Comments) != 1 {
		t.Errorf("context line should have 1 comment, got %d", len(ctxLine.Comments))
	} else if ctxLine.Comments[0].ID != 11 {
		t.Errorf("context line comment should have ID=11, got %d", ctxLine.Comments[0].ID)
	}
}

// TestBuildViewDiffDeletedLinesSelectable verifies that deleted lines are no
// longer blocked by the selectable guard and can be selected when
// selectionSide == "old".
//
// Scenario: deleted lines not filtered out — always included in output
// Scenario: deleted line selected when selectionSide="old" and OldLine in range
func TestBuildViewDiffDeletedLinesSelectable(t *testing.T) {
	df := &DiffFile{Path: "h.go", Hunks: []DiffHunk{{
		OldStart: 10,
		OldCount: 1,
		NewStart: 11,
		NewCount: 0,
		Lines: []DiffLine{
			{Kind: DiffDel, OldLine: 10, NewLine: 0, Text: "removed"},
		},
	}}}

	// First call: selectionSide="" with no selection range — line must appear in output.
	view := buildViewDiff(df, nil, 0, 0, false, 0, "")

	if len(view.Hunks) != 1 {
		t.Fatalf("case 1: expected 1 hunk, got %d", len(view.Hunks))
	}
	lines := view.Hunks[0].Lines
	if len(lines) != 1 {
		t.Fatalf("case 1: expected 1 line (del line must not be filtered), got %d", len(lines))
	}
	if lines[0].Kind != DiffDel {
		t.Fatalf("case 1: expected DiffDel line, got %q", lines[0].Kind)
	}
	if lines[0].OldLine != 10 {
		t.Errorf("case 1: expected OldLine=10, got %d", lines[0].OldLine)
	}
	// With no selection range the line must NOT be selected.
	if lines[0].Selected {
		t.Errorf("case 1: del line should not be selected when start=0, end=0")
	}

	// Second call: selectionSide="old", range covers OldLine=10 — del line must be selected.
	view2 := buildViewDiff(df, nil, 10, 10, false, 0, "old")

	if len(view2.Hunks) != 1 {
		t.Fatalf("case 2: expected 1 hunk, got %d", len(view2.Hunks))
	}
	lines2 := view2.Hunks[0].Lines
	if len(lines2) != 1 {
		t.Fatalf("case 2: expected 1 line, got %d", len(lines2))
	}
	if !lines2[0].Selected {
		t.Errorf("case 2: del line (OldLine=10) should be Selected when selectionSide='old' and range=[10,10]")
	}
}

// TestBuildViewDiffIntraLineUnified verifies that adjacent del/add pairs in
// unified view get intra-line word-level diff HTML.
func TestBuildViewDiffIntraLineUnified(t *testing.T) {
	df := &DiffFile{Path: "intra.go", Hunks: []DiffHunk{{
		OldStart: 1,
		OldCount: 1,
		NewStart: 1,
		NewCount: 1,
		Lines: []DiffLine{
			{Kind: DiffDel, OldLine: 1, NewLine: 0, Text: "return nil"},
			{Kind: DiffAdd, OldLine: 0, NewLine: 1, Text: "return err"},
		},
	}}}

	view := buildViewDiff(df, nil, 0, 0, false, 0, "")

	lines := view.Hunks[0].Lines
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if !strings.Contains(string(lines[0].HTML), "intra-del") {
		t.Errorf("del line HTML should contain intra-del, got: %s", lines[0].HTML)
	}
	if !strings.Contains(string(lines[1].HTML), "intra-add") {
		t.Errorf("add line HTML should contain intra-add, got: %s", lines[1].HTML)
	}
}

// TestBuildViewDiffIntraLineUnifiedNonAdjacent verifies that a del line NOT
// immediately followed by an add line does not get intra-line HTML.
func TestBuildViewDiffIntraLineUnifiedNonAdjacent(t *testing.T) {
	df := &DiffFile{Path: "nonadj.go", Hunks: []DiffHunk{{
		OldStart: 1,
		OldCount: 2,
		NewStart: 1,
		NewCount: 1,
		Lines: []DiffLine{
			{Kind: DiffDel, OldLine: 1, NewLine: 0, Text: "first deleted"},
			{Kind: DiffDel, OldLine: 2, NewLine: 0, Text: "second deleted"},
			{Kind: DiffAdd, OldLine: 0, NewLine: 1, Text: "only add"},
		},
	}}}

	view := buildViewDiff(df, nil, 0, 0, false, 0, "")

	lines := view.Hunks[0].Lines
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	// First del is followed by another del, not an add — no intra-line.
	if strings.Contains(string(lines[0].HTML), "intra-") {
		t.Errorf("first del should NOT have intra-line HTML when followed by another del, got: %s", lines[0].HTML)
	}
	// Second del IS followed by an add — should get intra-line.
	if !strings.Contains(string(lines[1].HTML), "intra-del") {
		t.Errorf("second del (followed by add) should have intra-del, got: %s", lines[1].HTML)
	}
	if !strings.Contains(string(lines[2].HTML), "intra-add") {
		t.Errorf("add line should have intra-add, got: %s", lines[2].HTML)
	}
}
