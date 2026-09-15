# __PROJECT_NAME__

Created with BaseStack 0.2.0. No AI or BaseStack account required.

```sh
npm install
npm run dev
```

Edit `basestack.json` to change section titles, text, or their order. Vite reloads the page automatically. With the CLI installed, use `basestack templates`, `basestack add features`, `basestack remove hero`, and `basestack check`.

Create pages with `basestack page add about --title "About us"`. Add page-specific sections using `basestack add features --page about --variant list`. Use `basestack page list` to see routes. Pages use query URLs such as `?page=about`, with browser navigation and no server rewrites. Each page's navbar lists available pages automatically. Use `basestack templates` to see alternate layouts and set a section's optional `variant` in the manifest.

Edit `src/sections.tsx` for links, feature cards and richer content. Edit `src/styles.css` for colours, spacing and fonts. This is your source code: generated projects do not need the CLI to run.

```sh
npm run build
npm run preview
```

Deploy `dist/` to a static host after building. Contact uses a sample `mailto:` link: replace `hello@example.com` before deployment. Pricing text is a placeholder. No authentication, payments, contact submission service or database is included.

Commit the generated `package-lock.json` after `npm install`; use `npm ci` on subsequent builds for reproducible dependencies. Do not place credentials in frontend files or configuration.
