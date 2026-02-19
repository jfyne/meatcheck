package app

import "testing"

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
	comments := []Comment{{Path: "x.go", StartLine: 1, EndLine: 1, Text: "hi"}}
	view := buildViewDiff(df, comments, 1, 1, false)
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
