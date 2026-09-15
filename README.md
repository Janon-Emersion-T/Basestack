# BaseStack

**Build applications from reusable building blocks. No AI required.**

BaseStack is an application-building toolkit for customers and developers: choose templates, assemble sections and customise the result. The long-term goal is a shared foundation for a simple CLI and a visual drag-and-drop Studio.

## Version 1, milestone 1 — CLI foundation (0.1.0)

This first implementation generates a standalone React + TypeScript website. It includes a Go CLI, six reusable section templates, editable JSON configuration, responsive styling, validation and GitHub Actions checks. It is the foundation of V1, not the complete application platform.

| Available now | Planned next |
| --- | --- |
| Project creation and independent source code | Multiple pages and template variants |
| Navbar, hero, features, pricing, contact, footer | Dashboard, commerce and auth templates |
| Add/remove sections and edit their order/content | Visual Studio with drag-and-drop |
| Local preview and production build commands | Go services, PostgreSQL, auth and RBAC |
| Configuration validation and automated checks | Storage, realtime and deployment adapters |

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
npm install
../bin/basestack dev
```

Open the URL printed by Vite. Edit `basestack.json` and save to see changes. The generated app starts with navbar, hero and footer; new sections are inserted before the first footer.

For a globally available CLI, run `go install ./cmd/basestack` from this repository and add your Go bin directory to PATH.

```sh
basestack templates
basestack add features
basestack remove features
basestack check
basestack build
```

The generated project also works with `npm run dev` and `npm run build` without the CLI. Build output is in `dist/`. Replace sample content and contact email before publishing. The pricing section is presentational; no checkout or backend form service is included.

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
