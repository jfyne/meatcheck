# Meatcheck

![Meatcheck logo](internal/ui/logo.png)

A local PR‑style review UI for LLM workflows. Run the `meatcheck` CLI with a set of files, review them in a browser, leave inline comments, and get structured feedback on stdout when you finish.

## Features

- File tree + code view UI similar to GitHub PR reviews
- Click to select a line, shift‑click for a range
- Inline comment threads under the referenced line
- Markdown rendering for comments (toggle raw/rendered)
- Syntax highlighting for code (toggle raw/rendered)
- Outputs TOON format to stdout on Finish

## Install / Build

```bash
go install github.com/jfyne/meatcheck@latest
```

## Usage

```bash
# open the meatcheck UI for specific files
./meatcheck path/to/file1.go path/to/file2.css

# or via go run during development
go run . -- path/to/file1.go path/to/file2.css

# include a review prompt/question
./meatcheck --prompt "Focus on security and error handling" path/to/file1.go

# render a unified diff
./meatcheck --diff changes.diff
cat changes.diff | ./meatcheck

# render only a section of a file
./meatcheck --range "path/to/file.go:10-40" path/to/file.go
```

## Keyboard Shortcuts

- `Ctrl+Enter` / `Cmd+Enter`: submit the inline comment

## Output

On “Finish Review”, the app prints TOON to stdout and exits.

Example (shape only):

```
comments[2]{end_line,path,start_line,text}:
  29,README.md,29,This is a comment
  40,README.md,40,This is another Example comment
```

## Notes

- Tabs/spaces are preserved in rendered mode via whitespace normalization to keep diffs accurate.
- Browser close behavior depends on browser security policies. If the tab can’t be closed programmatically, it will navigate to `about:blank` instead.

## Development

```bash
go test ./...
```
