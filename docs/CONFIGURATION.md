# Configuration — schema v2

`basestack.json` is the shared application model for the CLI, generated renderer and future Studio. No separate editor-only model is planned. All content is public frontend data; do not put secrets in this file.

```json
{
  "schemaVersion": 2,
  "name": "my-app",
  "theme": "modern",
  "pages": [
    {
      "id": "home",
      "path": "/",
      "title": "Home",
      "sections": [
        {
          "id": "hero-main",
          "type": "hero",
          "variant": "split",
          "props": {
            "title": "Build faster",
            "subtitle": "Build applications with BaseStack",
            "primaryButton": {
              "label": "Get Started",
              "href": "/contact"
            }
          }
        }
      ]
    },
    {
      "id": "contact",
      "path": "/contact",
      "title": "Contact Us",
      "sections": []
    }
  ]
}
```

## Application and page rules

- `schemaVersion` is the integer `2`. Other versions fail clearly.
- `name`, page IDs and section IDs match `^[a-z][a-z0-9-]{0,49}$`: start with a lowercase letter, then lowercase letters, digits or hyphens, at most 50 characters.
- `theme` is `default`, `modern` or `minimal`.
- `pages` is a non-empty array. Every page has a unique ID, unique path, non-blank title and a `sections` array.
- Paths are `/` or lowercase segments such as `/about` or `/services/web-design`. Each segment starts with a lowercase letter or digit, followed by lowercase letters, digits, underscores or hyphens. No trailing slash, empty segment, whitespace, dot segment, query, fragment, percent encoding or backslash is allowed.
- A root page is created automatically, but is not required after edits. Visiting an unconfigured URL shows “Page not found”; there is no implicit redirect.
- Sections are ordered and may be empty. Section IDs are unique **within a page**. `content` is reserved for the skip-link target.
- Unknown fields, including nested props and incorrectly capitalized keys, duplicate JSON keys, missing required fields, `null` values and trailing JSON values are rejected.
- Defaults are supplied by `init`/`add`, not silently filled in while reading config. `variant` and `props` are always explicit in saved configuration.

`basestack check` and `basestack build` validate before proceeding. Independent `npm run build` also validates the same config. The browser validates the raw JSON and displays an actionable error; it never renders unvalidated sections.

## Component props

Every component requires `title` (non-blank string) and `subtitle` (string, possibly empty). Additional properties are specific to its type:

| Type | Variants | Additional props |
| --- | --- | --- |
| navbar | default, centered | `links`: required array of `{ "label": "About", "href": "/about" }`; empty uses all application pages |
| hero | default, split | `primaryButton`: optional `{ "label": "Contact", "href": "/contact" }` |
| features | default, cards | `items`: required array of `{ "title": "Feature", "text": "Description" }` |
| pricing | default, simple | `plans`: required array of `{ "title": "Service", "price": "Contact us", "description": "Details" }` |
| contact | default, centered | `email`: required email address string |
| footer | default, columns | `links`: required array of `{ "label": "About", "href": "/about" }` |

Array entries require all documented fields; titles, labels and prices must be non-blank. Arrays may be empty. Optional `primaryButton` can be omitted but cannot be `null`. Text renders as escaped React text, never raw HTML.

Links accept root-relative paths, fragment anchors, HTTP(S) URLs and simple `mailto:` email links. Protocol-relative URLs, script/data URLs, whitespace and backslashes are rejected. Links are validated for safe syntax, not destination existence: removing a page updates automatic navbar links, but you must update custom links and buttons yourself. The CLI does not rewrite user-authored links.

Use `basestack templates <type>` to list variants, and edit props directly in JSON. There is no props-editing CLI or package installation command yet.

## Themes and registry

`internal/registry/builtin.json` defines each component's variants, default variant, default props and recursive property rules. Supported rule types are string, object and array, with required fields, non-empty strings and link/email formats. This intentionally small contract supports different schemas per component without arbitrary unchecked JSON.

The same catalog is embedded in the CLI and copied as `src/registry.json` into generated projects. The Go and JavaScript validators consume these definitions. Adding a component requires its definition, React props type, renderer, styles and tests. Adding a theme requires a token entry; component CSS is reused.

Themes contain `font`, `background`, `foreground`, `primary`, `secondary`, `muted`, `border`, `radius`, `spacing` and `container`. The renderer applies these as CSS custom properties, for example `--primary` and `--container`. `default` is light with lime accents, `modern` is dark with lavender accents and larger rounding, and `minimal` uses serif type, neutral colours and tighter spacing.

Generated registry/source changes belong to the project. The CLI currently knows its built-in catalog only; arbitrary local or remote component/theme packages are not yet supported. Keep the CLI and generated catalog compatible when customizing schemas.

## Upgrading a schema v1 project

Automatic `basestack migrate` is **not implemented** in 0.2. A v1 manifest's content mapping is straightforward, but its renderer expects v1. Updating only JSON would break it, and replacing source could destroy customizations. All v1 CLI project operations fail with an explicit manual-migration message without writing anything.

A safe manual upgrade:

1. Commit or back up the entire v1 project. Keep it usable with the 0.1 CLI or its existing npm commands.
2. Use 0.2 to initialize a **different, new directory**. Never initialize over the old project.
3. Use the generated v2 `basestack.json` as the starting point. Preserve the old project `name`, set `theme` to `default`, and put the old ordered sections in the Home page at `/`.
4. For each old section, preserve its `id` and `type`, set `variant` to `default`, and map old `title` to `props.title` and old `text` to `props.subtitle`. Add the type-specific defaults above: navbar/footer `links: []`, features `items`, pricing `plans`, contact `email`; hero's button is optional. Copy actual customized cards, plans, links and email from the old source where applicable. Rename any old section named `content` and update its anchors.
5. Compare and manually port frontend/CSS customizations into the new source. Copy other project files deliberately; do not overwrite the new scaffold blindly.
6. Run `npm install`, `basestack check`, `npm test`, `npm run typecheck` and `basestack build`. Review each page locally before replacing the old directory.

A future migration tool should provide a dry-run, backups, scaffold-version detection and explicit handling of modified source before offering automated source upgrades.
