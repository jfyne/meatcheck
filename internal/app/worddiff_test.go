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

// TestEscapeWordTokenPreservesSpaces verifies that escapeWordToken replaces
// ASCII spaces with &nbsp; entities so leading/trailing whitespace inside an
// inline highlight span survives the browser's default whitespace collapsing.
//
// Regression: a plain html.EscapeString left whitespace un-entitized, which
// caused adjacent highlighted words to visually merge in diff output.
func TestEscapeWordTokenPreservesSpaces(t *testing.T) {
	got := escapeWordToken("  ")
	want := "&nbsp;&nbsp;"
	if got != want {
		t.Errorf("escapeWordToken(\"  \"): got %q, want %q", got, want)
	}
	// Mixed spaces and letters.
	got = escapeWordToken(" foo ")
	want = "&nbsp;foo&nbsp;"
	if got != want {
		t.Errorf("escapeWordToken(\" foo \"): got %q, want %q", got, want)
	}
}

// TestEscapeWordTokenPreservesTabs verifies that escapeWordToken expands a tab
// into four &nbsp; entities so tab indentation remains visible inside highlight
// spans.
//
// Regression: tabs previously collapsed to a single space character in the
// rendered HTML.
func TestEscapeWordTokenPreservesTabs(t *testing.T) {
	got := escapeWordToken("\t")
	want := "&nbsp;&nbsp;&nbsp;&nbsp;"
	if got != want {
		t.Errorf("escapeWordToken(\"\\t\"): got %q, want %q", got, want)
	}
	// Mixed tab and content.
	got = escapeWordToken("\tfoo")
	want = "&nbsp;&nbsp;&nbsp;&nbsp;foo"
	if got != want {
		t.Errorf("escapeWordToken(\"\\tfoo\"): got %q, want %q", got, want)
	}
}

// TestEscapeWordTokenHTMLEscapesFirst verifies that HTML metacharacters are
// escaped before whitespace substitution so the output remains safe for
// template injection.
//
// Regression: ensure html.EscapeString runs before the space/tab substitution
// so that raw "<" or "&" from user text cannot leak as markup.
func TestEscapeWordTokenHTMLEscapesFirst(t *testing.T) {
	got := escapeWordToken("<a> b")
	// "<a>" escapes to "&lt;a&gt;", then " " -> "&nbsp;".
	want := "&lt;a&gt;&nbsp;b"
	if got != want {
		t.Errorf("escapeWordToken(\"<a> b\"): got %q, want %q", got, want)
	}
	// Ampersand already in input must be escaped to &amp; (once), not double-escaped.
	got = escapeWordToken("a & b")
	want = "a&nbsp;&amp;&nbsp;b"
	if got != want {
		t.Errorf("escapeWordToken(\"a & b\"): got %q, want %q", got, want)
	}
}

// TestRenderIntraLineHTMLSpaceTokenInsideHighlight verifies that when an
// inserted or deleted edit is itself a whitespace token, the resulting span
// contains &nbsp; (not a raw space). A span whose only text is a literal " "
// can be visually collapsed by the browser at the inline-box boundary, so
// changed-whitespace tokens must use the entity form.
//
// Regression: previously every whitespace character (even unchanged ones)
// was rewritten to &nbsp;, which broke indentation alignment with context
// lines. This test specifically targets the *changed* whitespace case so that
// fix doesn't regress the original collapse problem.
func TestRenderIntraLineHTMLSpaceTokenInsideHighlight(t *testing.T) {
	// "foo bar" -> "foo baz qux" inserts the tokens ["baz", " ", "qux"], so
	// the middle insert is a pure-whitespace span.
	_, newHTML := renderIntraLineHTML("foo bar", "foo baz qux")
	if newHTML == "" {
		t.Fatal("expected non-empty newHTML")
	}
	if !strings.Contains(string(newHTML), `<span class="intra-add">&nbsp;</span>`) {
		t.Errorf("newHTML should contain a whitespace-only intra-add span as &nbsp;, got: %s", newHTML)
	}
	// And no raw-space-only intra-add span (the regression form).
	if strings.Contains(string(newHTML), `<span class="intra-add"> </span>`) {
		t.Errorf("newHTML must not emit a raw-space-only intra-add span, got: %s", newHTML)
	}
}

// TestRenderIntraLineHTMLEqualSpacesAreLiteral verifies that whitespace
// belonging to *equal* tokens (i.e. spaces between unchanged words around a
// changed word) is emitted as a literal " " — not &nbsp;. The surrounding
// white-space:pre-wrap CSS preserves these spaces, and using a literal space
// keeps highlighted lines glyph-aligned with adjacent context lines.
func TestRenderIntraLineHTMLEqualSpacesAreLiteral(t *testing.T) {
	oldHTML, newHTML := renderIntraLineHTML("foo aaa bbb baz", "foo xxx yyy baz")
	if oldHTML == "" || newHTML == "" {
		t.Fatal("expected non-empty HTML")
	}
	// "foo " and " baz" are equal — they must contain a real space, not &nbsp;.
	if !strings.HasPrefix(string(oldHTML), "foo ") {
		t.Errorf("oldHTML equal-prefix should be literal \"foo \", got: %s", oldHTML)
	}
	if !strings.HasSuffix(string(newHTML), " baz") {
		t.Errorf("newHTML equal-suffix should be literal \" baz\", got: %s", newHTML)
	}
	// And the merged-span form (the *original* regression we still guard against)
	// must not appear: a single intra-* span containing "aaa bbb" / "xxx yyy"
	// would mean we'd lumped the change tokens together with a raw inner space.
	if strings.Contains(string(oldHTML), `<span class="intra-del">aaa bbb</span>`) {
		t.Errorf("oldHTML must not lump change tokens with a raw inner space, got: %s", oldHTML)
	}
	if strings.Contains(string(newHTML), `<span class="intra-add">xxx yyy</span>`) {
		t.Errorf("newHTML must not lump change tokens with a raw inner space, got: %s", newHTML)
	}
}

// TestRenderIntraLineHTMLLeadingWhitespaceLiteral verifies that leading
// whitespace in the unchanged prefix (emitted outside any span) is preserved
// as a literal tab/space so indented lines render at the same horizontal
// position as the surrounding context lines (which never go through
// renderIntraLineHTML and therefore keep raw \t under tab-size:4 CSS).
//
// Regression: an earlier fix indiscriminately replaced every space/tab in
// equal tokens with &nbsp;, which renders a sub-pixel-different glyph from a
// real \t in some monospace fonts and made highlighted lines visually offset
// from neighbouring context lines.
func TestRenderIntraLineHTMLLeadingWhitespaceLiteral(t *testing.T) {
	oldHTML, newHTML := renderIntraLineHTML("\treturn nil", "\treturn err")
	if oldHTML == "" || newHTML == "" {
		t.Fatal("expected non-empty HTML for indented change")
	}
	// The leading tab is part of the unchanged prefix — it must remain a
	// literal \t so it renders identically to the same indent on a context
	// line in the same diff.
	if !strings.HasPrefix(string(oldHTML), "\t") {
		t.Errorf("oldHTML should start with a literal tab, got: %q", oldHTML)
	}
	if !strings.HasPrefix(string(newHTML), "\t") {
		t.Errorf("newHTML should start with a literal tab, got: %q", newHTML)
	}
	// Tab in equal-token prefix must NOT be expanded to &nbsp; runs.
	if strings.HasPrefix(string(oldHTML), "&nbsp;") {
		t.Errorf("oldHTML leading tab should not be &nbsp;-expanded, got: %q", oldHTML)
	}
}
