# Configuration

Each generated project contains a `basestack.json` file. JSON is the initial configuration format to keep the first CLI dependency-free. YAML/TOML can be adapters later; there is no new programming language to learn.

```json
{
  "schemaVersion": 1,
  "name": "my-app",
  "sections": [
    {
      "id": "hero",
      "type": "hero",
      "title": "Build your next idea.",
      "text": "A website assembled from reusable sections."
    }
  ]
}
```

- `schemaVersion` must be `1`.
- `name` and section `id` start with a lowercase letter and use lowercase letters, digits or hyphens, up to 50 characters.
- `sections` is an ordered array; move objects to change display order. An empty array is valid.
- Section IDs must be unique.
- `type` is `navbar`, `hero`, `features`, `pricing`, `contact` or `footer`.
- `variant` is optional. Omit it or use `default` for the original layout. Alternative values: navbar `minimal`, hero `split`, features `list`, pricing `compact`, contact `centered`, footer `minimal`.
- `title` is non-empty text. `text` is a string; it may be empty.
- Unknown fields are rejected to catch typos.

The CLI assigns IDs such as `features`, `features-2`, `features-3` automatically. Removal uses the ID, not the type. Links from the navbar use section IDs, so removing or renaming a section updates navigation when the page reloads.

## Multiple pages

The root `sections` array is always Home. Add an optional `pages` array for other pages; each object requires `slug`, `title`, and `sections`. Slugs follow the same naming rules as IDs, are unique, and cannot be `home`. Section IDs are unique within each page, so every page may have its own `hero` and `footer`.

```json
{
  "schemaVersion": 1,
  "name": "my-app",
  "sections": [],
  "pages": [
    {
      "slug": "about",
      "title": "About us",
      "sections": [
        { "id": "hero", "type": "hero", "variant": "split", "title": "Meet our team", "text": "Our story starts here." }
      ]
    }
  ]
}
```

```sh
basestack page add about --title "About us"
basestack page list
basestack add features --page about --variant list
basestack remove features --page about
basestack page remove about
```

`page remove` removes the page and its sections from the manifest. Commit your generated project before editing if you need version history. Home cannot be removed. Without `--page`, section commands affect Home. Flags must follow the section type/ID or page slug; values containing spaces must be quoted. Unknown flags, duplicate flags, unknown pages and unsupported variants are errors that leave the manifest unchanged.

Navigation lists Home and additional pages when a navbar exists. Each page has independent navbar/footer content. Page links use `?page=<slug>`; section links stay on the current page. Missing page slugs render a recovery link to Home. An empty page renders an empty-canvas message. Omitting `pages` remains valid; explicit `pages: null` is rejected.

Titles and body text render as plain React text, not HTML. Do not put secrets in this file: it is part of the frontend bundle. Edit `src/sections.tsx` to customise cards, link labels, email addresses and richer content. Edit CSS variables in `src/styles.css` to change the theme. Google Fonts is optional; remove the CSS import to rely on system sans-serif fonts without external font requests.

`basestack check` validates configuration. `basestack build` validates before invoking TypeScript and Vite. The frontend displays an actionable error for structurally invalid JSON values; syntactically broken JSON produces Vite's normal build/development error.
