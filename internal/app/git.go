package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// gitRun executes a git command in the current working directory and returns
// the trimmed output. Returns an error if the command fails.
func gitRun(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// detectGitContext detects whether the current working directory is inside a
// git repository and returns a populated *GitContext if so. Returns nil when:
//   - os.Getwd() fails
//   - git is not installed or not found in PATH
//   - the current directory is not inside a git repository
func detectGitContext() *GitContext {
	wd, err := os.Getwd()
	if err != nil {
		return nil
	}

	// Check if we are inside a git repository. This command also returns the
	// repository root, which we use to populate RepoRoot.
	repoRoot, err := gitRun("rev-parse", "--show-toplevel")
	if err != nil {
		// Not a git repo, or git not available.
		return nil
	}

	// Get the current branch name. In detached HEAD state this returns the
	// literal string "HEAD", which we pass through as-is.
	branch, err := gitRun("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		// Unexpected failure; degrade gracefully.
		branch = ""
	}

	ctx := &GitContext{
		WorkDir:  wd,
		Branch:   branch,
		RepoRoot: repoRoot,
	}

	// Detect whether we are inside a linked worktree by comparing git-dir
	// with git-common-dir. In the main worktree they are the same (both
	// .git); in a linked worktree git-dir points to a per-worktree directory
	// inside .git/worktrees/<name> while git-common-dir still points to the
	// main .git directory.
	gitDir, err := gitRun("rev-parse", "--git-dir")
	if err != nil {
		return ctx
	}
	gitCommonDir, err := gitRun("rev-parse", "--git-common-dir")
	if err != nil {
		return ctx
	}

	// Normalise both paths to absolute paths so the comparison is reliable
	// even when git returns relative paths (which it can for git-common-dir
	// in some configurations).
	absGitDir, err := filepath.Abs(gitDir)
	if err != nil {
		return ctx
	}
	absGitCommonDir, err := filepath.Abs(gitCommonDir)
	if err != nil {
		return ctx
	}

	if absGitDir != absGitCommonDir {
		// We are in a linked worktree. Derive the main worktree path by
		// stripping the trailing ".git" component from git-common-dir.
		// The main worktree is the directory that contains the .git folder.
		ctx.IsWorktree = true
		ctx.MainWorktree = filepath.Dir(absGitCommonDir)
	}

	return ctx
}
