# BaseStack

**Build applications from reusable building blocks. No AI required.**

BaseStack helps customers and developers assemble real applications using a CLI, configuration and editable components. **0.2.0 — Composition Engine** provides a standalone React + TypeScript frontend with multiple pages, six component types, twelve variants and three token-based themes. A future visual Studio will use the same versioned application schema.

No account, AI subscription, BaseStack server or payment is required for local development. There is no AI functionality.

## Quick start

Prerequisites: Go 1.23+, Node.js 22+, npm and Git.

```sh
git clone https://github.com/Janon-Emersion-T/Basestack.git
cd Basestack
go install ./cmd/basestack
# Add your Go bin directory to PATH if needed.
basestack init my-business
cd my-business
npm install

basestack page add about
basestack page add services
basestack page add contact
basestack add hero --page home --variant split
basestack add features --page home --variant cards
basestack add contact --page contact --variant centered
basestack add footer --page home --variant columns
basestack theme set modern
basestack check
basestack dev
```

Open the URL printed by Vite. Edit `basestack.json` to change content, order, pages or theme; saving reloads the app. Home starts at `/` with navbar, hero and footer. New pages start empty. Adding content inserts it before the first footer; repeated types get IDs such as `hero-2`.

## Commands

```sh
basestack help
basestack templates                 # All components and variants
basestack templates hero
basestack page list
basestack page add company --path /company/about --title "About Us"
basestack add navbar --page company --variant centered
basestack remove hero-2 --page home
basestack page remove company
basestack theme list
basestack theme set minimal
basestack theme set default
basestack check
basestack build
basestack version
```

Options follow the positional argument. Omit `--page` to select `home`, or the only remaining page. With multiple pages and no `home`, the CLI asks you to specify a page. The final page cannot be removed. Removing a page deletes its sections; review your configuration or commit it before removing content.

| Component | Variants |
| --- | --- |
| navbar | default, centered |
| hero | default, split |
| features | default, cards |
| pricing | default, simple |
| contact | default, centered |
| footer | default, columns |

All components use validated, type-specific `props`. Themes `default`, `modern` and `minimal` change fonts, colours, spacing, radius and container width through shared design tokens.

## Your project, your source

Generated applications work independently with `npm run dev`, `npm test`, `npm run typecheck` and `npm run build`. You own and can edit every generated file. There are no BaseStack service calls or external font requests. CLI upgrades do not replace generated source.

Build output is in `dist/`. URL routing uses normal links with full page loads. A static host must serve `index.html` for application paths such as `/about`; hosting configuration and deployment automation are not implemented. Applications currently assume hosting at the domain root.

Replace sample content and email addresses before publishing. Pricing is presentational and contact uses `mailto:`. Authentication, RBAC, billing, licensing, database/PostgreSQL, storage, realtime, deployment and Studio are **not implemented**.

**Existing 0.1 projects:** schema v1 is rejected explicitly. Automatic migration is deferred to avoid replacing customized frontend source. See the [manual upgrade guide](docs/CONFIGURATION.md#upgrading-a-schema-v1-project).

## Development and verification

```sh
gofmt -w cmd internal
go vet ./...
go test ./...
go test -race ./...
go build -o bin/basestack ./cmd/basestack

# Generate outside the repository to avoid committing demo artifacts.
cd "$(mktemp -d)"
basestack init smoke-app
cd smoke-app
npm install
npm test
npm run typecheck
basestack build
```

The Go CLI uses the standard library only. React, TypeScript and Vite remain the frontend dependencies; no routing library is needed. Commit a generated project's `package-lock.json` after installing; use `npm ci` for subsequent reproducible installs.

GitHub Actions runs Go validation/tests, generates a multi-page demo, exercises variants/themes, runs frontend tests/type checking and builds production assets. Successful runs provide Linux CLI and demo build artifacts.

See [configuration](docs/CONFIGURATION.md), [architecture and roadmap](docs/ARCHITECTURE.md), and [working from your phone](docs/MOBILE.md).

Created by LKProfessionals (Pvt) Ltd.
