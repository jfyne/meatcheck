# Implementation Plan: Git Context Display

Add working directory, git branch, and worktree info to the meatcheck UI. Display context in both the HTML `<title>` and a header bar on the page.

## Context

**Key Files**:
- `internal/app/model.go` - Data types: `ReviewModel`, `Config`
- `internal/app/app.go` - `Run()` function, `buildLiveHandler()`, render pipeline
- `internal/ui/template.html` - HTML template with header, title, live framework integration
- `internal/ui/styles.css` - CSS with theme variables, header styling
- `internal/app/preferences.go` - Pattern for self-contained utility with graceful error handling

**Architectural Notes**:
- Server-driven reactive UI via `jfyne/live` — events go client->server, server mutates model, full re-render diffs to client
- All UI state lives in `ReviewModel` struct; template accesses it via `{{.Assigns}}`
- Template uses Go `html/template` with `{{with}}` and `{{if}}` for conditional rendering
- CSS uses custom properties (`:root` variables), embedded in binary via `//go:embed`
- No `os/exec` usage in the project currently; this introduces the first subprocess calls (for git)
- `Run()` already calls `os.Getwd()` at line 138 for the file handler root
- The `<title>` tag (line 42) is **outside** the `{{with .Assigns}}` block (line 50); access model fields via `{{.Assigns.FieldName}}`. Inside the block, use `{{$root.FieldName}}`.

**Functional Requirements** (EARS notation):
- When the application starts in a git repository, the system shall detect and display the current branch name
- When the application starts in a git repository, the system shall display the working directory path in a context bar
- When the application is running inside a git worktree, the system shall display worktree info (the worktree path and main repo path)
- When the application starts outside a git repository, the system shall not display a context bar
- The HTML `<title>` shall include branch name and directory when in a git repo, and be just "Meatcheck" otherwise
- If git is not installed or detection fails, the system shall degrade gracefully (no context bar, plain title)

## Acceptance Criteria

~~~gherkin
Feature: Git context display in meatcheck UI

  Scenario: Branch and directory shown in title for git repo
    Given the user launches meatcheck from a git repository on branch "feature-xyz"
    When the page loads
    Then the HTML title contains "Meatcheck - feature-xyz"
    And the HTML title contains the working directory path

  Scenario: Context bar shows branch and directory
    Given the user launches meatcheck from a git repository on branch "main"
    And the working directory is "/home/user/project"
    When the page loads
    Then the header contains a context bar with the branch "main"
    And the context bar shows the directory "/home/user/project"

  Scenario: Worktree info shown when in a git worktree
    Given the user launches meatcheck from a git worktree
    When the page loads
    Then the context bar shows a "worktree" indicator

  Scenario: No context bar for non-git directory
    Given the user launches meatcheck from a directory that is not a git repository
    When the page loads
    Then the HTML title is "Meatcheck"
    And no context bar is displayed in the header

  Scenario: Graceful fallback when git is not installed
    Given git is not available on the system PATH
    When the user launches meatcheck
    Then the application starts normally
    And no git context is displayed

  Scenario: Title fallback when GitContext is nil
    Given the ReviewModel has no GitContext set
    When the page renders
    Then the HTML title is exactly "Meatcheck"

  Scenario: Detached HEAD state
    Given the user launches meatcheck from a git repository in detached HEAD state
    When the page loads
    Then the context bar shows branch "HEAD"
~~~

**Source**: Generated from plan context

## Task List

### Git Context Detection

- [x] Add `GitContext` struct and field to `ReviewModel` (`internal/app/model.go`) [Stage 1]
  - Files: `internal/app/model.go` (modifies)
  - Add a `GitContext` struct with fields: `WorkDir string`, `Branch string`, `RepoRoot string`, `IsWorktree bool`, `MainWorktree string`
  - Add a `Git *GitContext` field to `ReviewModel` (pointer so `{{with}}` guards work — nil means no git context)

- [x] Create `detectGitContext()` in new file (`internal/app/git.go`) [Stage 1]
  - Files: `internal/app/git.go` (creates)
  - Function `detectGitContext() *GitContext` that:
    - Gets cwd via `os.Getwd()` -> `WorkDir`
    - Runs `git rev-parse --abbrev-ref HEAD` -> `Branch` (returns literal "HEAD" in detached state)
    - Runs `git rev-parse --show-toplevel` -> `RepoRoot`
    - Runs `git rev-parse --git-dir` and `git rev-parse --git-common-dir`, compares with `filepath.Abs()` to detect worktree
    - If worktree detected: `IsWorktree = true`, `MainWorktree` = directory containing git-common-dir (strip trailing `.git`)
  - **Nil policy**: Returns `nil` when not in a git repo (i.e., when `git rev-parse --show-toplevel` fails). This ensures `{{with .Git}}` template guards correctly hide the context bar for non-git directories.
  - Graceful error handling: if git is not installed or any command fails, return nil
  - Follow `preferences.go` pattern for self-contained utility with graceful degradation

- [x] Write tests for `detectGitContext()` (`internal/app/git_test.go`) [Stage 1]
  - Files: `internal/app/git_test.go` (creates)
  - Test that WorkDir, Branch, and RepoRoot are populated when run from a git repo
  - Test that `nil` is returned from a non-git temp directory
  - Test worktree detection by creating a real worktree in a temp git repo

- [x] Wire `detectGitContext()` into `app.Run()` (`internal/app/app.go`) [Stage 1]
  - Files: `internal/app/app.go` (modifies)
  - Call `detectGitContext()` early in `Run()`
  - Set `Git: gitCtx` on the `ReviewModel`
  - Replace the existing `os.Getwd()` call (line 138) with `gitCtx.WorkDir` when non-nil (fallback to `os.Getwd()` if nil)

### UI Display

- [x] Update title, add context bar, and add CSS (`internal/ui/template.html`, `internal/ui/styles.css`) [Stage 2, depends: Stage 1]
  - Files: `internal/ui/template.html` (modifies), `internal/ui/styles.css` (modifies)
  - **Title**: Replace `<title>Meatcheck</title>` (line 42, outside `{{with .Assigns}}`) with:
    `<title>Meatcheck{{with .Assigns.Git}}{{if .Branch}} - {{.Branch}}{{end}}{{if .WorkDir}} {{.WorkDir}}{{end}}{{end}}</title>`
    When `Git` is nil, `{{with}}` is skipped and title is just "Meatcheck".
  - **Context bar**: Insert `<div class="header-context">` between `.header-top` closing (line 88) and `</header>` (line 89), inside the `{{with .Assigns}}` scope. Guard with `{{with $root.Git}}` so it only renders when git context is available. Show branch, directory, and worktree items conditionally.
  - **CSS**: Add `.header-context` (flex, 12px, `var(--muted)`, border-top), `.ctx-item`, `.ctx-label` (uppercase, 10px), `.ctx-value` (monospace) after `.header-top` rules in styles.css.

- [x] Write render tests for title and context bar (`internal/app/http_render_test.go`) [Stage 2, depends: Stage 1]
  - Files: `internal/app/http_render_test.go` (modifies)
  - Test title contains branch when GitContext is set (Scenario 1)
  - Test title is just "Meatcheck" when GitContext is nil (Scenario 6)
  - Test header-context div rendered with branch info (Scenario 2)
  - Test header-context div absent when GitContext is nil (Scenario 4)
  - Test worktree info shown only when IsWorktree is true (Scenario 3)
  - Test CSS for `.header-context` is present in rendered output

## Implementation Notes

- **GitContext as pointer**: Must be `*GitContext` on ReviewModel so `{{with .Git}}` correctly skips when nil. A zero-value struct would be truthy in Go templates.
- **Nil policy**: `detectGitContext()` returns nil when not in a git repo. This cleanly gates both the title augmentation and the context bar via `{{with}}`.
- **Detached HEAD**: `git rev-parse --abbrev-ref HEAD` returns literal "HEAD" in detached state; display as-is.
- **Performance**: 4 git subprocess calls add ~50-100ms at startup; acceptable since it runs once.
- **`filepath.Abs` for worktree detection**: `git rev-parse --git-common-dir` can return relative paths; must normalize before comparing.
- **Existing `os.Getwd()` consolidation**: Replace the call at app.go:138 with `gitCtx.WorkDir` when available.
- **Template scoping**: `<title>` (line 42) is outside `{{with .Assigns}}`; must use `{{.Assigns.Git}}`. Context bar is inside the `{{with .Assigns}}` block; use `{{$root.Git}}`.

## Key Files

- `internal/app/model.go` - GitContext struct definition, ReviewModel field
- `internal/app/git.go` - Git detection logic (new file)
- `internal/app/git_test.go` - Tests for git detection (new file)
- `internal/app/app.go` - Wiring detectGitContext into Run()
- `internal/ui/template.html` - Title and header context bar
- `internal/ui/styles.css` - Context bar styling
- `internal/app/http_render_test.go` - Render tests for title and context bar

## Execution Stages

### Stage 1: Git Context Detection

#### Test Creation Phase (parallel)
- T-test-1A: Write tests for `GitContext` struct and `detectGitContext()` (hmm-test-writer)
  - Files: `internal/app/git_test.go` (creates)
  - New feature tests (RED): Scenarios 4, 5, 7 (nil return for non-git, graceful fallback, detached HEAD)
  - Test that WorkDir, Branch, and RepoRoot are populated in git repos
  - Test worktree detection when git-dir differs from git-common-dir

#### Implementation Phase (parallel, depends on Test Creation Phase)
- T-impl-1A: Add `GitContext` struct and field to `ReviewModel` (hmm-implement-worker, TDD mode)
  - Files: `internal/app/model.go` (modifies)
  - Make RED tests pass (GREEN)
- T-impl-1B: Create `detectGitContext()` function (hmm-implement-worker, TDD mode)
  - Files: `internal/app/git.go` (creates)
  - Make RED tests pass (GREEN)
- T-impl-1C: Wire `detectGitContext()` into `app.Run()` (hmm-implement-worker, TDD mode)
  - Files: `internal/app/app.go` (modifies)

### Stage 2: UI Display (depends on Stage 1)

#### Test Creation Phase
- T-test-2A: Write render tests for title and context bar (hmm-test-writer)
  - Files: `internal/app/http_render_test.go` (modifies)
  - New feature tests (RED): Scenarios 1, 2, 3, 4, 6

#### Implementation Phase (depends on Test Creation Phase)
- T-impl-2A: Update title, add context bar and CSS (hmm-implement-worker, TDD mode)
  - Files: `internal/ui/template.html` (modifies), `internal/ui/styles.css` (modifies)
  - Make RED tests pass (GREEN)

## Refs

- Existing plan pattern: `docs/plans/diff-view-options.md`
