# __PROJECT_NAME__

Created with BaseStack 0.3.2. No AI or BaseStack account required.

```sh
npm install
cp .env.example .env
# Uncomment BASESTACK_ENV=development and BASESTACK_AUTH_DELIVERY=local.
basestack services start
basestack db migrate
# Run basestack api in another terminal, then:
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

Replace sample email/content before publishing. Contact uses `mailto:` and pricing is presentational. PostgreSQL, SQL migrations and a Go API and Auth Core (signup, credential verification, verification/reset challenges) are included. Opaque sessions, RBAC, local bucket storage and compiled server functions are included. CRUD APIs, payments/billing, licensing, realtime, Studio and backend forms are not implemented. Do not place secrets in frontend configuration.

Commit `package-lock.json` after installing dependencies; use `npm ci` on subsequent installs. To upgrade a v1 project, back it up and port its content/customizations into a new v2 project; never change only the schema version in the old renderer.

## Go application runtime

`basestack/services.json` is a separate, strict services schema v3. Composition schema v2 in `basestack.json` is unchanged. Edit enabled flags, ports, API host and explicit CORS origins in the service file.

```sh
basestack services start
basestack db status
basestack migration new create_example
# Edit basestack/migrations/000004_create_example.sql, then:
basestack db migrate
basestack api
```

The API defaults to `http://127.0.0.1:54321/api/health`; local PostgreSQL uses port `54322`. Keep API and frontend processes in separate terminals. `basestack dev` reports API health before starting Vite. `services status` reports the container; `services stop` preserves its data volume.

First startup writes a random private password to `.basestack/local.env` (ignored, mode 0600). `.env` overrides that file; a selected `.basestack/environments/<environment>.json` profile overrides `.env`, and process variables override all files. Values are literal KEY=value, with optional quotes and whole-line comments, without shell expansion. See `.env.example` for `BASESTACK_DATABASE_URL`, `BASESTACK_API_HOST`, `BASESTACK_API_PORT`, `BASESTACK_DATABASE_PORT` and `BASESTACK_CORS_ORIGINS`. Never put secrets in frontend configuration or VITE variables.

An explicit database URL can point at any ordinary PostgreSQL server, without Docker. The local derived URL disables TLS for loopback development only. Configure verified TLS for remote connections. Deleting/changing the local credential file does not change passwords already stored in a PostgreSQL volume.

Migrations are ordered SQL. Optional `.down.sql` companions must exist before application; `basestack db rollback` reverses the latest applied migration using its recorded, checksummed down file. BaseStack records SHA-256 checksums in `basestack_internal.schema_migrations` and commits the entire pending batch transactionally. Failed batches roll back; applied files must not be edited. Do not include transaction-control commands or nontransactional operations such as CREATE INDEX CONCURRENTLY. The empty initial migration is followed by `000002_auth_core.sql`, creating users, challenges and events in `basestack_auth`. `000003_application_services.sql` adds sessions and RBAC.

You own the runtime source and database. The CLI compiles your source; it never replaces it. Independent commands from the project root:

```sh
go run ./cmd/server status
go run ./cmd/server migrate
go run ./cmd/server serve
go test ./...
docker compose --env-file .basestack/local.env ps --all
docker compose --env-file .basestack/local.env stop postgres
```

Direct Docker commands should receive the same port/environment overrides as the CLI. Use ordinary PostgreSQL tooling for inspection and backups. The API has bounded HTTP/database timeouts, graceful shutdown, explicit-origin CORS without credentials, and sanitized error output. Auth issues opaque bearer sessions with 12-hour expiry; no cookie is set.

## Auth Core

Run `basestack auth status` to inspect configuration. Auth and required email verification are enabled in `basestack/services.json`. Production is the default environment; startup requires a delivery provider. For local use, explicitly set `BASESTACK_ENV=development` and `BASESTACK_AUTH_DELIVERY=local` in `.env`. Private verification/reset messages are written to `.basestack/auth-outbox/` (0700 directory, 0600 files); never publish these files. Production rejects local delivery. To use real delivery, implement `auth.Delivery` and inject it through `app.RunWithDelivery`.

JSON POST routes: `/api/auth/signup`, `/api/auth/login`, `/api/auth/verify-email`, `/api/auth/verify-email/request`, `/api/auth/password/forgot`, `/api/auth/password/reset`. Passwords accept passphrases of at least 15 characters and at most 1024 bytes. Login returns `credentialsVerified: true, sessionIssued: true` plus `data.session.token`. Use its bearer header for `GET /api/auth/me` and `POST /api/auth/logout`; never log the token.

Passwords use salted Argon2id; database challenges store only SHA-256 digests and expire after 24 hours (verification) or 30 minutes (reset). Challenges are single-use. Forgot/resend return generic accepted responses. Signup distinguishes success from duplicates. Per-IP, per-route in-memory rate limits reset on restart and are not shared across instances. No production email provider or Auth UI is bundled. See the source repository `docs/AUTH.md` for the full contract and security limitations.

## Environments, storage, roles and functions

```sh
basestack env list
basestack env set development PRIVATE_KEY < /private/value-file
basestack env show development  # values always redacted
basestack env unset development PRIVATE_KEY
basestack services list
basestack storage create-bucket files
basestack storage put files < ./document.pdf
basestack storage list files
basestack functions list
basestack functions inspect hello
basestack functions run hello
```

Profiles support development/test/production; select with BASESTACK_ENV. Test and
production require an injected delivery provider. Private state belongs in ignored
`.basestack/`; do not commit it. Local storage uses private buckets and generated
object IDs, not uploaded filenames. `storage get <bucket> <id>` writes bytes to
stdout; `storage delete <bucket> <id>` deletes an object.

Edit `internal/runtime/functions/example.go` to register trusted compiled handlers.
`hello` is public; `whoami` requires `profile.read`. Use `roles create <role>`,
`roles grant <role> <permission>` and `roles assign <user-uuid> <role>` as a trusted
operator. Storage HTTP needs `storage.<bucket>.read`/`.write`. Private function CLI
runs accept BASESTACK_FUNCTION_TOKEN from private environment values. Roles have
no implicit privileges. Never return secrets from your function handlers.

`src/services.json` contains only public API connection settings. Import `base`
from `src/services` for `base.auth`, `base.storage` and `base.functions`. Tokens are
kept in memory; browser reload requires login. Set apiURL to your API origin, or
empty for same-origin hosting. Never import backend config into frontend code.
See the source repository docs/SERVICES.md for the complete contract and limits.
