# Research: Markdown Styling Comparison with GitHub

## Question

How does the Meatcheck `.markdown` CSS compare to GitHub's `.markdown-body` styling? Focus on typography, spacing, and image overflow prevention.

## Context

Markdown rendering appears in three places in the UI:
- **Comment bodies** — `.line-comment-body.markdown` (inline review comments)
- **Prompt display** — `.prompt-body.markdown` (review prompt in header)
- **Markdown file preview** — `.markdown-file-preview.markdown` > `.markdown-file-body` (full document rendering of `.md` files)

The Go backend uses `github.com/yuin/goldmark` with the GFM extension (`internal/app/assets.go:25-31`). Images in markdown documents are rewritten from relative paths to `/file?path=<encoded>` for local serving (`internal/app/assets.go:50-84`).

## Findings

### Typography

| Property | Meatcheck | GitHub Dark |
|----------|-----------|-------------|
| Body font | `-apple-system, BlinkMacSystemFont, avenir next, avenir, segoe ui, helvetica neue, Adwaita Sans, Cantarell, Ubuntu, roboto, noto, helvetica, arial, sans-serif` | `-apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans", Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji"` |
| Body font size | Inherited (no explicit size on `.markdown`) | `16px` |
| Body line-height | `1.5` | `1.5` |
| Body color | `var(--ink)` = `#f3e9ea` (warm white) | `#f0f6fc` (cool white) |
| Mono font | `Menlo, Consolas, Monaco, Adwaita Mono, Liberation Mono, Lucida Console, monospace` | `ui-monospace, SFMono-Regular, SF Mono, Menlo, Consolas, Liberation Mono, monospace` |

**Headings:**

| Heading | Meatcheck | GitHub Dark |
|---------|-----------|-------------|
| Shared | `margin: 1.3em 0 0.5em 0; line-height: 1.25` | `margin-top: 1.5rem; margin-bottom: 1rem; font-weight: 600; line-height: 1.25` |
| h1 | `32px` | `2em` (32px at 16px base) |
| h2 | `24px` | `1.5em` (24px at 16px base) |
| h3 | `20px` | `1.25em` (20px at 16px base) |
| h4 | No rule (inherits body size) | `1em`, `font-weight: 600` |
| h5 | No rule | `.875em` (~14px), `font-weight: 600` |
| h6 | No rule | `.85em` (~13.6px), `font-weight: 600`, color: `#9198a1` (muted) |
| h1/h2 border | None | `border-bottom: 1px solid #3d444db3; padding-bottom: .3em` |
| font-weight | Not set (browser default bold) | Explicitly `600` on all headings |

**Observations:**
- Heading sizes match for h1-h3. h4-h6 have no styles in Meatcheck.
- GitHub uses `em` units (relative to parent) while Meatcheck uses `px` (absolute). The computed sizes are equivalent at 16px base.
- GitHub uses `rem` for heading margins (`1.5rem` top / `1rem` bottom), Meatcheck uses `em` (`1.3em` top / `0.5em` bottom). The Meatcheck top margin is slightly smaller, and the bottom margin is noticeably smaller.
- GitHub adds a bottom border to h1 and h2 for visual separation. Meatcheck does not.
- `font-weight: 600` is explicit on all GitHub headings. Meatcheck relies on browser defaults.

### Spacing

| Element | Meatcheck | GitHub Dark |
|---------|-----------|-------------|
| Paragraphs | `margin: 0 0 8px 0` | `margin-top: 0; margin-bottom: 1rem` (16px) |
| Lists (ul/ol) | `margin: 0 0 10px 22px` | `padding-left: 2em; margin-bottom: 1rem` |
| List item spacing | None | `li + li { margin-top: .25em }` |
| Blockquotes | `margin: 0 0 10px 0; padding-left: 12px` | `margin: 0; padding: 0 1em; margin-bottom: 1rem` |
| Tables | `margin: 0 0 12px 0` | `margin-bottom: 1rem` |
| Code blocks (pre) | `padding: 10px 12px` | `padding: 1rem` (16px) |
| Horizontal rules | No styles | `height: .25em; margin: 1.5rem 0; background-color: #3d444d` |

**Observations:**
- Meatcheck paragraph spacing (`8px`) is half of GitHub's (`16px` / `1rem`). This makes text blocks feel denser.
- GitHub consistently uses `1rem` bottom margins across most block elements. Meatcheck uses mixed pixel values (8px, 10px, 12px).
- Lists in GitHub use `padding-left: 2em` for indentation; Meatcheck uses `margin: 0 0 10px 22px` (margin-based, ~22px ≈ 1.375em).
- No `li + li` spacing in Meatcheck — list items have no inter-item gap.
- No horizontal rule (`hr`) styling exists in Meatcheck. HR elements will render with browser defaults (typically a thin etched line).

### Images

| Property | Meatcheck | GitHub Dark |
|----------|-----------|-------------|
| `max-width` | **Not set** | `100%` |
| `display` | Not set (inline default) | `block` |
| `box-sizing` | Not set | `content-box` |
| Overflow prevention | None — images can overflow container | `max-width: 100%` constrains to container width |

**Observations:**
- Meatcheck has **no `img` rules at all** in its CSS. Images rendered from markdown will use browser defaults (`display: inline`, no max-width constraint).
- If a markdown document contains a large image (e.g., a screenshot), it will overflow the `.markdown-file-body` container (which has `max-width: 980px`) and cause horizontal scrolling or layout breakage.
- GitHub's approach is straightforward: `max-width: 100%; display: block` prevents all overflow while maintaining aspect ratio.
- The Go-side image rewriting (`rewriteMarkdownImageSources`) handles URL translation but does not inject any inline styles or classes on `<img>` elements.

### Inline Code

| Property | Meatcheck | GitHub Dark |
|----------|-----------|-------------|
| Background | `rgba(45, 108, 223, 0.12)` (blue tint) | `#656c7633` (grey, 20% opacity) |
| Padding | `0 4px` | `.2em .4em` |
| Border-radius | `0` | `6px` |
| Font-size | Inherited | `85%` (~13.6px) |
| White-space | Inherited | `break-spaces` |

**Observations:**
- Meatcheck uses a blue-tinted background while GitHub uses a neutral grey. The blue tint is a deliberate Meatcheck branding choice.
- GitHub rounds inline code with `6px` border-radius; Meatcheck uses `0` (square corners, consistent with the UI's angular design language).
- GitHub reduces inline code font-size to 85% of surrounding text. Meatcheck inherits the parent size, making code the same height as prose.
- GitHub uses `white-space: break-spaces` allowing long inline code to wrap. Meatcheck inherits default white-space behavior.

### Code Blocks

| Property | Meatcheck | GitHub Dark |
|----------|-----------|-------------|
| Background | `#0f131a` | `#151b23` |
| Padding | `10px 12px` | `1rem` (16px all sides) |
| Border-radius | `0` | `6px` |
| Font-size | Inherited | `85%` (~13.6px) |
| Line-height | Inherited | `1.45` |
| `code` inside `pre` | No reset — inline code styles leak in | `background: transparent; padding: 0; border: 0` |

**Observations:**
- Meatcheck does not reset `code` inside `pre`. This means the inline code background (`rgba(45, 108, 223, 0.12)`) and padding (`0 4px`) will apply inside code blocks, adding unwanted styling. GitHub explicitly strips all inline code styling when `code` is inside `pre`.
- The missing `pre > code` reset is a visual bug — code blocks will have a subtle blue-tinted background overlay on top of the `#0f131a` pre background.

### Blockquotes

| Property | Meatcheck | GitHub Dark |
|----------|-----------|-------------|
| Border-left | `3px solid var(--accent-soft)` (`#3a1212`, dark red) | `.25em solid #3d444d` (grey) |
| Padding | `padding-left: 12px` | `padding: 0 1em` |
| Color | `var(--muted)` (`#b7a7aa`) | `#9198a1` |
| Margin | `0 0 10px 0` | `margin: 0; margin-bottom: 1rem` |

**Observations:**
- Meatcheck uses its red accent for the blockquote border. GitHub uses a neutral grey.
- GitHub pads both left and right (`0 1em`); Meatcheck only pads left (`12px`).
- Both use muted text color, slightly different tones.

### Links

| Property | Meatcheck | GitHub Dark |
|----------|-----------|-------------|
| Color | `var(--accent)` (`#c00000`, red) | `#4493f8` (blue) |
| Text-decoration | Not set (browser default underline) | `none` (underline on hover) |

**Observations:**
- Meatcheck uses its red accent for links. GitHub uses a standard blue.
- GitHub explicitly removes underlines until hover. Meatcheck inherits browser default (typically underlined).

### Tables

| Property | Meatcheck | GitHub Dark |
|----------|-----------|-------------|
| Cell padding | `6px 10px` | `6px 13px` |
| Border color | `var(--border)` (`#2a1b1e`) | `#3d444d` |
| Header weight | Not set (browser default bold) | Explicit `600` |
| Row striping | None | `tr:nth-child(2n) { background: #151b23 }` |
| Overflow | Not handled | `display: block; max-width: 100%; overflow: auto` |

**Observations:**
- Cell padding is nearly identical (3px wider in GitHub).
- GitHub adds alternating row backgrounds for readability. Meatcheck does not.
- GitHub wraps tables in `display: block; overflow: auto` to prevent overflow on wide tables. Meatcheck has no overflow handling for tables.

### Missing Elements (no Meatcheck styles)

The following elements have styles in GitHub's markdown CSS but no corresponding rules in Meatcheck:

- `hr` — horizontal rules (GitHub: `.25em` tall bar with `#3d444d` fill, `1.5rem` vertical margin)
- `img` — image constraints (GitHub: `max-width: 100%; display: block`)
- `pre > code` — code block code reset (GitHub: strips inline code background/padding)
- `h4`, `h5`, `h6` — smaller heading sizes
- `li + li` — list item spacing
- `dl`, `dt`, `dd` — definition lists
- `kbd` — keyboard shortcut styling
- `details`, `summary` — disclosure widgets
- Word-wrap / overflow-wrap on the markdown container

## Key Files

| File | Role |
|------|------|
| `internal/ui/styles.css:417-513` | All `.markdown` CSS rules |
| `internal/ui/styles.css:234-248` | `.markdown-file-preview` and `.markdown-file-body` container |
| `internal/app/assets.go:25-31` | Goldmark renderer setup (GFM extension) |
| `internal/app/assets.go:33-39` | `renderMarkdown()` function |
| `internal/app/assets.go:41-48` | `renderMarkdownDocument()` with image rewriting |
| `internal/app/assets.go:50-84` | `rewriteMarkdownImageSources()` — rewrites relative img src to `/file?path=` |
| `internal/app/assets.go:86-94` | `isExternalAssetURL()` — detects external URLs |
| `internal/ui/template.html:27` | Comment body markdown rendering |
| `internal/ui/template.html:58` | Prompt markdown rendering |
| `internal/ui/template.html:141-148` | Markdown file preview rendering |

## References

- [sindresorhus/github-markdown-css](https://github.com/sindresorhus/github-markdown-css) — Canonical replication of GitHub's `.markdown-body` CSS
- [github-markdown-dark.css v5.8.0](https://unpkg.com/github-markdown-css@5.8.0/github-markdown-dark.css) — Dark theme CSS used for all comparison values
- [Primer Typography Foundations](https://primer.style/foundations/typography/) — GitHub's design token system
