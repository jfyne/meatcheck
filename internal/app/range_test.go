package app

import "testing"

func TestParseRangeFlag(t *testing.T) {
	ranges, err := ParseRangeFlag([]string{"a.go:10-20", "a.go:30-25", "b.go:5-8"})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(ranges["a.go"]) != 2 {
		t.Fatalf("expected 2 ranges for a.go, got %d", len(ranges["a.go"]))
	}
	if ranges["a.go"][1].Start != 25 || ranges["a.go"][1].End != 30 {
		t.Fatalf("expected normalized range 25-30, got %v", ranges["a.go"][1])
	}
	if len(ranges["b.go"]) != 1 {
		t.Fatalf("expected 1 range for b.go")
	}
}

func TestNormalizeRangesMerge(t *testing.T) {
	input := []LineRange{{Start: 5, End: 10}, {Start: 9, End: 15}, {Start: 20, End: 22}}
	out := normalizeRanges(input)
	if len(out) != 2 {
		t.Fatalf("expected 2 ranges, got %d", len(out))
	}
	if out[0].Start != 5 || out[0].End != 15 {
		t.Fatalf("expected merged 5-15, got %+v", out[0])
	}
}

func TestDiffLineExists(t *testing.T) {
	df := DiffFile{Path: "x.go", Hunks: []DiffHunk{{Lines: []DiffLine{{Kind: DiffAdd, NewLine: 3}, {Kind: DiffDel, OldLine: 2}}}}}
	if !diffLineExists([]DiffFile{df}, "x.go", 3) {
		t.Fatal("expected diff line 3 to exist")
	}
	if diffLineExists([]DiffFile{df}, "x.go", 2) {
		t.Fatal("did not expect deleted line to be selectable")
	}
}
