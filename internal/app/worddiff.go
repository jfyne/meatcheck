package app

import (
	"html"
	"html/template"
	"strings"
	"unicode"
)

type wordEditKind int

const (
	wordEqual wordEditKind = iota
	wordInsert
	wordDelete
)

type wordEdit struct {
	kind wordEditKind
	text string
}

// tokenizeWords splits text into tokens by character class: word characters
// ([a-zA-Z0-9_]), whitespace, and punctuation/operators. Each contiguous run
// of one class is a single token. The tokens join losslessly back to the input.
func tokenizeWords(s string) []string {
	if s == "" {
		return nil
	}
	var tokens []string
	var cur strings.Builder
	lastClass := runeClass(rune(s[0]))
	for _, r := range s {
		c := runeClass(r)
		if c != lastClass {
			tokens = append(tokens, cur.String())
			cur.Reset()
			lastClass = c
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

const (
	classWord  = 0
	classSpace = 1
	classPunct = 2
)

func runeClass(r rune) int {
	if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
		return classWord
	}
	if unicode.IsSpace(r) {
		return classSpace
	}
	return classPunct
}

// diffWords computes a Myers diff on two token slices, returning a sequence of
// equal/insert/delete edits.
func diffWords(old, new []string) []wordEdit {
	n := len(old)
	m := len(new)
	if n == 0 && m == 0 {
		return nil
	}
	if n == 0 {
		edits := make([]wordEdit, m)
		for i, t := range new {
			edits[i] = wordEdit{kind: wordInsert, text: t}
		}
		return edits
	}
	if m == 0 {
		edits := make([]wordEdit, n)
		for i, t := range old {
			edits[i] = wordEdit{kind: wordDelete, text: t}
		}
		return edits
	}

	// Myers algorithm.
	max := n + m
	// v stores the furthest-reaching x for each diagonal k.
	// Indexed as v[k + max].
	v := make([]int, 2*max+1)
	// trace stores a copy of v at each step d.
	trace := make([][]int, 0, max)

	var found bool
outer:
	for d := 0; d <= max; d++ {
		vc := make([]int, len(v))
		copy(vc, v)
		trace = append(trace, vc)

		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[k-1+max] < v[k+1+max]) {
				x = v[k+1+max] // move down
			} else {
				x = v[k-1+max] + 1 // move right
			}
			y := x - k
			// Follow diagonal (equal tokens).
			for x < n && y < m && old[x] == new[y] {
				x++
				y++
			}
			v[k+max] = x
			if x >= n && y >= m {
				found = true
				break outer
			}
		}
	}

	if !found {
		// Fallback: shouldn't happen, but return simple delete+insert.
		var edits []wordEdit
		for _, t := range old {
			edits = append(edits, wordEdit{kind: wordDelete, text: t})
		}
		for _, t := range new {
			edits = append(edits, wordEdit{kind: wordInsert, text: t})
		}
		return edits
	}

	// Backtrack to build the edit script.
	type point struct{ x, y int }
	var path []point
	x, y := n, m
	for d := len(trace) - 1; d >= 0; d-- {
		vc := trace[d]
		k := x - y

		var prevK int
		if k == -d || (k != d && vc[k-1+max] < vc[k+1+max]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX := vc[prevK+max]
		prevY := prevX - prevK

		// Diagonal moves (equal).
		for x > prevX && y > prevY {
			x--
			y--
			path = append(path, point{x, y})
		}

		if d > 0 {
			if prevK == k+1 {
				// Down move = insert.
				path = append(path, point{-1, prevY})
			} else {
				// Right move = delete.
				path = append(path, point{prevX, -1})
			}
		}

		x = prevX
		y = prevY
	}

	// Reverse path to get forward order.
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	// Convert path to edits.
	edits := make([]wordEdit, 0, len(path))
	for _, p := range path {
		switch {
		case p.x >= 0 && p.y >= 0:
			edits = append(edits, wordEdit{kind: wordEqual, text: old[p.x]})
		case p.x < 0:
			edits = append(edits, wordEdit{kind: wordInsert, text: new[p.y]})
		default:
			edits = append(edits, wordEdit{kind: wordDelete, text: old[p.x]})
		}
	}

	return edits
}

// renderIntraLineHTML computes word-level diff between oldText and newText and
// returns HTML for each side with changed tokens wrapped in highlight spans.
// Returns empty HTML if the texts are empty, identical, or too different (>70%
// of tokens changed).
func renderIntraLineHTML(oldText, newText string) (template.HTML, template.HTML) {
	if oldText == "" || newText == "" {
		return "", ""
	}
	if oldText == newText {
		return "", ""
	}

	oldTokens := tokenizeWords(oldText)
	newTokens := tokenizeWords(newText)
	edits := diffWords(oldTokens, newTokens)

	// Check threshold: if >70% of tokens changed, bail out.
	totalTokens := len(oldTokens) + len(newTokens)
	changedTokens := 0
	for _, e := range edits {
		if e.kind != wordEqual {
			changedTokens++
		}
	}
	if totalTokens > 0 && float64(changedTokens)/float64(totalTokens) > 0.70 {
		return "", ""
	}

	// Build HTML for each side.
	var oldBuf, newBuf strings.Builder
	for _, e := range edits {
		escaped := html.EscapeString(e.text)
		switch e.kind {
		case wordEqual:
			oldBuf.WriteString(escaped)
			newBuf.WriteString(escaped)
		case wordDelete:
			oldBuf.WriteString(`<span class="intra-del">`)
			oldBuf.WriteString(escaped)
			oldBuf.WriteString(`</span>`)
		case wordInsert:
			newBuf.WriteString(`<span class="intra-add">`)
			newBuf.WriteString(escaped)
			newBuf.WriteString(`</span>`)
		}
	}

	return template.HTML(oldBuf.String()), template.HTML(newBuf.String())
}

// applyIntraLineDiff computes word-level diff HTML for a del/add pair. On
// success, overwrites both sides' HTML fields. On failure (threshold exceeded
// or empty result), leaves HTML unchanged.
func applyIntraLineDiff(delHTML *template.HTML, addHTML *template.HTML, delText, addText string) {
	oldH, newH := renderIntraLineHTML(delText, addText)
	if oldH == "" && newH == "" {
		return
	}
	*delHTML = oldH
	*addHTML = newH
}
