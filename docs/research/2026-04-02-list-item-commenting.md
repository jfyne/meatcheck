---
date: 2026-04-02T00:00:00Z
topic: "Enable per-list-item commenting in markdown review"
source: "engineer prompt"
tags: [research, markdown, commenting, goldmark, ui]
---

# Research: Enable per-list-item commenting in markdown review

## Summary

> Meatcheck currently treats markdown lists as single blocks, making it impossible to comment on individual list items. The goal is to allow users to click on a specific list item and leave a comment anchored to just that item's line range.

## Research Questions

### Codebase Exploration

1. In `renderMarkdownBlocks()` (internal/app/assets.go), the function iterates top-level AST children to produce `MarkdownBlock` slices. How exactly does the `nodeByteRange()` helper compute byte ranges for a node, and can it accurately isolate individual `ast.ListItem` children within an `ast.List` node? What happens with the byte-to-line mapping when list items span multiple lines (e.g., items with continuation paragraphs or nested content)?

2. The `MarkdownBlock` struct (internal/app/model.go) carries `StartLine`, `EndLine`, and `HTML`. If a list is split into per-item blocks, each item would need to be rendered as a standalone HTML fragment (e.g., `<li>...</li>` without the wrapping `<ul>`). How does goldmark's renderer handle rendering a single `ast.ListItem` node in isolation — does it produce valid HTML, or does it require the parent `ast.List` context for attributes like list type (ordered vs unordered) and start number?

3. The template (internal/ui/template.html) renders markdown blocks inside `<div class="md-block">` wrappers with `data-line` and `data-line-end` attributes. If list items become individual blocks, the `<ul>` or `<ol>` wrapper must still exist for valid HTML and correct styling. How is the template currently structured for markdown block iteration, and what changes would be needed to support a "grouped blocks" concept where individual items are commentable but visually grouped under a shared list container?

4. Comment projection in `view.go` (`projectBlockComments()`) maps comments to blocks by checking line-range overlap. If a legacy comment was anchored to the full list range (e.g., lines 6-8) but the list is now split into per-item blocks (6-6, 7-7, 8-8), how does the projection logic handle this? Would existing comments still display correctly, or would they be orphaned?

5. The CSS in `styles.css` styles `.md-block` with hover highlights, selection backgrounds, and left-border accents for commented blocks. If list items become individual `.md-block` elements, the visual grouping of the list could break — there would be separate hover/selection states per item with visible gaps. How are `.md-block` elements currently spaced and bordered, and what CSS changes would be needed to make per-item blocks appear as a cohesive list while still being individually interactive?

6. Are there other markdown block types besides lists that have the same single-block-for-nested-content behavior? Specifically, how does `renderMarkdownBlocks()` handle blockquotes (which can contain multiple paragraphs), tables (which have rows), and definition lists? Understanding the full scope of "container blocks" that might benefit from per-child commenting helps avoid solving only the list case.

7. The JavaScript click handler (template.html, around line 338-359) finds the closest `[data-line]` element when a user clicks. If per-item blocks are nested inside a list wrapper, how does event bubbling interact with `data-line` attribute lookup? Could a click on a list item accidentally match the parent list's `data-line` instead of the item's?

### External Research

1. Goldmark's AST represents lists as `ast.List` containing `ast.ListItem` children, and each `ListItem` can contain block-level content (paragraphs, nested lists, code blocks). What does goldmark's API provide for rendering a subtree of the AST rather than the full document — is there a way to render individual child nodes while preserving their parent's context (list type, start number, tightness)?

2. GitHub's pull request review UI allows line-level commenting on markdown files rendered as diffs. How does GitHub handle the mapping between rendered markdown blocks and commentable line ranges — do they allow per-list-item commenting, or do they also treat lists as single blocks? What about other code review tools (GitLab, Gerrit, Reviewable)?

3. The goldmark GFM extension adds task list support (checkboxes in list items). If list items are split into individual blocks, does the GFM extension's rendering of `- [ ] item` still work correctly when a `ListItem` is rendered outside its parent `List` context? Are there other GFM-specific list behaviors (like autolinks in list items) that could be affected?

## Detailed Findings

### 1. How `renderMarkdownBlocks()` and `nodeByteRange()` handle lists

The `renderMarkdownBlocks()` function (`internal/app/assets.go:115-195`) iterates only direct children of the document AST node:

```go
for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
```

When goldmark parses a markdown list, it produces a single `ast.List` top-level node containing `ast.ListItem` children. The loop treats this entire `ast.List` as one `MarkdownBlock`.

The `nodeByteRange()` helper (`internal/app/assets.go:199-240`) recursively searches all descendants for byte positions. For an `ast.List` node, it aggregates byte ranges across all `ast.ListItem` children and their content, returning the combined [start, end) range spanning the full list.

The `byteToLine()` closure (`internal/app/assets.go:140-143`) uses `sort.SearchInts` on a prebuilt `lineStarts` slice to convert byte offsets to 1-based line numbers with frontmatter offset. When applied to individual `ast.ListItem` nodes, `nodeByteRange()` correctly isolates per-item byte ranges — each `ListItem` has its own text segments. Multi-line list items (with continuation paragraphs or nested content) span multiple line-start entries and `byteToLine()` maps them to the correct range.

The existing test `TestRenderMarkdownBlocksLineNumbers` (`internal/app/markdown_test.go:99-130`) confirms the current behavior: input `"# Heading\n\nParagraph text\nwith two lines.\n\n- item 1\n- item 2\n"` produces a list block starting at line 6, with the entire list treated as one block.

### 2. Goldmark's rendering of isolated `ast.ListItem` nodes

Goldmark's `Renderer.Render()` calls `ast.Walk()` starting at the supplied node and fires registered renderer functions for every node visited. It does not walk ancestors.

When an `ast.ListItem` is passed to `Render()`:
- The `renderListItem` function writes `<li>` and `</li>` tags.
- No `<ul>` or `<ol>` wrapper is emitted — those tags are the responsibility of `renderList`, which reads `n.IsOrdered()` on the parent `ast.List` node.
- The `Start` attribute (ordered list start number) is stored on `ast.List` and used only in `renderList`, not `renderListItem`.

**Tight vs. loose lists** are handled structurally in the AST, not at render time. During parsing, goldmark's list parser `Close()` method replaces `Paragraph` nodes with `TextBlock` nodes inside tight list items. The `renderTextBlock` function renders text without `<p>` wrappers, while `renderParagraph` includes them. This structural difference is baked into the `ListItem`'s children before `Render()` is called, so rendering a `ListItem` in isolation correctly reflects tight/loose behavior without needing the parent `ast.List.IsTight` field.

The `renderListItem` function checks:
```go
fc := n.FirstChild()
if fc != nil {
    if _, ok := fc.(*ast.TextBlock); !ok {
        _ = w.WriteByte('\n')
    }
}
```
This newline insertion depends only on the child type, not the parent.

### 3. Template structure and click handling for markdown blocks

The markdown block rendering section of `internal/ui/template.html:222-247`:

```html
<div class="markdown-file-preview markdown" id="code-view-{{.CodeViewKey}}">
  {{range .ViewFile.MarkdownBlocks}}
    <div class="md-block..." id="line-{{id $root.SelectedPath}}-{{.StartLine}}">
      <div class="md-block-content" data-line="{{.StartLine}}" data-line-end="{{.EndLine}}">
        {{.HTML}}
      </div>
      {{/* comment thread and inline comment form */}}
    </div>
  {{end}}
</div>
```

Each `MarkdownBlock` becomes a `<div class="md-block">` with a nested `<div class="md-block-content">` carrying `data-line` and `data-line-end` attributes. The click handler (`template.html:338-359`) uses `target.closest("[data-line], [data-old-line]")` to find the nearest ancestor with line data.

When a user clicks inside a list rendered as a single block, the click event bubbles up to the `.md-block-content` div, which has `data-line` set to the list's start line and `data-line-end` set to the list's end line. The handler sends `select-line` with `line: startLine, line_end: endLine`, selecting the entire list.

If list items were split into per-item blocks nested inside a list wrapper, and each item's `<div>` had its own `data-line`/`data-line-end`, `closest()` would match the innermost (item-level) element first due to DOM traversal order. Event bubbling travels from the target upward, and `closest()` searches from the element outward — so the item's `data-line` would take precedence over any parent's.

### 4. Comment projection for blocks

The `projectBlockComments()` function (`internal/app/view.go:339-358`) maps comments to blocks using line-range overlap:

```go
func projectBlockComments(path string, startLine, endLine int, comments []Comment, editingID int) (bool, []ViewComment) {
    for _, c := range comments {
        if c.Path != path { continue }
        if c.StartLine <= endLine && c.EndLine >= startLine {
            commented = true
        }
        if c.StartLine >= startLine && c.StartLine <= endLine {
            blockComments = append(blockComments, ...)
        }
    }
}
```

Two separate checks exist:
1. **Commented flag** (`c.StartLine <= endLine && c.EndLine >= startLine`): Any overlap marks the block as commented.
2. **Comment attachment** (`c.StartLine >= startLine && c.StartLine <= endLine`): Comments attach to the block containing their `StartLine`.

If a legacy comment was anchored to lines 6-8 (full list) and the list is split into per-item blocks (6-6, 7-7, 8-8):
- The comment's `StartLine=6` falls within block 6-6, so the comment attaches to the first item.
- The `Commented` flag is set for all three blocks because the comment's range 6-8 overlaps with each.
- The comment text displays under the first item block (where `StartLine` falls), and all three items show the commented visual indicator.

### 5. CSS styling of `.md-block` elements

The `.md-block` CSS (`internal/ui/styles.css:329-360`):

```css
.md-block {
  max-width: 980px;
  padding: 4px 32px;
  cursor: pointer;
  border-left: 3px solid transparent;
  transition: background 100ms ease;
}
.md-block:hover { background: var(--line-hover); }
.md-block.selected { background: var(--line-selected); }
.md-block.commented { border-left-color: var(--accent); }
```

Each `.md-block` has independent hover, selection, and comment indicator states. There is no margin or gap between blocks — they stack with `4px` padding top and bottom. If list items become individual `.md-block` elements:
- Each item gets its own hover highlight, selection background, and comment border.
- The `<ul>`/`<ol>` wrapper would need to exist outside the `.md-block` elements for valid HTML and correct list styling (bullets, numbers, indentation).
- The `padding-left: 2em` on `.markdown ul, .markdown ol` (`styles.css:577-579`) provides list indentation. If `.md-block` wraps each `<li>`, this padding applies per-block rather than to a single `<ul>` wrapper.

### 6. Other container block types with single-block behavior

All goldmark block types that contain child block elements exhibit the same single-block behavior in `renderMarkdownBlocks()`:

| AST Node | Rendered HTML | Children | Current behavior |
|---|---|---|---|
| `ast.List` | `<ul>` or `<ol>` | `ast.ListItem` | Single block for entire list |
| `ast.Blockquote` | `<blockquote>` | Paragraphs, lists, code blocks | Single block for entire blockquote |
| `ast.Table` (GFM) | `<table>` | `TableHeader`, `TableRow` | Single block for entire table |

No tests exercise blockquotes or tables as separate blocks. The existing tests cover: headings, paragraphs, lists (as single blocks), and frontmatter.

Blockquotes can contain multiple paragraphs spanning many lines. Tables can have many rows. Both are treated as atomic blocks.

### 7. JavaScript event bubbling and `data-line` lookup

The click handler (`template.html:338-359`):

```javascript
root.addEventListener("click", (ev) => {
  const target = ev.target;
  const clickable = target.closest("[data-line], [data-old-line]");
  if (!clickable) return;
  var line = Number(clickable.dataset.line || 0);
  const lineEnd = Number(clickable.dataset.lineEnd || 0);
  // ...
  window.Live.send("select-line", { line: line, line_end: lineEnd || line, shift: shift });
});
```

`Element.closest()` traverses from the element itself up through its ancestors, returning the first match. If nested elements both have `data-line`, the innermost one wins. The handler early-returns if the click is inside `.inline-comment`, `.line-comment-thread`, or `.comment-form`.

For the `select-line` server handler (`internal/app/app.go:291-333`): it receives `line` and `line_end` params. When `line_end < line`, it normalizes to `lineEnd = line`. When shift is held and a prior selection exists, it extends the range. The server stores `SelectionStart` and `SelectionEnd` on the model, then calls `updateView()` to rebuild the view with selection/comment state applied.

## Code References

| File | Lines | Role |
|---|---|---|
| `internal/app/assets.go` | 115-195 | `renderMarkdownBlocks()` — top-level AST traversal producing `[]MarkdownBlock` |
| `internal/app/assets.go` | 199-240 | `nodeByteRange()` — recursive byte range computation for AST nodes |
| `internal/app/assets.go` | 140-143 | `byteToLine()` closure — byte offset to 1-based line number mapping |
| `internal/app/assets.go` | 163 | Loop: `for child := doc.FirstChild(); child != nil; child = child.NextSibling()` — only iterates top-level nodes |
| `internal/app/model.go` | 50-57 | `MarkdownBlock` struct definition |
| `internal/app/model.go` | 8-15 | `Comment` struct definition |
| `internal/app/view.go` | 36-49 | Selection and comment projection onto markdown blocks in `updateFileView()` |
| `internal/app/view.go` | 339-358 | `projectBlockComments()` — line-range overlap matching |
| `internal/ui/template.html` | 222-247 | Markdown block rendering template with `data-line`/`data-line-end` |
| `internal/ui/template.html` | 338-359 | JavaScript click handler using `closest("[data-line]")` |
| `internal/ui/styles.css` | 329-360 | `.md-block` styling: hover, selected, commented states |
| `internal/ui/styles.css` | 576-579 | `.markdown ul, .markdown ol` padding |
| `internal/app/app.go` | 291-333 | `select-line` event handler — sets `SelectionStart`/`SelectionEnd` |
| `internal/app/app.go` | 335-363 | `add-comment` event handler — creates `Comment` from selection range |
| `internal/app/markdown_test.go` | 99-130 | `TestRenderMarkdownBlocksLineNumbers` — confirms list as single block |

## External Context

### Goldmark rendering API

Goldmark's `Renderer.Render(w, source, node)` calls `ast.Walk()` from the given node downward. Passing a child node (e.g., `ast.ListItem`) renders only that subtree. The `<ul>`/`<ol>` wrapper is emitted by `renderList`, not `renderListItem`, so rendering a `ListItem` in isolation produces `<li>...</li>` without a wrapper.

The tight/loose distinction is structural: tight lists have `TextBlock` children (no `<p>` tags), loose lists have `Paragraph` children. This is baked into the AST at parse time and does not require the parent `ast.List` at render time.

`ast.Node.AppendChild()` moves (not copies) nodes — calling it removes the node from its original parent via `ensureIsolated()`. Goldmark has no built-in node-clone API. Building a temporary `ast.List` wrapper for rendering requires either re-parsing the source fragment or accepting that the original AST is mutated.

Source: [yuin/goldmark](https://github.com/yuin/goldmark), [renderer/html/html.go](https://github.com/yuin/goldmark/blob/master/renderer/html/html.go), [goldmark Discussion #186](https://github.com/yuin/goldmark/discussions/186)

### GFM task list checkboxes

The GFM extension's `TaskCheckBoxHTMLRenderer` reads only `node.(*TaskCheckBox).IsChecked` to decide between `<input checked="" .../>` and `<input .../>`. It does not reference the parent `ListItem` or `List`. The `TaskCheckBox` is an inline node embedded in the item's content. Rendering a `ListItem` subtree in isolation correctly renders task checkboxes.

Source: [extension/tasklist.go](https://github.com/yuin/goldmark/blob/master/extension/tasklist.go)

### Code review tool comparison

| Tool | Rendered view in diff? | Per-list-item commenting? | Anchor mechanism |
|---|---|---|---|
| GitHub PR | Yes (toggle, read-only) | No — source view only | Raw text line number |
| GitLab MR | No rendered view in diffs | No — line-level on raw text | Raw text line number |
| Notion | Always rendered | Yes — each bullet is a block | Block ID |
| Confluence | Always rendered | Yes — by text selection | Character range |

GitHub's rich diff for markdown in PRs is read-only; commenting requires switching to source view and is line-level. GitLab does not render markdown in MR diffs at all. Notion treats each list item as a separate block with its own ID, making them independently commentable. Confluence anchors comments to highlighted text selections.

Sources: [GitHub Docs — Working with non-code files](https://docs.github.com/en/github/managing-files-in-a-repository/working-with-non-code-files/rendering-differences-in-prose-documents), [GitHub Community Discussion #16289](https://github.com/orgs/community/discussions/16289), [GitLab Issue #17000](https://gitlab.com/gitlab-org/gitlab/-/issues/17000), [Notion API — Block reference](https://developers.notion.com/reference/block)

## Open Questions

_None — all research questions resolved._
