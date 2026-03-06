---
date: 2026-03-06T23:30:00Z
researcher: josh
topic: "Grouped review mode with viewed/commented indicators"
tags: [research, codebase, review-mode, tree-pane, viewed-state, grouping]
last_updated: 2026-03-06
last_updated_by: josh
---

# Research: Grouped Review Mode with Viewed/Commented Indicators

## Research Question

How should meatcheck be extended to support:
1. Grouping files/diffs into named feature groups
2. "Viewed" tracking per file with tree pane indicators
3. "Commented" indicators in the tree pane
4. A "Mark as viewed" button below each file that advances to the next unviewed file
5. All of the above working in both grouped and ungrouped modes

## Summary

Meatcheck is a Go CLI that opens a browser-based PR-style review UI using the `jfyne/live` library for server-rendered live views over WebSocket. The codebase is compact (~18 source files) with a clear separation: `main.go` handles CLI flags and config, `internal/app/` contains the core logic (model, view, tree, diff parsing, IO, assets), `internal/highlight/` handles syntax highlighting, and `internal/ui/` embeds static assets (HTML template, CSS, images).

The current architecture has two modes (`ModeFile` and `ModeDiff`) controlled by a `ViewMode` enum. Adding a grouped mode and viewed/commented tracking requires changes across several layers: CLI flags, the data model, tree building, the HTML template, CSS, and event handlers.

## Detailed Findings

### 1. CLI & Config Layer

The CLI is defined in `main.go:24-77`. Flags are parsed with Go's `flag` package. The `Config` struct (`model.go:127-135`) passes parsed values into `app.Run()`.

A new `--groups` flag will accept a path to a JSON file containing an ordered array of `{name, files}` objects. This is simpler than a repeatable flag for complex groupings and preserves ordering.

Current config fields:
- `Host`, `Port`, `Paths`, `Prompt`, `Diff`, `Ranges`, `StdDiff`

### 2. Data Model (`model.go`)

The `ReviewModel` struct (`model.go:95-119`) is the central state object, assigned to each WebSocket session via `live.Socket.Assigns()`. Key fields:

- `Files []File` — loaded files in file mode
- `DiffFiles []DiffFile` — parsed diff files in diff mode
- `Tree []TreeItem` — flat list of tree items for sidebar rendering
- `SelectedPath string` — currently selected file
- `Comments []Comment` — all comments across files
- `Mode ViewMode` — either `ModeFile` or `ModeDiff`

The `TreeItem` struct (`model.go:22-28`) currently has: `Name`, `Path`, `Depth`, `IsDir`, `Selected`. It has no concept of grouping, viewed state, or comment indicators.

The `Comment` struct (`model.go:8-14`) tracks: `ID`, `Path`, `StartLine`, `EndLine`, `Text`.

### 3. Tree Building (`tree.go`)

The `buildTree()` function (`tree.go:8-63`) takes a `[]File` and `selectedPath`, builds a directory-based tree hierarchy using internal `treeNode` structs, then walks it depth-first to produce a flat `[]TreeItem` list.

The tree is rebuilt on every file selection event (`app.go:208`, `app.go:218`). It sorts directories before files, and directories before files alphabetically.

For grouped mode, the tree building needs to produce groups as top-level headings instead of (or in addition to) the directory hierarchy. Each group would contain its files, potentially still organized by directory within the group.

### 4. View Layer (`view.go`)

`updateView()` (`view.go:11-18`) dispatches to `updateFileView()` or `updateDiffView()` based on mode. These functions build the `ViewFile` or `ViewDiffFile` structs that the template renders.

Comment projection happens via `projectLineComments()` (`view.go:159-178`) and `projectBlockComments()` (`view.go:180-199`), which filter comments by path and line range.

### 5. HTML Template (`internal/ui/template.html`)

The template is a single-page live view using `jfyne/live`. Key sections:

- **Header** (lines 52-81): avatar, prompt, toggle buttons, Finish button
- **Sidebar/Tree** (lines 84-102): iterates `{{range .Tree}}` to render `tree-item` divs
- **Main content** (lines 104-201): conditional rendering for diff mode vs file mode (code view, markdown view)
- **JavaScript hooks** (lines 207-326): line selection, sidebar resize, comment actions, keyboard shortcuts

The tree rendering at lines 88-96 is straightforward:
```html
{{range .Tree}}
  {{if .IsDir}}
    <div class="tree-item dir" ...>{{.Name}}</div>
  {{else}}
    <div class="tree-item file {{if .Selected}}selected{{end}}" ...>{{.Name}}</div>
  {{end}}
{{end}}
```

Adding viewed/commented indicators requires extending `TreeItem` and adding conditional classes/icons here.

### 6. CSS (`internal/ui/styles.css`)

The tree item styling is at lines 67-94. The `.tree-item.selected` class uses `background: var(--accent-soft)` and `color: var(--accent)`. New states (viewed, has-comments) would follow the same pattern with new CSS classes and possibly small icon/indicator elements.

### 7. Event Handlers (`app.go`)

`buildLiveHandler()` (`app.go:146-377`) registers all live event handlers:

- `select-file` — changes selected path, rebuilds tree
- `select-line` — sets selection range
- `add-comment` — adds a comment to the model
- `delete-comment`, `edit-comment` — modify comments
- `toggle-sidebar`, `toggle-file-render`, `toggle-comment-render` — UI toggles
- `finish` — closes the UI, triggers stdout output

New events needed:
- `mark-viewed` — toggles viewed state for current file, advances to next unviewed
- Potentially `select-group` for group navigation

### 8. Output (`io_helpers.go`)

`emitToon()` (`io_helpers.go:88-98`) encodes comments to TOON format via `gotoon.Encode()`. The output structure is `{"comments": [...]}`. The viewed state does not need to be in the output — it's purely a UI tracking feature.

### 9. Live Framework (`jfyne/live`)

The app uses `jfyne/live` v0.16.3, which provides:
- Server-rendered HTML over WebSocket
- `live-click`, `live-submit` attributes for event binding
- `live-value-*` attributes for passing data with events
- `live.Handler` with `MountHandler`, `RenderHandler`, `HandleEvent()`
- Full page re-render on each event (no partial updates)

Since the entire page re-renders on each event, state changes (viewed, group selection) automatically reflect in the UI without needing client-side state management.

## Code References

- `main.go:13-22` — `listFlag` type for repeatable flags (pattern for `--group`)
- `main.go:24-77` — CLI flag definitions and `Config` construction
- `model.go:8-14` — `Comment` struct
- `model.go:16-28` — `File` and `TreeItem` structs
- `model.go:95-119` — `ReviewModel` struct (central state)
- `model.go:127-135` — `Config` struct
- `tree.go:8-63` — `buildTree()` function
- `view.go:11-18` — `updateView()` dispatcher
- `view.go:159-178` — `projectLineComments()` for comment projection
- `app.go:42-144` — `Run()` function (initialization flow)
- `app.go:146-377` — `buildLiveHandler()` (all event handlers)
- `app.go:194-222` — `select-file` event handler
- `app.go:280-305` — `add-comment` event handler
- `io_helpers.go:88-98` — `emitToon()` output function
- `internal/ui/template.html:88-96` — Tree rendering in template
- `internal/ui/styles.css:67-94` — Tree item CSS

## Architecture Documentation

### Pattern: Mode-based dispatch

The app uses a `ViewMode` enum (`ModeFile`/`ModeDiff`) to conditionally branch in multiple places: `Run()`, `updateView()`, `select-file` handler, and the template. A new `ModeGrouped` (or similar) would follow this pattern, or grouping could be an orthogonal feature that works with both existing modes.

### Pattern: Full re-render on event

Every event handler returns the updated model, and `jfyne/live` re-renders the full template. There is no client-side diffing — the server sends the complete HTML. This means all state changes are automatically reflected without explicit DOM manipulation.

### Pattern: Flat tree items

The tree is stored as a flat `[]TreeItem` list with a `Depth` field for indentation. This makes it easy to prepend group headers as items with `Depth: 0` and a new `IsGroup` flag.

### Pattern: Embedded assets

All UI assets (template, CSS, images) are embedded via `embed.FS` in `internal/ui/ui.go`. The CSS is built at runtime by combining the embedded `styles.css` with Chroma theme CSS.

## Decisions

1. **Group input format**: Ordered JSON array via `--groups groups.json`. The JSON format is:
   ```json
   [
     {"name": "Auth", "files": ["auth.go", "middleware.go"]},
     {"name": "API", "files": ["handler.go", "routes.go"]}
   ]
   ```
   Groups are displayed in the order specified in the array.

2. **Mode support**: Grouping works with both `--diff` and plain file arguments. When used with `--diff`, the group file paths reference diff file paths.

3. **Ungrouped files**: Files not assigned to any group appear in an auto-created "Other" group at the bottom of the tree.

4. **Navigation on mark-viewed**: When the last unviewed file in a group is marked viewed, navigation advances to the first unviewed file in the next group.

5. **Viewed state persistence**: The viewed state lives in `ReviewModel` on the server, so it persists across re-renders and browser refreshes within the same session.

## Open Questions

None — all questions resolved.
