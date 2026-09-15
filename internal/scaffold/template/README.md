# __PROJECT_NAME__

Created with BaseStack 0.2.0. No AI or BaseStack account required.

```sh
npm install
npm run dev
```

Edit `basestack.json` to compose pages, ordered sections, variants, props and theme. Saving reloads the app. Home starts at `/`. The navbar automatically links to all pages unless its `props.links` array supplies custom links.

```sh
basestack page add about
basestack add hero --page about --variant split
basestack theme set modern
basestack templates hero
basestack page list
basestack check
```

Every section requires `id`, `type`, `variant` and `props`. See `src/registry.json` for supported variants, property rules, default props and theme tokens. Unknown fields and invalid values fail validation. IDs use lowercase letters, digits and hyphens, starting with a letter (max 50). Page paths are `/` or lowercase URL segments without trailing slashes, queries or fragments.

Edit `src/sections.tsx` and `src/types.ts` to customize component rendering, `src/styles.css` for layouts and `src/registry.json` for tokens/defaults/property definitions. Registry defaults apply when creating sections with a compatible CLI; they do not replace existing props. The CLI uses its built-in registry, so keep schemas compatible or validate custom definitions with this project's npm commands.

This is your source code. The project runs independently of BaseStack, and CLI upgrades never replace your customizations.

```sh
npm test
npm run check
npm run typecheck
npm run build
npm run preview
```

Build output is in `dist/`. A static host must serve `index.html` for application URLs such as `/about`; routing uses full page loads and assumes hosting at the domain root. Unknown routes show a page-not-found screen. Deployment automation is not included.

Replace sample email/content before publishing. Contact uses `mailto:` and pricing is presentational. No authentication, RBAC, payments/billing, licensing, database, storage, realtime, Studio or backend form service is included. Do not place secrets in frontend configuration.

Commit `package-lock.json` after installing dependencies; use `npm ci` on subsequent installs. To upgrade a v1 project, back it up and port its content/customizations into a new v2 project; never change only the schema version in the old renderer.
