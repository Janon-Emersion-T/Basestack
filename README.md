# BaseStack

**Build applications from reusable building blocks. No AI required.**

BaseStack is an application-building toolkit for customers and developers: choose templates, assemble sections and customise the result. The long-term goal is a shared foundation for a simple CLI and a visual drag-and-drop Studio.

## Version 1, milestone 2 — pages and variants (0.2.0)

BaseStack generates a standalone React + TypeScript website. It includes a Go CLI, multiple pages, six section types with two layouts each, editable JSON configuration, responsive styling, validation and GitHub Actions checks. It is the foundation of V1, not the complete application platform.

| Available now | Planned next |
| --- | --- |
| Project creation and independent source code | Multiple pages and template variants |
| Navbar, hero, features, pricing, contact, footer, each with an alternate layout | Dashboard, commerce and auth templates |
| Multiple pages with navigation and page-specific sections | Visual Studio with drag-and-drop |
| Add/remove sections and edit their order/content | Rich content properties and theme tokens |
| Local preview and production build commands | Go services, PostgreSQL, auth and RBAC |
| CLI/frontend validation parity and desktop/mobile browser checks | Storage, realtime and deployment adapters |

No account, AI subscription, BaseStack cloud service or payment is required. `login` and `balance` are intentionally not implemented until there is a real account/billing service.

## Quick start

Prerequisites: Go 1.23 or newer, Node.js 22 or newer, npm and Git.

```sh
git clone https://github.com/Janon-Emersion-T/Basestack.git
cd Basestack
go build -o bin/basestack ./cmd/basestack
./bin/basestack init my-app
cd my-app
../bin/basestack add features
../bin/basestack add pricing
../bin/basestack add contact
../bin/basestack page add about --title "About us"
../bin/basestack add features --page about --variant list
npm install
../bin/basestack dev
```

Open the URL printed by Vite. Use the navbar to visit About us, or open `?page=about`. Edit `basestack.json` and save to see changes. Home and new pages start with navbar, hero and footer. New content is inserted before the first footer; new navbars go at the beginning.

For a globally available CLI, run `go install ./cmd/basestack` from this repository and add your Go bin directory to PATH.

```sh
basestack templates
basestack add features
basestack remove features
basestack page list
basestack add hero --page about --variant split
basestack remove hero-2 --page about
basestack check
basestack build
```

The generated project also works with `npm run dev` and `npm run build` without the CLI. Build output is in `dist/`. Replace sample content and contact email before publishing. The pricing section is presentational; no checkout or backend form service is included.

Pages use query URLs (`?page=about`) with full-page navigation, so static hosts do not need custom rewrite rules. These are client-rendered views of one built HTML entry, not separate prerendered HTML documents. Page titles update in the browser; server-rendered metadata and clean-path routes are future work.

**Compatibility:** 0.1.0 single-page manifests remain valid. Updating the CLI does not overwrite generated source files. To add pages/variants to an older generated app, follow the [upgrade guide](docs/UPGRADING.md).

## Work directly on GitHub

You can read and edit these files in GitHub from your phone. The **Actions → BaseStack CI** workflow runs tests, compiles the CLI and builds an example app on each push. Successful runs include a Linux CLI artifact and a demo build artifact. A Linux executable does not run directly on an iPhone.

See [the mobile workflow](docs/MOBILE.md), [configuration guide](docs/CONFIGURATION.md), and [architecture and roadmap](docs/ARCHITECTURE.md).

## Development

```sh
gofmt -w cmd internal
go vet ./...
go test -race ./...
go build ./cmd/basestack
```

The CLI uses the Go standard library only. Frontend templates are embedded in the CLI binary. Commit each generated project's `package-lock.json` after installing dependencies; use `npm ci` for subsequent reproducible builds.

Created by LKProfessionals (Pvt) Ltd.
