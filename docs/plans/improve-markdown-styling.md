# Implementation Plan: Improve Markdown Styling

Align Meatcheck's `.markdown` CSS with GitHub's `.markdown-body` conventions for typography, spacing, and overflow prevention. Preserve Meatcheck's design language (angular corners, accent colors) while fixing functional gaps.

## Context

**Research Document**: `docs/research/markdown-styling.md`

**Key Files**:
- `internal/ui/styles.css:417-513` — All `.markdown` CSS rules
- `internal/ui/styles.css:234-248` — `.markdown-file-preview` and `.markdown-file-body` container styles
- `internal/app/assets.go:33-48` — `renderMarkdown()` and `renderMarkdownDocument()` functions

**Architectural Notes**:
- All CSS lives in a single file (`internal/ui/styles.css`) embedded at build time.
- Markdown is rendered by goldmark with GFM extension. The HTML it produces uses standard tags (`p`, `h1-h6`, `pre > code`, `img`, `hr`, `table`, `blockquote`, etc).
- The `.markdown` class is applied in three contexts: comment bodies, prompt display, and markdown file preview. Changes affect all three.
- Meatcheck uses angular design language (`border-radius: 0`) and red accent colors — these are intentional and should not change.

**Functional Requirements** (EARS notation):
- The `.markdown` container shall constrain images to not overflow their parent container.
- The `.markdown` container shall reset inline code styling inside code blocks (`pre > code`).
- The `.markdown` container shall provide consistent vertical rhythm using `16px` bottom margins on block-level elements.
- The `.markdown` container shall style all heading levels (h1–h6) with appropriate sizes and weights.
- The `.markdown` container shall prevent wide tables from breaking layout.
- The `.markdown` container shall style horizontal rules.
- The `.markdown` container shall use `word-wrap: break-word` to prevent long text from overflowing.
- When a markdown document begins with YAML frontmatter (`---` delimited block), the system shall render the frontmatter as a styled metadata table above the document content.

## Batch Size

| Metric | Count | Rating |
|--------|-------|--------|
| Tasks | 3 | Small |
| Files | 2 | Small |
| Stages | 1 | Small |

**Overall: Small**

## Execution Stages

### Stage 1

#### Test Creation Phase (parallel)
- T-test-1: Write render tests for markdown styling and frontmatter rendering (hmm-test-writer)
  - Files: `internal/app/http_render_test.go` (modifies), `internal/app/assets_test.go` (modifies or creates)
  - New feature tests (RED): image max-width rule present, pre>code has no background, hr element styled, h4-h6 rendered with font-size, last paragraph has no bottom margin, frontmatter rendered as table

#### Implementation Phase (parallel, depends on Test Creation Phase)
- T-impl-1: Update all markdown CSS rules (hmm-implement-worker, TDD mode)
  - Covers: Typography, Spacing, Overflow, Code Blocks, HR (all four CSS Task List items)
  - Files: `internal/ui/styles.css` (modifies)
- T-impl-2: Strip YAML frontmatter before rendering (hmm-implement-worker, TDD mode)
  - Covers: Frontmatter Stripping task
  - Files: `internal/app/assets.go` (modifies)

## Task List

All four task groups below are executed as a single implementation worker (T-impl-1) since they all modify the same file (`internal/ui/styles.css`). They are listed separately for logical clarity.

### Typography

- [x] Update heading styles, add h4–h6, update link decoration (`internal/ui/styles.css`) [Stage 1, T-impl-1]
  - Files: `internal/ui/styles.css` (modifies)
  - Update shared heading rule: `margin: 1.5em 0 1em 0` (was `1.3em 0 0.5em 0`) and add `font-weight: 600`
  - Add `h1`/`h2` bottom border: `padding-bottom: .3em; border-bottom: 1px solid var(--border)`
  - Convert h1–h3 sizes from `px` to `em` units: h1 `2em`, h2 `1.5em`, h3 `1.25em`
  - Add h4: `font-size: 1em`
  - Add h5: `font-size: .875em`
  - Add h6: `font-size: .85em; color: var(--muted)`
  - Add `.markdown a`: `text-decoration: none` and `.markdown a:hover`: `text-decoration: underline`

### Spacing and Overflow

- [x] Update block-level spacing to consistent 16px bottom margins (`internal/ui/styles.css`) [Stage 1, T-impl-1]
  - Files: `internal/ui/styles.css` (modifies)
  - Change `.markdown p` margin from `0 0 8px 0` to `0 0 16px 0`
  - Preserve existing `.markdown p:last-child { margin-bottom: 0 }` rule (prevents trailing space in containers)
  - Change `.markdown ul, .markdown ol` from `margin: 0 0 10px 22px` to `margin: 0 0 16px 0; padding-left: 2em`
  - Add `.markdown li + li { margin-top: .25em }`
  - Change `.markdown blockquote` margin-bottom from `10px` to `16px`, padding from `padding-left: 12px` to `padding: 0 1em`
  - Change `.markdown table` margin from `0 0 12px 0` to `0 0 16px 0`
  - Change `.markdown pre` padding from `10px 12px` to `16px`
  - Add `.markdown { word-wrap: break-word }` on the container

### Image and Table Overflow

- [x] Add image constraints and table overflow handling (`internal/ui/styles.css`) [Stage 1, T-impl-1]
  - Files: `internal/ui/styles.css` (modifies)
  - Add `.markdown img { max-width: 100%; height: auto; display: block }`
  - Add `.markdown table { display: block; max-width: 100%; overflow: auto }`
  - Add `.markdown th { font-weight: 600 }`
  - Add `.markdown tr:nth-child(2n) { background: rgba(255, 255, 255, 0.03) }` (subtle row striping)

### Code Blocks and HR

- [x] Fix pre>code reset, add HR styles (`internal/ui/styles.css`) [Stage 1, T-impl-1]
  - Files: `internal/ui/styles.css` (modifies)
  - Add `pre > code` reset: `.markdown pre code { background: transparent; padding: 0; border: 0 }`
  - Add `.markdown pre { font-size: 85%; line-height: 1.45 }`
  - Add `.markdown code { font-size: 85%; white-space: break-spaces }`
  - Add HR styles: `.markdown hr { height: .25em; padding: 0; margin: 24px 0; background-color: var(--border); border: 0 }`

### Frontmatter Rendering

- [x] Render YAML frontmatter as a styled metadata table (`internal/app/assets.go`, `internal/ui/styles.css`) [Stage 1, T-impl-2]
  - Files: `internal/app/assets.go` (modifies), `internal/ui/styles.css` (modifies)
  - Add a `renderFrontmatter(input string) (html string, rest string)` function that detects YAML frontmatter and converts it to an HTML metadata table
  - Frontmatter is defined as: content starts with `---\n`, followed by any content, closed by `\n---\n` (or `\n---` at EOF)
  - Parse the YAML key-value pairs (use `gopkg.in/yaml.v3` or simple line-by-line `key: value` splitting)
  - Render as a `<table class="frontmatter">` with key-value rows, prepended before the rest of the markdown output
  - Call at the start of `renderMarkdown()`: extract frontmatter, render the rest with goldmark, concatenate the frontmatter HTML + rendered body
  - If no frontmatter is detected, render as normal (no table prepended)
  - Add CSS styles for `.markdown .frontmatter`: subtle background, muted text, compact layout, border matching the existing design
  - Edge cases: only detect if `---` is the very first line (no leading whitespace or content before it)

## Acceptance Criteria

~~~gherkin
Feature: Markdown styling improvements

  Scenario: Images constrained to container width
    Given a markdown document contains an image wider than the viewport
    When the document is rendered in the markdown file preview
    Then the image is scaled down to fit within the container
    And the image maintains its aspect ratio

  Scenario: Code blocks do not show inline code background
    Given a markdown document contains a fenced code block
    When the document is rendered
    Then the code element inside the pre has no background color
    And the code element inside the pre has no extra padding

  Scenario: Horizontal rules are visible
    Given a markdown document contains a horizontal rule (---)
    When the document is rendered
    Then a visible bar with the border color background is displayed
    And it has vertical margin separating it from surrounding content

  Scenario: All heading levels are styled
    Given a markdown document uses headings h1 through h6
    When the document is rendered
    Then each heading level has a distinct font size
    And h1 and h2 have a bottom border
    And all headings have font-weight 600

  Scenario: Wide tables do not break layout
    Given a markdown document contains a table wider than the viewport
    When the document is rendered
    Then the table scrolls horizontally within its container
    And the surrounding layout is not affected

  Scenario: Block-level elements have consistent spacing
    Given a markdown document contains paragraphs, lists, and blockquotes
    When the document is rendered
    Then all block-level elements have 16px bottom margins
    And list items have spacing between siblings

  Scenario: YAML frontmatter rendered as metadata table
    Given a markdown document starts with YAML frontmatter
    """
    ---
    title: My Document
    date: 2024-01-01
    ---
    # Hello
    """
    When the document is rendered
    Then a table with class "frontmatter" appears before the document content
    And the table contains rows for "title" and "date" with their values
    And the content after the frontmatter ("# Hello") is rendered as normal markdown

  Scenario: Document without frontmatter renders unchanged
    Given a markdown document does not start with "---"
    When the document is rendered
    Then no frontmatter table appears
    And the full content is rendered as markdown

  Scenario: Last paragraph has no trailing margin
    Given a markdown container ends with a paragraph
    When the document is rendered
    Then the final paragraph has no bottom margin
    And no extra whitespace appears at the bottom of the container
~~~

**Source**: Generated from plan context

## Implementation Notes

- **Preserve design identity**: Keep `border-radius: 0` everywhere, keep `var(--accent)` for link color, keep blue-tinted inline code background. These are intentional Meatcheck design choices.
- **Row striping**: Use a very subtle `rgba(255, 255, 255, 0.03)` rather than GitHub's opaque color, to work with Meatcheck's darker panel background.
- **Comment body context**: The `.line-comment-body.markdown` has a small max-width by nature of the comment thread layout. The spacing changes will apply there too — verify that `16px` paragraph margins don't feel too spacious in that narrow context. The existing `.prompt-inline .markdown p { margin: 0 }` override already handles the prompt context.
- **Font-size 85% on code**: This applies to inline `code` elements. Inside `pre > code`, the `pre` already sets its own font-size, and the code reset strips inherited styles, so the 85% won't double-compound.
- **Preserve `p:last-child`**: The existing `.markdown p:last-child { margin-bottom: 0 }` rule must be preserved. It prevents trailing whitespace in comment bodies and other compact containers.

## Refs

- `docs/research/markdown-styling.md` — Detailed comparison with GitHub's markdown CSS
- [github-markdown-css](https://github.com/sindresorhus/github-markdown-css) — Reference implementation
