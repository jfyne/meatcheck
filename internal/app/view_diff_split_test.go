package app

import (
	"fmt"
	"testing"
)

// TestBuildViewDiffSplitContextLines verifies that context lines appear on
// both the left and right sides of a split-diff row.
//
// Scenario: Context lines appear on both sides
func TestBuildViewDiffSplitContextLines(t *testing.T) {
	df := &DiffFile{Path: "ctx.go", Hunks: []DiffHunk{{
		OldStart: 1,
		OldCount: 2,
		NewStart: 1,
		NewCount: 2,
		Lines: []DiffLine{
			{Kind: DiffContext, OldLine: 1, NewLine: 1, Text: "line one"},
			{Kind: DiffContext, OldLine: 2, NewLine: 2, Text: "line two"},
		},
	}}}

	hunks := buildViewDiffSplit(df, nil, 0, 0, false, 0, "")

	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}
	rows := hunks[0].Rows
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	// Row 0: context line 1
	r0 := rows[0]
	if r0.Left.Line != 1 {
		t.Errorf("row 0 Left.Line: got %d, want 1", r0.Left.Line)
	}
	if r0.Right.Line != 1 {
		t.Errorf("row 0 Right.Line: got %d, want 1", r0.Right.Line)
	}
	if r0.Left.Text != "line one" {
		t.Errorf("row 0 Left.Text: got %q, want %q", r0.Left.Text, "line one")
	}
	if r0.Right.Text != "line one" {
		t.Errorf("row 0 Right.Text: got %q, want %q", r0.Right.Text, "line one")
	}
	if r0.Left.Kind != DiffContext {
		t.Errorf("row 0 Left.Kind: got %q, want %q", r0.Left.Kind, DiffContext)
	}
	if r0.Right.Kind != DiffContext {
		t.Errorf("row 0 Right.Kind: got %q, want %q", r0.Right.Kind, DiffContext)
	}
	if r0.Left.Empty {
		t.Errorf("row 0 Left.Empty: got true, want false")
	}
	if r0.Right.Empty {
		t.Errorf("row 0 Right.Empty: got true, want false")
	}

	// Row 1: context line 2
	r1 := rows[1]
	if r1.Left.Line != 2 {
		t.Errorf("row 1 Left.Line: got %d, want 2", r1.Left.Line)
	}
	if r1.Right.Line != 2 {
		t.Errorf("row 1 Right.Line: got %d, want 2", r1.Right.Line)
	}
	if r1.Left.Text != "line two" {
		t.Errorf("row 1 Left.Text: got %q, want %q", r1.Left.Text, "line two")
	}
	if r1.Right.Text != "line two" {
		t.Errorf("row 1 Right.Text: got %q, want %q", r1.Right.Text, "line two")
	}
}

// TestBuildViewDiffSplitDelAddPairing verifies that a del/add block is paired
// 1:1, with unpaired add lines getting an Empty left side.
//
// Scenario: Del/Add blocks: collect consecutive dels then adds, zip 1:1;
// unpaired lines get Empty: true opposite side (more adds than dels case).
func TestBuildViewDiffSplitDelAddPairing(t *testing.T) {
	df := &DiffFile{Path: "pair.go", Hunks: []DiffHunk{{
		OldStart: 1,
		OldCount: 2,
		NewStart: 1,
		NewCount: 3,
		Lines: []DiffLine{
			{Kind: DiffDel, OldLine: 1, NewLine: 0, Text: "old1"},
			{Kind: DiffDel, OldLine: 2, NewLine: 0, Text: "old2"},
			{Kind: DiffAdd, OldLine: 0, NewLine: 1, Text: "new1"},
			{Kind: DiffAdd, OldLine: 0, NewLine: 2, Text: "new2"},
			{Kind: DiffAdd, OldLine: 0, NewLine: 3, Text: "new3"},
		},
	}}}

	hunks := buildViewDiffSplit(df, nil, 0, 0, false, 0, "")

	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}
	rows := hunks[0].Rows
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	// Row 0: del old1 paired with add new1
	r0 := rows[0]
	if r0.Left.Line != 1 {
		t.Errorf("row 0 Left.Line: got %d, want 1", r0.Left.Line)
	}
	if r0.Left.Kind != DiffDel {
		t.Errorf("row 0 Left.Kind: got %q, want %q", r0.Left.Kind, DiffDel)
	}
	if r0.Left.Text != "old1" {
		t.Errorf("row 0 Left.Text: got %q, want %q", r0.Left.Text, "old1")
	}
	if r0.Left.Empty {
		t.Errorf("row 0 Left.Empty: got true, want false")
	}
	if r0.Right.Line != 1 {
		t.Errorf("row 0 Right.Line: got %d, want 1", r0.Right.Line)
	}
	if r0.Right.Kind != DiffAdd {
		t.Errorf("row 0 Right.Kind: got %q, want %q", r0.Right.Kind, DiffAdd)
	}
	if r0.Right.Text != "new1" {
		t.Errorf("row 0 Right.Text: got %q, want %q", r0.Right.Text, "new1")
	}
	if r0.Right.Empty {
		t.Errorf("row 0 Right.Empty: got true, want false")
	}

	// Row 1: del old2 paired with add new2
	r1 := rows[1]
	if r1.Left.Line != 2 {
		t.Errorf("row 1 Left.Line: got %d, want 2", r1.Left.Line)
	}
	if r1.Left.Kind != DiffDel {
		t.Errorf("row 1 Left.Kind: got %q, want %q", r1.Left.Kind, DiffDel)
	}
	if r1.Left.Text != "old2" {
		t.Errorf("row 1 Left.Text: got %q, want %q", r1.Left.Text, "old2")
	}
	if r1.Right.Line != 2 {
		t.Errorf("row 1 Right.Line: got %d, want 2", r1.Right.Line)
	}
	if r1.Right.Kind != DiffAdd {
		t.Errorf("row 1 Right.Kind: got %q, want %q", r1.Right.Kind, DiffAdd)
	}
	if r1.Right.Text != "new2" {
		t.Errorf("row 1 Right.Text: got %q, want %q", r1.Right.Text, "new2")
	}

	// Row 2: no del to pair — left side must be empty
	r2 := rows[2]
	if !r2.Left.Empty {
		t.Errorf("row 2 Left.Empty: got false, want true (unpaired add)")
	}
	if r2.Right.Line != 3 {
		t.Errorf("row 2 Right.Line: got %d, want 3", r2.Right.Line)
	}
	if r2.Right.Kind != DiffAdd {
		t.Errorf("row 2 Right.Kind: got %q, want %q", r2.Right.Kind, DiffAdd)
	}
	if r2.Right.Text != "new3" {
		t.Errorf("row 2 Right.Text: got %q, want %q", r2.Right.Text, "new3")
	}
	if r2.Right.Empty {
		t.Errorf("row 2 Right.Empty: got true, want false")
	}
}

// TestBuildViewDiffSplitMoreDelsThanAdds verifies that when there are more del
// lines than add lines, the unpaired del rows get an Empty right side.
//
// Scenario: Del/Add blocks: unpaired lines get Empty: true opposite side
// (more dels than adds case).
func TestBuildViewDiffSplitMoreDelsThanAdds(t *testing.T) {
	df := &DiffFile{Path: "moredel.go", Hunks: []DiffHunk{{
		OldStart: 1,
		OldCount: 3,
		NewStart: 1,
		NewCount: 1,
		Lines: []DiffLine{
			{Kind: DiffDel, OldLine: 1, NewLine: 0, Text: "del1"},
			{Kind: DiffDel, OldLine: 2, NewLine: 0, Text: "del2"},
			{Kind: DiffDel, OldLine: 3, NewLine: 0, Text: "del3"},
			{Kind: DiffAdd, OldLine: 0, NewLine: 1, Text: "add1"},
		},
	}}}

	hunks := buildViewDiffSplit(df, nil, 0, 0, false, 0, "")

	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}
	rows := hunks[0].Rows
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	// Row 0: del1 paired with add1
	r0 := rows[0]
	if r0.Left.Kind != DiffDel {
		t.Errorf("row 0 Left.Kind: got %q, want %q", r0.Left.Kind, DiffDel)
	}
	if r0.Left.Text != "del1" {
		t.Errorf("row 0 Left.Text: got %q, want %q", r0.Left.Text, "del1")
	}
	if r0.Left.Empty {
		t.Errorf("row 0 Left.Empty: got true, want false")
	}
	if r0.Right.Kind != DiffAdd {
		t.Errorf("row 0 Right.Kind: got %q, want %q", r0.Right.Kind, DiffAdd)
	}
	if r0.Right.Text != "add1" {
		t.Errorf("row 0 Right.Text: got %q, want %q", r0.Right.Text, "add1")
	}
	if r0.Right.Empty {
		t.Errorf("row 0 Right.Empty: got true, want false")
	}

	// Row 1: del2 unpaired — right side must be empty
	r1 := rows[1]
	if r1.Left.Kind != DiffDel {
		t.Errorf("row 1 Left.Kind: got %q, want %q", r1.Left.Kind, DiffDel)
	}
	if r1.Left.Text != "del2" {
		t.Errorf("row 1 Left.Text: got %q, want %q", r1.Left.Text, "del2")
	}
	if r1.Left.Empty {
		t.Errorf("row 1 Left.Empty: got true, want false")
	}
	if !r1.Right.Empty {
		t.Errorf("row 1 Right.Empty: got false, want true (unpaired del)")
	}

	// Row 2: del3 unpaired — right side must be empty
	r2 := rows[2]
	if r2.Left.Kind != DiffDel {
		t.Errorf("row 2 Left.Kind: got %q, want %q", r2.Left.Kind, DiffDel)
	}
	if r2.Left.Text != "del3" {
		t.Errorf("row 2 Left.Text: got %q, want %q", r2.Left.Text, "del3")
	}
	if r2.Left.Empty {
		t.Errorf("row 2 Left.Empty: got true, want false")
	}
	if !r2.Right.Empty {
		t.Errorf("row 2 Right.Empty: got false, want true (unpaired del)")
	}
}

// TestBuildViewDiffSplitSelection verifies that selection is applied
// independently to each side: selectionSide="old" selects only the left (del)
// side when OldLine is in range, and the right (add) side is not selected.
//
// Scenario: Selection works on each side independently
func TestBuildViewDiffSplitSelection(t *testing.T) {
	df := &DiffFile{Path: "sel.go", Hunks: []DiffHunk{{
		OldStart: 5,
		OldCount: 1,
		NewStart: 5,
		NewCount: 1,
		Lines: []DiffLine{
			{Kind: DiffDel, OldLine: 5, NewLine: 0, Text: "old text"},
			{Kind: DiffAdd, OldLine: 0, NewLine: 5, Text: "new text"},
		},
	}}}

	// selectionSide="old", range [5,5]: only the del line (OldLine=5) should be selected.
	hunks := buildViewDiffSplit(df, nil, 5, 5, false, 0, "old")

	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}
	rows := hunks[0].Rows
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	r := rows[0]
	// Left side (del, OldLine=5) must be selected.
	if !r.Left.Selected {
		t.Errorf("Left (del, OldLine=5) should be Selected when selectionSide='old' and range=[5,5]")
	}
	// Right side (add, OldLine=0) must NOT be selected.
	if r.Right.Selected {
		t.Errorf("Right (add) should NOT be selected when selectionSide='old'")
	}
}

// TestBuildViewDiffSplitComments verifies that comments are projected to the
// correct side: old-side comments project to the left, new-side comments to
// the right.
//
// Scenario: Comments project to correct side
func TestBuildViewDiffSplitComments(t *testing.T) {
	df := &DiffFile{Path: "cmt.go", Hunks: []DiffHunk{{
		OldStart: 3,
		OldCount: 1,
		NewStart: 4,
		NewCount: 1,
		Lines: []DiffLine{
			{Kind: DiffContext, OldLine: 3, NewLine: 4, Text: "context text"},
		},
	}}}

	oldComment := Comment{ID: 10, Path: "cmt.go", StartLine: 3, EndLine: 3, Text: "old comment", Side: "old"}
	newComment := Comment{ID: 11, Path: "cmt.go", StartLine: 4, EndLine: 4, Text: "new comment", Side: ""}
	comments := []Comment{oldComment, newComment}

	hunks := buildViewDiffSplit(df, comments, 0, 0, false, 0, "")

	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}
	rows := hunks[0].Rows
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	r := rows[0]

	// Left side uses "old" side for comments.
	if !r.Left.Commented {
		t.Errorf("Left side (OldLine=3) should be Commented by the old-side comment")
	}
	if len(r.Left.Comments) != 1 {
		t.Errorf("Left side should have 1 comment, got %d", len(r.Left.Comments))
	} else if r.Left.Comments[0].ID != 10 {
		t.Errorf("Left side comment ID: got %d, want 10", r.Left.Comments[0].ID)
	}

	// Right side uses "" (new) side for comments.
	if !r.Right.Commented {
		t.Errorf("Right side (NewLine=4) should be Commented by the new-side comment")
	}
	if len(r.Right.Comments) != 1 {
		t.Errorf("Right side should have 1 comment, got %d", len(r.Right.Comments))
	} else if r.Right.Comments[0].ID != 11 {
		t.Errorf("Right side comment ID: got %d, want 11", r.Right.Comments[0].ID)
	}
}

// TestBuildViewDiffSplitHunkHeader verifies that the Header field of a
// returned ViewDiffSplitHunk matches the standard unified-diff format.
//
// Scenario: Hunk header is set correctly
func TestBuildViewDiffSplitHunkHeader(t *testing.T) {
	df := &DiffFile{Path: "hdr.go", Hunks: []DiffHunk{{
		OldStart: 10,
		OldCount: 3,
		NewStart: 12,
		NewCount: 4,
		Lines: []DiffLine{
			{Kind: DiffContext, OldLine: 10, NewLine: 12, Text: "a"},
			{Kind: DiffContext, OldLine: 11, NewLine: 13, Text: "b"},
			{Kind: DiffContext, OldLine: 12, NewLine: 14, Text: "c"},
		},
	}}}

	hunks := buildViewDiffSplit(df, nil, 0, 0, false, 0, "")

	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}

	want := fmt.Sprintf("@@ -%d,%d +%d,%d @@", 10, 3, 12, 4)
	if hunks[0].Header != want {
		t.Errorf("hunk Header: got %q, want %q", hunks[0].Header, want)
	}
}
