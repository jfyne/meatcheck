---
name: meatcheck
description: Request a PR-style review UI for a set of files or a diff and collect inline feedback with file/line anchors from the user.
---

# Meatcheck

Use this skill to request a human-style review of a set of files via meatcheck.

## How to invoke

```bash
meatcheck <file1> <file2> ...
meatcheck --prompt "Focus on security and error handling" <file1>
meatcheck --diff changes.diff
cat changes.diff | meatcheck
meatcheck --range "path/to/file.go:10-40" <file1>
```

The CLI opens a browser UI with a GitHub-like review layout. The reviewer can select lines/ranges, add inline comments, and click **Finish**.

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
- Use `--diff` or pipe a unified diff to render changes.
- Use `--range` to render only specific sections of a file.
- Use `--skill` to print this SKILL.md content.
