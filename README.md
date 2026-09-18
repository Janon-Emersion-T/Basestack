# BaseStack

**Build applications from reusable building blocks. No AI required.**

BaseStack helps customers and developers assemble real applications using a CLI, configuration and editable components. **0.3.1 — Auth Core** adds email/password accounts, verification and password reset to the editable Go runtime, PostgreSQL, SQL migrations and local Docker services to the existing React + TypeScript Composition Engine. Multiple pages, six component types, twelve variants and three themes remain available. A future visual Studio will use the same versioned application schema.

No account, AI subscription, BaseStack server or payment is required for local development. There is no AI functionality.

## Quick start

Prerequisites: Go 1.23+, Node.js 22+, npm and Git. Local PostgreSQL also requires Docker and Compose v2. An externally managed PostgreSQL server can be used instead.

```sh
git clone https://github.com/Janon-Emersion-T/Basestack.git
cd Basestack
go install ./cmd/basestack
# Add your Go bin directory to PATH if needed.
basestack init my-business
cd my-business
npm install
cp .env.example .env
# In .env, explicitly enable the two development settings shown below.
basestack services start
basestack db migrate
# In a separate terminal: basestack api

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

For local Auth delivery, uncomment these settings in `.env`:

```dotenv
BASESTACK_ENV=development
BASESTACK_AUTH_DELIVERY=local
```

Verification/reset messages are private files in `.basestack/auth-outbox/`. Production mode is the default and rejects local delivery; an enabled production Auth runtime requires an injected delivery provider. See [Auth architecture, API and security](docs/AUTH.md).

## Application services

```sh
basestack services start            # Start local PostgreSQL; private random credentials
basestack db status
basestack migration new create_example
# Edit the generated SQL before applying it.
basestack db migrate
basestack api                       # Foreground Go API; leave running in another terminal
basestack dev                       # Frontend; reports API health
basestack services status
basestack auth status
basestack services stop             # Preserve database volume
```

The API health endpoint is `http://127.0.0.1:54321/api/health`; local PostgreSQL uses port `54322`. Auth provides signup, credential verification, email verification and password reset; login does not issue a session. CRUD is deferred. PostgreSQL remains ordinary PostgreSQL: use SQL, pgx, psql, pg_dump and your preferred database tools.

Composition schema v2 remains unchanged. Public backend settings use a separate `basestack/services.json` schema v2 (legacy services v1 remains supported with Auth absent). Environment overrides use `.env`; local credentials live in ignored `.basestack/local.env`. Neither is committed or imported into the frontend. Existing v2 projects without services retain their frontend workflow.

Read the [services guide](docs/SERVICES.md) for configuration, local/external PostgreSQL, migration guarantees, ownership, security and troubleshooting.

## Composition commands

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

Generated applications work independently with `npm run dev`, `npm test`, `npm run typecheck` and `npm run build`. You own and can edit every generated file. There are no BaseStack-hosted service calls or external font requests. Generated Go source runs independently with `go run ./cmd/server serve`, `migrate` or `status`. CLI upgrades do not replace generated source.

Build output is in `dist/`. URL routing uses normal links with full page loads. A static host must serve `index.html` for application paths such as `/about`; hosting configuration and deployment automation are not implemented. Applications currently assume hosting at the domain root.

Replace sample content and email addresses before publishing. Pricing is presentational and contact uses `mailto:`. Sessions/tokens, RBAC, CRUD APIs, billing, licensing, storage, realtime, deployment and Studio are **not implemented**.

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

Go HTTP tooling uses the standard library; PostgreSQL uses the pinned pgx driver/pool. React, TypeScript and Vite remain the frontend dependencies; no routing library is needed. Commit a generated project's `package-lock.json` after installing; use `npm ci` for subsequent reproducible installs.

Run `scripts/verify-generated.sh` for full integration verification. GitHub Actions runs it to provision PostgreSQL with random private credentials, test the runtime, Auth and migrations (including race tests), exercise a generated multi-page demo, verify live API/frontend behavior, and build production assets. Successful runs provide Linux CLI and demo build artifacts.

See [configuration](docs/CONFIGURATION.md), [architecture and roadmap](docs/ARCHITECTURE.md), and [working from your phone](docs/MOBILE.md).

Created by LKProfessionals (Pvt) Ltd.
