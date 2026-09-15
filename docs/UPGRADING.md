# Upgrading a generated 0.1.0 app

The 0.2.0 CLI accepts 0.1.0 configuration without migration. Existing single-page apps continue working. Their old frontend, however, does not understand pages or variants. Installing a new CLI alone does not upgrade copied templates.

1. Commit or back up your generated app, including customised source files.
2. Build/install the new CLI from the BaseStack repository.
3. Generate a separate reference project with `basestack init upgrade-reference`.
4. Compare and merge these reference files into your app: `src/config.ts` (new), `src/main.tsx`, `src/sections.tsx`, `src/styles.css`, and `vite.config.ts` (new). Preserve your own components, styles, contact details and links. Do not replace your manifest with the reference manifest.
5. Run `basestack check` and `npm run build`. Check your existing page before adding another page.
6. Use `basestack page add about --title "About us"` and open `?page=about`.

There is no automatic source-upgrade command yet. Newly generated 0.2.0 apps already include all required files. Keep your existing dependency lockfile unless you intentionally update dependencies.
