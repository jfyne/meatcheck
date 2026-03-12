# Requirements: Review UI

## Functional Requirements

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

## Key Files

- `internal/app/model.go` - GitContext struct definition, ReviewModel field
- `internal/app/git.go` - Git detection logic
- `internal/app/app.go` - Wiring detectGitContext into Run()
- `internal/ui/template.html` - Title and header context bar
- `internal/ui/styles.css` - Context bar styling
- `internal/app/http_render_test.go` - Render tests for title and context bar
