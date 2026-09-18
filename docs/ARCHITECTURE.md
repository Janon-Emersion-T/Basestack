# Architecture and V1 roadmap

## Product principle

Customers and developers should assemble applications without requiring AI. The CLI and future Studio use the same inspectable versioned project configuration. Generated source, SQL and PostgreSQL belong to the application owner; no BaseStack-hosted infrastructure is required.

## Current implementation — 0.3.1 Auth Core

| Location | Responsibility |
| --- | --- |
| `cmd/basestack/main.go` | Thin entry point, signal context and exit status |
| `internal/cli` | Composition commands, service/database commands, generated runtime compilation and foreground process management |
| `internal/project` | Unchanged composition schema v2, validation and atomic writes |
| `internal/registry` | Portable component/variant/props definitions and theme tokens |
| `internal/strictjson` | Shared exact-field, duplicate-key and trailing-value rejection |
| `internal/runtime/config` | Services schema v2 (legacy v1 supported) and private environment loading/overrides |
| `internal/runtime/database` | pgx pool creation, connectivity checks and safe error messages |
| `internal/runtime/migrations` | SQL discovery/creation, checksums, metadata, locking and atomic application |
| `internal/runtime/auth` | Identity service, SQL repository, Argon2id, digest challenges, delivery, rate limits and HTTP |
| `internal/runtime/server` | Standard Go HTTP boundary, health route, CORS and bounded shutdown |
| `internal/runtime/app` | Runtime wiring and serve/migrate/status entry points |
| `internal/runtime/command` | Source of generated `cmd/server/main.go` |
| `internal/services` | Local credential creation and Docker Compose operations |
| `internal/scaffold` | Exclusive new-directory creation and embedded source copying |
| `scripts/verify-generated.sh` | Full real-PostgreSQL and generated-application verification |

The application runtime is one **modular monolith**, not one deployable service per feature. Auth and future RBAC/data modules attach to the same HTTP/runtime boundary unless later scaling evidence justifies a split.

Dependency direction: CLI → project/scaffold/services; services → runtime config; runtime app → config/database/migrations/server/auth; Auth HTTP → service → repository → PostgreSQL; migrations → database; config and registry → strictjson. Scaffold embeds runtime source, not an opaque server binary. The runtime has no dependency on the CLI or frontend renderer. Go's standard library supplies HTTP; pgx supplies direct PostgreSQL access.

The embedded dependency manifest/checksums match the repository's pinned pgx dependencies and are tested for drift. The Go 1.23-compatible pgx release is pinned explicitly rather than floating. Runtime sources and their tests are copied with the generated module path substituted, so owners can run and test independently. Changing dependency pins requires syncing `internal/runtime/module.txt` and `sums.txt` after `go mod tidy`.

## Configuration contracts

**Composition schema v2 is unchanged:** `schemaVersion`, `name`, `theme`, `pages`, and page-scoped ordered sections with `id`, `type`, `variant`, `props`. Old v2 configs are not reinterpreted. Unknown `services` fields in that manifest remain invalid.

**Services schema v2 is separate:** `basestack/services.json` specifies database/API enabled flags, ports, bind host, CORS origins and explicit Auth/verification flags. Legacy services v1 remains Auth-free. This explicitly versioned sidecar extends the project without changing a strict browser-facing schema or bundling backend settings into browser assets. Existing projects without it remain frontend-only. There is no automatic source upgrade; adoption is reviewed copying/merging of generated runtime files.

Public configuration contains no connection passwords. Private process variables, `.env` and `.basestack/local.env` supply runtime values without mutating global environment state. See [services configuration and compatibility](SERVICES.md#configuration-and-compatibility-decision).

## Runtime and PostgreSQL

Startup creates a standard pgx pool and verifies connectivity before opening the HTTP listener when database support is enabled. Database operations receive contexts, pools close after HTTP shutdown, and errors omit sensitive driver details. `/api/health` reflects a real pool ping; it reports unavailable/disabled states explicitly.

PostgreSQL remains directly accessible using SQL, pgx, psql, pg_dump and external tools. Local Compose runs PostgreSQL 17 with loopback port publishing, persistent volumes and a health check. It does not deploy an API microservice. `services stop` preserves data; there is no reset/drop CLI.

Migrations are owner-written SQL ordered by six-digit sequence. `basestack_internal.schema_migrations` records applied version/name/checksum/time. A transaction-scoped advisory lock serializes runners. The pending batch and its metadata commit together; failures roll back. Status is read-only with respect to persistent metadata. Applied history must match a prefix of local files exactly. The initial migration is an empty baseline; the next migration creates the Auth users, challenges and events tables.

HTTP has explicit-origin CORS, JSON errors, basic headers and bounded request/idle/shutdown timeouts. Auth verifies credentials and account challenges; sessions and permission enforcement remain deferred. Default local access is loopback-only; production TLS, secrets, authorization and deployment remain separate work.

## Composition frontend — preserved

Six component types each have two variants. Shared CSS applies three token themes. The registry catalog is copied into each project and consumed by Go/JavaScript validators; React uses discriminated TypeScript props. Page routing matches URL paths exactly using ordinary links. Static hosting still requires fallback to `index.html` and assumes domain-root hosting.

Accessibility includes one page h1, semantic elements, labeled navigation, visible keyboard focus and a skip link. Tests cover rendering/escaping/navigation; this is not a full accessibility audit. No API health call or backend environment value is inserted into the public React application.

## Safety and ownership

`init` refuses existing files, directories and symlinks. Composition mutations validate before temporary-file/rename replacement. Service startup creates random credentials only in ignored local state and never overwrites them. Each generated Compose project has a unique name. SQL creation uses exclusive locking/file creation; historical SQL is never rewritten automatically.

`api`/`db` compile the project's own Go source into a unique ignored temporary directory and run it in the foreground. Cancellation signals the owned process tree, waits for bounded shutdown, and falls back to termination if needed. Windows uses native process-tree termination when programmatic interrupt is unavailable; run the generated binary directly for native console signal behavior. Frontend development remains a separate, independently usable npm workflow.

No CLI/source upgrade silently overwrites generated customizations. Registry package installation and automatic source migration remain future work. Concurrent composition editors are still unsupported; database migration concurrency is handled by PostgreSQL locking.

## Tests and CI

The original 0.1/0.2 lifecycle, schema, props, variants/themes and rendering tests remain. New tests cover Auth password/token/lifecycle security, concurrent signup/redemption, HTTP privacy and PostgreSQL constraints, strict services/env configuration, unavailable databases, safe errors, HTTP lifecycle/health/CORS, migration discovery/order/checksums/rollback/concurrency, command validation, generated ownership and private credential files.

The end-to-end script creates a fresh project, starts its PostgreSQL using Compose with random private credentials, runs root and generated Go tests/race/vet (including isolated real-database tests), checks migration idempotence and persistence, verifies live API/frontend behavior, and runs all frontend tests/typechecks/build. CI executes that script. Only the CLI and frontend build are uploaded; secrets, database files and runtime logs are not artifacts.

## Roadmap

- **0.1 Foundation — complete:** initial CLI, sections, preview/build, tests and documentation.
- **0.2 Composition Engine — complete:** multi-page schema, variants, typed props, token themes, registry and expanded tests.
- **0.3.0 Application Services Foundation — complete:** Go modular runtime, PostgreSQL, local services, SQL migrations, health and initial HTTP/environment safeguards.
- **0.3.1 Auth Core — current:** UUID accounts, Argon2id credentials, email verification/reset challenges, lifecycle, rate limits, safe audit events and HTTP. See [Auth](AUTH.md).
- **0.3.2 Sessions, Tokens & Auth Security — next:** authenticated session/token lifecycle, revocation, expiry and security hardening.
- **0.3.3 RBAC/Permissions — planned:** real role/permission enforcement against authenticated identities.
- **0.3.4 Data/CRUD — planned:** PostgreSQL-backed data modules and validated CRUD APIs with permission checks.
- **0.3.5 Auth/UI integration — planned:** real UI flows connected to the implemented Auth/session services.
- **0.4 Studio — planned:** visual composition and configuration editing using these same schemas.
- **V1 release candidate — planned:** storage, deployment adapters, package/source upgrade strategy, accessibility/security review and integration coverage.

Sessions/tokens, RBAC, CRUD APIs, billing/licensing, AI, storage, realtime, Studio and deployment are not implemented in 0.3.1. This milestone stops at Auth Core. Login issues no session or bearer credential.
