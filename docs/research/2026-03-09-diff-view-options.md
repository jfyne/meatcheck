---
date: 2026-03-09T00:00:00+00:00
researcher: josh
topic: "Diff view mode switching (unified/side-by-side), preference persistence, and code background color"
tags: [research, codebase, diff, ui, preferences]
last_updated: 2026-03-09
last_updated_by: josh
---

# Research: Diff View Options

## Research Question

How does the current diff view work, and what would be involved in:
1. Making the diff view switchable between unified and side-by-side modes
2. Persisting the user's choice across sessions
3. Changing the code view background from red to black

## Summary

Meatcheck currently renders diffs in a single **unified diff format** only. The diff pipeline flows through three layers: parsing (`diff.go`), view model building (`view.go`), and HTML rendering (`template.html`). The code view area background uses `var(--panel)` which resolves to `#161014` — a very dark reddish-brown. There is no side-by-side diff view and no mechanism for persisting view preferences beyond sidebar width (stored in `localStorage`).

## Detailed Findings

### Diff Parsing

The `parseUnifiedDiff()` function in `internal/app/diff.go` parses standard Git unified diff output into structured data:

- `DiffLineKind` enum: `DiffContext`, `DiffAdd`, `DiffDel`
- `DiffLine`: stores kind, old line number, new line number, and text
- `DiffHunk`: groups lines with old/new start positions and counts
- `DiffFile`: contains file path and an array of hunks

The parser is a line-by-line state machine that classifies lines by their prefix character (`+`, `-`, ` `). Deleted lines have `NewLine=0`, added lines have `OldLine=0`, context lines have both.

### View Model Building

`buildViewDiff()` in `internal/app/view.go:119` converts parsed `DiffFile` data into `ViewDiffFile` for template rendering:

- Iterates hunks, building hunk header strings (`@@ -oldStart,oldCount +newStart,newCount @@`)
- For each line: maps kind, line numbers, raw text, optional syntax-highlighted HTML
- Selection is only allowed on new-file lines (not deletions): `selectable := dl.NewLine > 0 && dl.Kind != DiffDel`
- Comments can only anchor to new file lines via `projectLineComments()`

When `side-by-side` mode is introduced, this function will need a parallel variant that pairs old/new lines together rather than interleaving them sequentially.

### HTML Template (Unified View)

The diff section in `internal/ui/template.html:118-151` renders a unified view:

```html
<div class="diff">
  {{range .ViewDiff.Hunks}}
    <div class="hunk-header">{{.Header}}</div>
    {{range .Lines}}
      <div class="diff-line" data-kind="{{.Kind}}">
        <span class="ln old">{{.OldLine}}</span>
        <span class="ln new">{{.NewLine}}</span>
        <span class="diff-sign {{.Kind}}">+/-/ </span>
        <div class="code-text">{{.HTML or .Text}}</div>
      </div>
    {{end}}
  {{end}}
</div>
```

The grid layout uses 4 columns: `4ch 4ch 1ch max-content` (old line#, new line#, sign, content).

For side-by-side mode, the template will need a conditional branch (`{{if eq .DiffFormat "split"}}`) with a different HTML structure — likely two columns, each with its own line number, sign, and code content.

### CSS Styling

**Theme variables** (`internal/ui/styles.css:1-13`):

| Variable | Value | Description |
|----------|-------|-------------|
| `--bg` | `#0f0b0c` | Page background |
| `--panel` | `#161014` | Panel/code area background — **dark reddish-brown** |
| `--ink` | `#f3e9ea` | Primary text color |
| `--muted` | `#b7a7aa` | Secondary text color |
| `--accent` | `#c00000` | Accent color (red) |
| `--border` | `#2a1b1e` | Border color (dark red-brown) |
| `--line-hover` | `#241317` | Line hover background |
| `--line-selected` | `#3a1515` | Selected line background |

The `.diff` container uses `background: var(--panel)` (`styles.css:777`), which is `#161014`. This is the "red" background the user is referring to — it's a dark maroon/reddish-brown rather than pure black.

**Diff line coloring** (`styles.css:828-842`):
- Added lines: `background: rgba(46, 160, 67, 0.12)` with green sign `#2ea043`
- Deleted lines: `background: rgba(248, 81, 73, 0.16)` with red sign `#f85149`
- Context lines: no background color

### Preference Persistence

**Current state**: Only sidebar width is persisted via `localStorage.getItem/setItem("meatcheck-sidebar-width")` in the JS hook (`template.html:223-228`).

**All other state** (render toggles, viewed files, comments) lives in the server-side `ReviewModel` struct and is lost when the session ends.

**Pattern for adding diff format persistence**: Following the existing sidebar-width pattern:
1. Store in `localStorage` with key like `"meatcheck-diff-format"`
2. On mount, read from localStorage and send initial value to server via `Live.send()`
3. Server stores the preference in `ReviewModel.DiffFormat` field
4. Toggle button sends event to server, server updates model, JS also persists to localStorage

### Event Handling Pattern

Event handlers are registered in `buildLiveHandler()` (`internal/app/app.go:190`). Existing toggle patterns:

- `toggle-file-render`: flips `model.RenderFile` boolean, calls `updateView(model)`
- `toggle-comment-render`: flips `model.RenderComments` boolean
- `toggle-sidebar`: flips `model.SidebarCollapsed` boolean

A `toggle-diff-format` event would follow the same pattern: flip a `DiffFormat` field on `ReviewModel` and call `updateView(model)`.

### Toggle UI Pattern

Existing toggle buttons in the column header (`template.html:65-80`) use this pattern:

```html
<button class="btn btn-icon{{if .RenderFile}} active{{end}}" live-click="toggle-file-render" title="Toggle render">
  <span class="icon">icon</span>
</button>
```

A diff format toggle would follow this same pattern, placed alongside the existing toggles.

## Code References

- `internal/app/diff.go` — Unified diff parser (`parseUnifiedDiff()`, `DiffLine`, `DiffFile`)
- `internal/app/model.go:66-71` — `ViewMode` enum (`ModeFile`, `ModeDiff`)
- `internal/app/model.go:73-92` — `ViewDiffLine`, `ViewDiffHunk`, `ViewDiffFile` structs
- `internal/app/model.go:105-132` — `ReviewModel` struct (all session state)
- `internal/app/view.go:11-18` — `updateView()` dispatch by mode
- `internal/app/view.go:119-157` — `buildViewDiff()` — builds view model for unified diff
- `internal/app/app.go:190-232` — `buildLiveHandler()` — event handler registration, render function
- `internal/ui/template.html:118-151` — Unified diff HTML template
- `internal/ui/template.html:216-324` — JavaScript hooks (localStorage, click handlers)
- `internal/ui/styles.css:1-13` — CSS theme variables (including `--panel: #161014`)
- `internal/ui/styles.css:776-867` — Diff-specific CSS (`.diff`, `.diff-line`, line kind colors)

## Architecture Documentation

The application follows a server-driven reactive pattern using `jfyne/live`:

1. **Server-side state**: All UI state lives in `ReviewModel` struct, shared across WebSocket connections
2. **Event-driven updates**: Client sends events via WebSocket → Go handler mutates model → full template re-render → diff patched to client
3. **Client-side persistence**: Only sidebar width uses `localStorage`; everything else is ephemeral per session
4. **Template pattern**: Single `template.html` with conditional blocks for diff vs file mode
5. **CSS pattern**: CSS variables for theming, embedded in binary via `//go:embed`

## Design Decisions

1. **Background color**: Change only the `.diff` container background to black (`#000000`). Leave sidebar, hunk headers, and other panels using `--panel` (`#161014`) unchanged.
2. **Comment anchoring**: Comments should be attachable to **both old and new lines** — in both unified and side-by-side modes. This is a change from the current behavior where comments can only anchor to new-file lines. The `selectable` guard in `buildViewDiff()` (`view.go:142`) and the template's line-click handler need to be updated to allow selection of deleted/old lines.
3. **Toggle visibility**: The unified/side-by-side toggle button should only be visible in diff mode. Hide it when viewing plain files.

## Open Questions

None — all resolved.
