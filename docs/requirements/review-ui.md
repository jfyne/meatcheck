# Requirements: Review UI

## Functional Requirements

- When the application starts in a git repository, the system shall detect and display the current branch name
- When the application starts in a git repository, the system shall display the working directory path in a context bar
- When the application is running inside a git worktree, the system shall display worktree info (the worktree path and main repo path)
- When the application starts outside a git repository, the system shall not display a context bar
- The HTML `<title>` shall include branch name and directory when in a git repo, and be just "Meatcheck" otherwise
- If git is not installed or detection fails, the system shall degrade gracefully (no context bar, plain title)
- When a user clicks on a list item in rendered markdown view, the system shall select only that item's line range (not the entire list)
- When a user submits a comment while a single list item is selected, the system shall anchor the comment to that item's StartLine and EndLine
- When a rendered markdown file contains a list, the system shall display each list item as an independently clickable and commentable block
- When a list item has a comment, the system shall display the comment thread directly below that item (not after the entire list)
- While rendering per-item blocks, the system shall wrap consecutive list-item blocks in the correct `<ul>` or `<ol>` HTML tag for proper list styling
- If an ordered list starts at a number other than 1, the system shall include the `start` attribute on the `<ol>` tag
- When rendering GFM task list items as individual blocks, the system shall correctly display checkboxes

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

~~~gherkin
Feature: Per-list-item commenting in markdown review

  Scenario: Click selects individual list item
    Given a markdown file with a 3-item unordered list on lines 5-7
    When the user clicks on the second list item
    Then only line 6 is selected
    And the first and third items are not selected

  Scenario: Comment anchors to single list item
    Given a markdown file with a 3-item list
    And the user has selected the second item (line 6)
    When the user submits a comment "Fix this item"
    Then the comment is created with StartLine=6 and EndLine=6
    And the comment thread appears below the second item only

  Scenario: Ordered list preserves numbering
    Given a markdown file with an ordered list starting at 3
    When the file is rendered in markdown preview mode
    Then the list items are numbered starting from 3
    And each item is independently clickable

  Scenario: GFM task list checkboxes render correctly
    Given a markdown file containing "- [ ] todo" and "- [x] done"
    When the file is rendered in markdown preview mode
    Then each task item shows its checkbox (unchecked or checked)
    And each task item is independently commentable

  Scenario: Nested lists stay within parent item
    Given a markdown file with a list item containing a nested sub-list
    When the file is rendered in markdown preview mode
    Then the parent item's block includes the nested list HTML
    And the nested items are not split into separate commentable blocks

  Scenario: Non-list blocks are unaffected
    Given a markdown file with headings, paragraphs, and blockquotes
    When the file is rendered in markdown preview mode
    Then these blocks render exactly as before
    And they carry no list wrapper metadata

  Scenario: Shift-click selects range across list items
    Given a markdown file with a 3-item list
    And the user has clicked item 1 (line 5)
    When the user shift-clicks item 3 (line 7)
    Then lines 5-7 are selected (spanning all three items)

  Scenario: Legacy comment on full list range displays correctly
    Given a comment anchored to lines 5-7 (the full list range)
    And the list is now rendered as 3 per-item blocks
    Then all three items show the "commented" visual indicator
    And the comment text displays below the first item
~~~

## Key Files

- `internal/app/model.go` - GitContext struct definition, ReviewModel field, MarkdownBlock struct
- `internal/app/git.go` - Git detection logic
- `internal/app/app.go` - Wiring detectGitContext into Run(), event handlers
- `internal/app/assets.go` - renderMarkdownBlocks(), nodeByteRange()
- `internal/app/view.go` - projectBlockComments(), updateFileView()
- `internal/ui/template.html` - Title, header context bar, markdown block rendering
- `internal/ui/styles.css` - Context bar styling, md-block styling
- `internal/app/markdown_test.go` - Markdown block rendering tests
- `internal/app/http_render_test.go` - Render tests for title and context bar
