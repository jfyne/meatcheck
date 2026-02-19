package app

import "testing"

func TestParseUnifiedDiff(t *testing.T) {
	input := "diff --git a/hello.txt b/hello.txt\n" +
		"index 111..222 100644\n" +
		"--- a/hello.txt\n" +
		"+++ b/hello.txt\n" +
		"@@ -1,3 +1,4 @@\n" +
		" line1\n" +
		"-line2\n" +
		"+line2-new\n" +
		" line3\n" +
		"+line4\n"

	files, err := parseUnifiedDiff(input)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	f := files[0]
	if f.Path != "hello.txt" {
		t.Fatalf("expected path hello.txt, got %q", f.Path)
	}
	if len(f.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(f.Hunks))
	}
	h := f.Hunks[0]
	if h.OldStart != 1 || h.NewStart != 1 {
		t.Fatalf("unexpected hunk starts: old %d new %d", h.OldStart, h.NewStart)
	}
	if len(h.Lines) != 6 {
		t.Fatalf("expected 6 lines, got %d", len(h.Lines))
	}
	if h.Lines[1].Kind != DiffDel || h.Lines[1].OldLine != 2 || h.Lines[1].NewLine != 0 {
		t.Fatalf("unexpected delete line mapping: %+v", h.Lines[1])
	}
	if h.Lines[2].Kind != DiffAdd || h.Lines[2].NewLine != 2 {
		t.Fatalf("unexpected add line mapping: %+v", h.Lines[2])
	}
}

func TestParseUnifiedDiffFileAdd(t *testing.T) {
	input := "diff --git a/dev/null b/new.txt\n" +
		"--- /dev/null\n" +
		"+++ b/new.txt\n" +
		"@@ -0,0 +1,2 @@\n" +
		"+hello\n" +
		"+world\n"

	files, err := parseUnifiedDiff(input)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "new.txt" {
		t.Fatalf("expected new.txt, got %q", files[0].Path)
	}
}
