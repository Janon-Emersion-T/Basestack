# Application services — BaseStack 0.3.1

BaseStack now generates a local Go application runtime alongside its React frontend. It is a modular monolith: HTTP, PostgreSQL access and migrations share one runtime. Auth Core is included; sessions/tokens, RBAC, CRUD APIs, billing/licensing, AI, storage, realtime and deployment are not implemented.

## Local workflow

Requirements: Go 1.23+, Node.js 22+, npm, Docker Engine/Desktop and Docker Compose v2 with `up --wait` support. Docker must be running and accessible to your user. Native Windows users can run CLI commands; the repository verification script uses Bash (CI runs on Linux).

```sh
basestack init my-app
cd my-app
npm install
cp .env.example .env
# Uncomment BASESTACK_ENV=development and BASESTACK_AUTH_DELIVERY=local.
basestack services start
basestack db migrate
basestack api
```

Leave the API running, then open another terminal in the same project:

```sh
basestack dev
```

The API defaults to `http://127.0.0.1:54321`, PostgreSQL to `127.0.0.1:54322`, and Vite normally uses `http://127.0.0.1:5173`. `dev` probes API health, reports whether it is healthy/unavailable/disabled, and continues the frontend when the API is unavailable. It does not start hidden background processes.

```sh
basestack services status
basestack db status
basestack services stop
```

`services start` starts **local PostgreSQL only**, waits for its container health check, and preserves existing volumes. `api` runs the Go HTTP server in the foreground. Ctrl+C stops it gracefully. `services stop` stops PostgreSQL without deleting its volume or changing credentials; stop the API separately. `services status` reports container state; `db status` actually authenticates to the configured database and checks migration history.

Each generated Compose project has a unique name. Ports are configurable but not automatically selected for normal development: if running multiple projects, give each one different API/database ports. The verification script chooses unused ports for its temporary project.

## Configuration and compatibility decision

**Composition schema v2 is unchanged.** No `services` property has been added to `basestack.json`; v2 still rejects that unknown field. Backend settings live in a separate, versioned `basestack/services.json`:

```json
{
  "schemaVersion": 2,
  "auth": {"enabled": true, "requireEmailVerification": true},
  "database": {
    "enabled": true,
    "port": 54322
  },
  "api": {
    "enabled": true,
    "host": "127.0.0.1",
    "port": 54321,
    "corsOrigins": [
      "http://localhost:5173",
      "http://127.0.0.1:5173"
    ]
  }
}
```

This sidecar extends the project configuration without changing the strict public composition contract or bundling backend settings into browser assets. Service schema versioning can evolve separately from pages/components. The CLI and a future Studio should edit this same service file, not introduce a separate private model.

Services v1 remains supported with no `auth` field (even `auth: null` is rejected). In services v2 every displayed field is required; enabled Auth requires the database. Enabled values are booleans; ports are integers from 1 to 65535; host is an IP address or `localhost`; origins are unique explicit HTTP(S) origins without paths, credentials or wildcards. `corsOrigins: []` disables cross-origin access. Unknown/case-mismatched fields, duplicate keys, missing values and unsupported versions fail validation. `basestack check` validates the service file when it exists, as well as the composition manifest.

Old v2 projects without this file remain frontend-only and retain all composition commands. Service commands fail with an upgrade message. To adopt services, generate a new 0.3 project in a different directory and review/copy `cmd/server`, `internal/runtime`, `internal/strictjson`, `go.mod`, `go.sum`, `compose.yaml`, `.env.example` and `basestack/` into the old project. Merge `.gitignore`; keep the old `basestack.json` and frontend source. Do not overwrite existing Go modules or source blindly. No automatic source upgrade is performed.

## Environment and credentials

Generated source contains **no usable database password** and no `.env` file. The first `services start` creates a random 256-bit password in ignored `.basestack/local.env` with mode `0600` (directory `0700`). It never replaces existing credentials. The local URL is derived from this password, the database port, user `basestack` and database `basestack`.

Environment precedence, highest first:

1. Process environment.
2. Project `.env`, if present.
3. `.basestack/local.env`, if present.
4. Public service settings/default local URL derivation.

Copy `.env.example` to `.env` and uncomment only the overrides you need. Both files above are parsed as literal `KEY=value`, with optional surrounding quotes and whole-line comments. There is **no shell expansion**, `export` syntax, variable interpolation or inline-comment processing. Duplicate keys within a file are rejected. Empty process variables are explicit overrides, not ignored. The loader does not mutate process-wide environment state.

| Variable | Purpose |
| --- | --- |
| `BASESTACK_DATABASE_URL` | Explicit PostgreSQL URL/pgx connection string; overrides local URL derivation |
| `BASESTACK_API_HOST` | Bind address; defaults to service config's loopback host |
| `BASESTACK_API_PORT` | API port; defaults to service config |
| `BASESTACK_DATABASE_PORT` | Local Compose host port and derived URL port |
| `BASESTACK_CORS_ORIGINS` | Comma-separated explicit origins; empty disables CORS |
| `BASESTACK_DB_PASSWORD` | Local Compose password, normally generated automatically |
| `BASESTACK_TEST_DATABASE_URL` | Opt-in integration-test URL for a disposable server with CREATEDB permission |

Never put a database URL/password in `basestack.json`, `basestack/services.json`, `VITE_*` variables or committed files. The React bundle does not import service settings or private environment files. An external PostgreSQL server only requires `BASESTACK_DATABASE_URL`; Docker is optional in that workflow. `services start` always manages the local Compose database, even if the API URL points to an external server.

The generated local URL explicitly disables TLS for the loopback development container. Configure TLS verification in the URL for remote databases. Container administrators can inspect container environment; this is a local-development credential approach, not a production secret manager. PostgreSQL's image uses its password setting when initializing a **new** data directory; editing/deleting the local credential file does not change passwords already stored in a volume.

## PostgreSQL belongs to the application owner

The runtime exposes a normal `*pgxpool.Pool`. It does not impose a proprietary query language or database API. Use parameterized, context-aware queries directly from your Go code. The pool currently has a maximum of 10 connections, five-second startup connectivity timeout, five-minute idle lifetime and one-hour maximum lifetime. Source owners can change these policies.

Use ordinary PostgreSQL SQL, `psql`, `pg_dump`, backups and external tooling. The Compose file is inspectable. Without the CLI, the default local configuration can be operated with:

```sh
docker compose --env-file .basestack/local.env up -d --wait postgres
docker compose --env-file .basestack/local.env ps --all
docker compose --env-file .basestack/local.env exec postgres psql -U basestack -d basestack
docker compose --env-file .basestack/local.env stop postgres
```

Supply the same port/environment overrides if using them; Compose's own environment interpolation rules apply to direct Docker commands. Ordinary host-side PostgreSQL clients require their normal connection settings, environment or `.pgpass`; BaseStack does not modify them.

## SQL migrations

```sh
basestack migration new create_example
# Edit basestack/migrations/000003_create_example.sql before applying it.
basestack db status
basestack db migrate
```

Files use a six-digit positive sequence and lowercase underscore name, for example `000003_create_example.sql`. The CLI refuses duplicate names, duplicate versions, unsafe names, symlinked migration files and invalid SQL filenames. Creation uses an exclusive lock and exclusive file creation; it never overwrites an existing migration. After a crash, inspect `.create.lock` inside the migrations directory before manually removing a stale creation lock.

The initial `000001_initial.sql` is an intentional empty baseline. `000002_auth_core.sql` creates `basestack_auth.users`, `challenges` and `events`; see [Auth](AUTH.md). The runner bootstraps only `basestack_internal.schema_migrations`, containing version, filename, SHA-256 checksum and application timestamp.

Rules:

- Discover and apply migrations in ascending version order.
- Acquire a transaction-scoped PostgreSQL advisory lock to serialize migration/status runners in the same database.
- Run the whole pending batch and insert its metadata in **one transaction**. On failure, all pending changes and their metadata roll back; previously applied batches remain intact.
- Record exact file-byte checksums. Changed content, renames, missing applied files or inserted historical versions cause an integrity error.
- Already-applied migrations are not rerun. `db status` does not create the metadata schema/table.
- Do not edit applied files, including formatting/line endings. Add a new forward migration to change the database. Failed/unapplied SQL may be corrected before retrying.
- Do not include `BEGIN`, `COMMIT`, `ROLLBACK`, savepoints or other transaction-control statements. PL/pgSQL function bodies can still contain their own blocks. The SQL files are trusted owner-written code, not sandboxed input.
- Nontransactional operations such as `CREATE INDEX CONCURRENTLY` are not supported by this workflow. There is no automatic down/reset/drop command or nontransactional opt-out.
- CLI/runtime migration operations have a five-minute deadline. Long migrations need a deliberate policy/source change.

Database errors report an operation and SQLSTATE where available, with driver details omitted to prevent values, SQL or connection credentials from leaking. Inspect your SQL and use standard PostgreSQL diagnostics when deeper details are needed.

## API and security baseline

`GET /api/health` checks the live pool using the request context and a two-second deadline:

```json
{"status":"ok","service":"basestack","version":"0.3.1","database":"connected"}
```

If the database becomes unavailable, health returns HTTP 503 with `status: "error"` and `database: "unavailable"`. If database support is explicitly disabled, it reports `database: "disabled"`. Enabled database connections must succeed **before** the API listens. API startup does not apply migrations automatically.

Unknown routes and unsupported methods have structured JSON errors. The server sets read/header/write/idle limits, caps request headers, returns no-store/nosniff/frame/referrer/CSP headers, and provides a bounded ten-second shutdown. Context cancellation reaches database calls; the pool closes after the HTTP server stops.

CORS matches only configured origins. It does not emit wildcard or credentialed access. Health permits GET; enabled Auth routes permit POST and matching preflights with Content-Type. If Vite chooses another port, update the allowed origin for browser API calls. CLI health probes do not send an Origin header. CORS is not authentication and does not restrict non-browser clients.

The default HTTP host and published PostgreSQL port are loopback-only. Explicitly binding the API elsewhere makes an unauthenticated API reachable there. This baseline is not a complete security layer; credential verification is implemented, but sessions, tokens and permissions remain deferred. Future middleware belongs at the HTTP boundary in the same Go runtime.

## Ownership and independent commands

`basestack api` and `basestack db` compile and run the **generated project's** source, so your runtime edits take effect. They do not use a hidden BaseStack server or silently replace that source. Go may download pinned public dependencies on the first build.

You can work without the CLI:

```sh
go run ./cmd/server status
go run ./cmd/server migrate
go run ./cmd/server serve
go test ./...
npm run dev
npm test
npm run build
```

These commands run from the project root and use the same service/environment files. Rename the generated `example.com/my-app` Go module if desired. `go.mod`, `go.sum`, SQL, Compose and source all belong to you.

## Troubleshooting

- **Docker unavailable:** start Docker Engine/Desktop, check user permissions, and confirm `docker compose version`. No containers are needed for an externally managed PostgreSQL server.
- **Port already in use:** set a free `BASESTACK_DATABASE_PORT` or `BASESTACK_API_PORT`. Restart the relevant process/container.
- **Database unavailable:** inspect `services status`, then `db status`; verify host, port, database, role, password and TLS requirements privately. No failed connection is treated as healthy.
- **Password changed/lost:** restore `.basestack/local.env` or use PostgreSQL's normal credential-management process. Restarting Compose does not rotate credentials in an existing volume. BaseStack never resets the volume to resolve this.
- **Migration integrity error:** restore the applied historical files from version control; create a new migration for the intended change. Do not alter metadata to hide a mismatch.
- **SQL failure:** the pending batch was rolled back. Correct unapplied SQL and retry; the SQLSTATE identifies the PostgreSQL error class.
- **Services file absent:** this is a frontend-only 0.2 project; follow the reviewed source-upgrade steps above.

## Verification

Run `scripts/verify-generated.sh` from the repository to create a disposable project, start its own PostgreSQL, execute root/generated Go tests (including integration and race tests), vet, frontend tests/typechecking/build, live health/CORS checks, frontend startup, migration idempotence and stop/start persistence. It stops its processes and database on exit and retains local volumes; it never resets an existing project.

Without `BASESTACK_TEST_DATABASE_URL`, ordinary Go tests skip real-database migration and Auth tests. With it, integration tests create randomly named databases on the supplied disposable server and remove only those databases afterward. CI uses the full script with random private credentials and publishes only the CLI and frontend build artifacts.

## Auth configuration and upgrades

Fresh projects enable Auth and require email verification. Runtime defaults to production mode; local delivery requires both explicit development settings in `.env`. `basestack auth status` reports configuration without secrets. [Auth documentation](AUTH.md) lists all security settings, provider wiring, endpoints and limitations.

For a 0.3.0 source upgrade, preserve every existing migration byte. Merge the new runtime/dependencies and service schema v2 deliberately. Copy `internal/runtime/auth/schema.sql` into the **next unused** migration sequence if `000002` is already occupied; never overwrite or renumber applied history. Disabling Auth removes its HTTP routes, not stored accounts or migrations.
