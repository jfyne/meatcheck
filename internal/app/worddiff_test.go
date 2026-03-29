package app

import (
	"html/template"
	"strings"
	"testing"
)

func TestTokenizeWords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"simple words", "foo bar", []string{"foo", " ", "bar"}},
		{"punctuation", "a.b(c)", []string{"a", ".", "b", "(", "c", ")"}},
		{"mixed whitespace", "\tfoo  bar", []string{"\t", "foo", "  ", "bar"}},
		{"empty", "", nil},
		{"only whitespace", "   ", []string{"   "}},
		{"code tokens", "func foo(x int)", []string{"func", " ", "foo", "(", "x", " ", "int", ")"}},
		{"underscores in words", "my_var = 1", []string{"my_var", " ", "=", " ", "1"}},
		{"digits", "x123 + y456", []string{"x123", " ", "+", " ", "y456"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenizeWords(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("tokenizeWords(%q): got %d tokens %v, want %d tokens %v", tt.input, len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("tokenizeWords(%q)[%d]: got %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
			// Verify lossless reconstruction.
			if joined := strings.Join(got, ""); joined != tt.input {
				t.Errorf("tokenizeWords(%q): joined tokens = %q, want %q", tt.input, joined, tt.input)
			}
		})
	}
}

func TestDiffWordsIdentical(t *testing.T) {
	tokens := []string{"foo", " ", "bar", " ", "baz"}
	edits := diffWords(tokens, tokens)
	for i, e := range edits {
		if e.kind != wordEqual {
			t.Errorf("edit[%d]: got kind %v, want wordEqual for identical input", i, e.kind)
		}
	}
}

func TestDiffWordsCompletelyDifferent(t *testing.T) {
	old := []string{"aaa", " ", "bbb"}
	new := []string{"xxx", " ", "yyy"}
	edits := diffWords(old, new)
	hasDelete := false
	hasInsert := false
	for _, e := range edits {
		switch e.kind {
		case wordDelete:
			hasDelete = true
		case wordInsert:
			hasInsert = true
		}
	}
	if !hasDelete {
		t.Error("expected at least one delete edit")
	}
	if !hasInsert {
		t.Error("expected at least one insert edit")
	}
}

func TestDiffWordsSingleTokenChange(t *testing.T) {
	old := tokenizeWords("foo bar baz")
	new := tokenizeWords("foo qux baz")
	edits := diffWords(old, new)

	// Reconstruct what was deleted and inserted.
	var deleted, inserted []string
	for _, e := range edits {
		switch e.kind {
		case wordDelete:
			deleted = append(deleted, e.text)
		case wordInsert:
			inserted = append(inserted, e.text)
		}
	}
	if len(deleted) != 1 || deleted[0] != "bar" {
		t.Errorf("deleted: got %v, want [bar]", deleted)
	}
	if len(inserted) != 1 || inserted[0] != "qux" {
		t.Errorf("inserted: got %v, want [qux]", inserted)
	}
}

func TestDiffWordsInsertToken(t *testing.T) {
	old := tokenizeWords("a b")
	new := tokenizeWords("a c b")
	edits := diffWords(old, new)

	var inserted []string
	for _, e := range edits {
		if e.kind == wordInsert {
			inserted = append(inserted, e.text)
		}
	}
	if len(inserted) == 0 {
		t.Error("expected at least one insert")
	}
	found := false
	for _, s := range inserted {
		if s == "c" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'c' in inserted tokens, got %v", inserted)
	}
}

func TestDiffWordsDeleteToken(t *testing.T) {
	old := tokenizeWords("a b c")
	new := tokenizeWords("a c")
	edits := diffWords(old, new)

	var deleted []string
	for _, e := range edits {
		if e.kind == wordDelete {
			deleted = append(deleted, e.text)
		}
	}
	found := false
	for _, s := range deleted {
		if s == "b" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'b' in deleted tokens, got %v", deleted)
	}
}

func TestRenderIntraLineHTMLSimple(t *testing.T) {
	oldHTML, newHTML := renderIntraLineHTML("return nil", "return err")
	if oldHTML == "" {
		t.Fatal("expected non-empty oldHTML")
	}
	if newHTML == "" {
		t.Fatal("expected non-empty newHTML")
	}
	if !strings.Contains(string(oldHTML), `<span class="intra-del">nil</span>`) {
		t.Errorf("oldHTML should contain intra-del span for 'nil', got: %s", oldHTML)
	}
	if !strings.Contains(string(newHTML), `<span class="intra-add">err</span>`) {
		t.Errorf("newHTML should contain intra-add span for 'err', got: %s", newHTML)
	}
	// Unchanged part should not be wrapped.
	if strings.Contains(string(oldHTML), `<span class="intra-del">return`) {
		t.Error("oldHTML should not wrap 'return' in intra-del")
	}
	if strings.Contains(string(newHTML), `<span class="intra-add">return`) {
		t.Error("newHTML should not wrap 'return' in intra-add")
	}
}

func TestRenderIntraLineHTMLEscaping(t *testing.T) {
	oldHTML, newHTML := renderIntraLineHTML("a < b", "a > b")
	if oldHTML == "" || newHTML == "" {
		t.Fatal("expected non-empty HTML")
	}
	// The < and > should be escaped.
	if strings.Contains(string(oldHTML), "<") && !strings.Contains(string(oldHTML), "&lt;") && !strings.Contains(string(oldHTML), "<span") {
		t.Error("oldHTML should have escaped '<'")
	}
	if !strings.Contains(string(newHTML), "&gt;") {
		t.Errorf("newHTML should contain escaped '>', got: %s", newHTML)
	}
	// The span tags themselves should not be escaped.
	if !strings.Contains(string(oldHTML), `<span class="intra-del">`) {
		t.Errorf("oldHTML should contain intra-del span, got: %s", oldHTML)
	}
}

func TestRenderIntraLineHTMLThreshold(t *testing.T) {
	oldHTML, newHTML := renderIntraLineHTML("completely different line", "nothing alike here at all")
	if oldHTML != "" {
		t.Errorf("expected empty oldHTML for threshold exceeded, got: %s", oldHTML)
	}
	if newHTML != "" {
		t.Errorf("expected empty newHTML for threshold exceeded, got: %s", newHTML)
	}
}

func TestRenderIntraLineHTMLWhitespaceOnly(t *testing.T) {
	oldHTML, newHTML := renderIntraLineHTML("a  b", "a b")
	// Should produce non-empty output highlighting the whitespace change.
	if oldHTML == "" || newHTML == "" {
		t.Fatal("expected non-empty HTML for whitespace-only change")
	}
}

func TestRenderIntraLineHTMLEmpty(t *testing.T) {
	// One side empty.
	oldHTML, newHTML := renderIntraLineHTML("", "something")
	if oldHTML != "" || newHTML != "" {
		t.Error("expected empty HTML when one side is empty")
	}
	// Both sides empty.
	oldHTML, newHTML = renderIntraLineHTML("", "")
	if oldHTML != "" || newHTML != "" {
		t.Error("expected empty HTML when both sides are empty")
	}
}

func TestRenderIntraLineHTMLMultipleChanges(t *testing.T) {
	oldHTML, newHTML := renderIntraLineHTML(
		"func process(data string) error",
		"func process(data string, strict bool) error",
	)
	if oldHTML == "" || newHTML == "" {
		t.Fatal("expected non-empty HTML for multiple changes")
	}
	// The inserted ", strict bool" tokens should be wrapped.
	if !strings.Contains(string(newHTML), `intra-add`) {
		t.Errorf("newHTML should contain intra-add span, got: %s", newHTML)
	}
}

func TestApplyIntraLineDiff(t *testing.T) {
	var delHTML, addHTML template.HTML
	applyIntraLineDiff(&delHTML, &addHTML, "return nil", "return err")
	if delHTML == "" {
		t.Error("expected delHTML to be set")
	}
	if addHTML == "" {
		t.Error("expected addHTML to be set")
	}
	if !strings.Contains(string(delHTML), "intra-del") {
		t.Errorf("delHTML should contain intra-del, got: %s", delHTML)
	}
	if !strings.Contains(string(addHTML), "intra-add") {
		t.Errorf("addHTML should contain intra-add, got: %s", addHTML)
	}
}

func TestApplyIntraLineDiffThresholdKeepsExisting(t *testing.T) {
	delHTML := template.HTML("original-del")
	addHTML := template.HTML("original-add")
	applyIntraLineDiff(&delHTML, &addHTML, "completely different line", "nothing alike here at all")
	// Should be unchanged when threshold is exceeded.
	if delHTML != "original-del" {
		t.Errorf("delHTML should be unchanged, got: %s", delHTML)
	}
	if addHTML != "original-add" {
		t.Errorf("addHTML should be unchanged, got: %s", addHTML)
	}
}
