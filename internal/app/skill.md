---
name: meatcheck
description: Request a PR-style review UI for a set of files or a diff and collect inline feedback with file/line anchors from the user.
---

# Meatcheck

Use this skill to request a human-style review of a set of files via meatcheck.
When you want targeted feedback, provide `--prompt` with specific review goals (for example: security, correctness, performance, or API design) so the reviewer knows what to prioritize.

## How to invoke

```bash
meatcheck <file1> <file2> ...
meatcheck --prompt "Focus on security and error handling" <file1>
meatcheck --diff changes.diff
cat changes.diff | meatcheck
meatcheck --range "path/to/file.go:10-40" <file1>
```

The CLI opens a browser UI with a GitHub-like review layout. The reviewer can select lines/ranges, add inline comments, and click **Finish**.

## Reviewing diffs

Use diff mode when you want feedback scoped to changed lines instead of full files.

```bash
# review from a saved patch file
meatcheck --diff changes.diff

# review from git output directly
git diff -- . ':!go.sum' | meatcheck

# review staged changes
git diff --cached | meatcheck --prompt "Focus on regressions and missing tests"
```

In diff mode:
- Comments anchor to new-file line numbers for added/context lines.
- Deleted lines are shown for context but are not comment targets.
- File and hunk headers come from the unified diff, so prefer standard `git diff` output.

## Important

- Run the `meatcheck` command and wait for the process to finish.
- Do not stop or terminate the process manually; keep it running until it exits on its own.

## Output

On finish, the CLI prints TOON to stdout with a list of comments:

```
comments[2]{end_line,path,start_line,text}:
  29,README.md,29,This is a comment
  40,README.md,40,This is another Example comment
```

## Notes

- Use `--host` / `--port` to control binding.
- Use `--prompt` to tell the reviewer what to focus on.
- Use `--diff` or pipe a unified diff to render changes.
- Use `--range` to render only specific sections of a file.
- Use `--skill` to print this SKILL.md content.
