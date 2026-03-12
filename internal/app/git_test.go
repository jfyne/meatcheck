package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestDetectGitContextInGitRepo verifies that detectGitContext populates
// WorkDir, Branch, and RepoRoot when run from within a git repository.
// WorkDir must match os.Getwd().
//
// Scenario: Branch and directory shown in title for git repo
// Scenario: Context bar shows branch and directory
func TestDetectGitContextInGitRepo(t *testing.T) {
	// This test runs from within the meatcheck project, which is a git repo.
	ctx := detectGitContext()
	if ctx == nil {
		t.Fatal("detectGitContext returned nil in a git repository; want non-nil *GitContext")
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd failed: %v", err)
	}

	if ctx.WorkDir == "" {
		t.Error("GitContext.WorkDir is empty; want non-empty string")
	}
	if ctx.WorkDir != wd {
		t.Errorf("GitContext.WorkDir = %q; want %q (os.Getwd result)", ctx.WorkDir, wd)
	}

	if ctx.Branch == "" {
		t.Error("GitContext.Branch is empty; want non-empty string")
	}

	if ctx.RepoRoot == "" {
		t.Error("GitContext.RepoRoot is empty; want non-empty string")
	}
}

// TestDetectGitContextNonGitDir verifies that detectGitContext returns nil
// when called from a directory that is not a git repository.
//
// Scenario: No context bar for non-git directory
// Scenario: Graceful fallback when git is not installed
func TestDetectGitContextNonGitDir(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	ctx := detectGitContext()
	if ctx != nil {
		t.Errorf("detectGitContext returned non-nil (%+v) from a non-git directory; want nil", ctx)
	}
}

// TestDetectGitContextWorktreeDetection verifies that detectGitContext sets
// IsWorktree to true and populates MainWorktree when run from inside a git
// worktree (not the main worktree).
//
// Scenario: Worktree info shown when in a git worktree
func TestDetectGitContextWorktreeDetection(t *testing.T) {
	// Skip if git is unavailable.
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	// Create a temporary bare-ish repo: git init, initial empty commit.
	mainRepo := t.TempDir()
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		// Suppress git output in tests.
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v in %q failed: %v\n%s", args, dir, err, out)
		}
	}

	run(mainRepo, "init")
	run(mainRepo, "commit", "--allow-empty", "-m", "initial")

	// Add a linked worktree.
	wtDir := filepath.Join(t.TempDir(), "linked-worktree")
	run(mainRepo, "worktree", "add", wtDir)

	// Change into the linked worktree and detect context.
	t.Chdir(wtDir)

	ctx := detectGitContext()
	if ctx == nil {
		t.Fatal("detectGitContext returned nil in a git worktree; want non-nil *GitContext")
	}

	if !ctx.IsWorktree {
		t.Error("GitContext.IsWorktree is false; want true when running inside a linked worktree")
	}

	if ctx.MainWorktree == "" {
		t.Error("GitContext.MainWorktree is empty; want path to main worktree")
	}

	// MainWorktree should point to the main repo directory (or contain it).
	absMain, err := filepath.Abs(mainRepo)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) failed: %v", mainRepo, err)
	}
	absMain = filepath.Clean(absMain)
	gotMain := filepath.Clean(ctx.MainWorktree)
	if gotMain != absMain {
		t.Errorf("GitContext.MainWorktree = %q; want %q", gotMain, absMain)
	}
}

// TestDetectGitContextDetachedHead verifies that detectGitContext sets Branch
// to "HEAD" when the repository is in detached HEAD state.
//
// Scenario: Detached HEAD state
func TestDetectGitContextDetachedHead(t *testing.T) {
	// Skip if git is unavailable.
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	repo := t.TempDir()
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v in %q failed: %v\n%s", args, dir, err, out)
		}
	}

	run(repo, "init")
	run(repo, "commit", "--allow-empty", "-m", "initial")
	run(repo, "checkout", "--detach", "HEAD")

	t.Chdir(repo)

	ctx := detectGitContext()
	if ctx == nil {
		t.Fatal("detectGitContext returned nil in a git repo with detached HEAD; want non-nil *GitContext")
	}

	if ctx.Branch != "HEAD" {
		t.Errorf("GitContext.Branch = %q; want %q in detached HEAD state", ctx.Branch, "HEAD")
	}
}
