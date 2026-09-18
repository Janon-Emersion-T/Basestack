# Application services audit

The repository at the start of this change is 0.3.1, not 0.2. Baseline verification
passed on Linux with Go 1.23 and Node 22, including `go test`, race tests, vet,
CLI build and `scripts/verify-generated.sh`. The latter exercised real PostgreSQL,
transactional migrations, generated Go source, frontend tests/typecheck/build,
live Auth, CORS, API shutdown and database persistence across stop/start.

Composition schema v2 is strictly validated in Go and JavaScript using a shared
registry. Its atomic configuration writes, page commands and renderer should not
change. `basestack/services.json` is the backend extension point; schema v1
contains database/API settings and v2 adds Auth. Projects without the sidecar
remain frontend-only. Private environment values are loaded without mutating the
process environment. SQL migrations already have checksums, transaction locking,
ordered creation and rollback tests.

The CLI compiles the generated project's Go source. The scaffold embeds tested
runtime packages and rewrites their module imports; new runtime packages must be
added to that embed list. The HTTP boundary owns CORS, timeouts and shutdown.
`app` owns resource construction and teardown. Docker Compose manages only local
PostgreSQL; external PostgreSQL already works without Docker. Auth has credential
verification and challenge delivery, but does not issue an authenticated session.

Missing from the requested application-services scope: named environment CRUD
and test-mode support, storage, authorization policy foundations, a server
function registration boundary, and deliberate public frontend service settings.
These can be added within the current modular runtime. Preserve existing SQL,
Auth contracts and composition source. Use explicit opt-in settings for new
services and do not interpret successful login as a reusable identity credential.

The completed specification also requires sessions, logout/current-user routes,
persistent RBAC, bucket storage and explicit rollback. These extend Auth Core
through services schema v3 and an additive application-services migration. No
existing migration is rewritten. Cloud providers and deployment remain out of scope.

## Final verification

The 2026-09-18 Linux verification passed after implementation:

- `gofmt`, `git diff --check`, `go test ./...`, `go test -race ./...`, `go vet ./...`.
- `scripts/verify-generated.sh`: CLI build, root/generated Go tests and race tests
  with actual PostgreSQL, ordered migrations, explicit rollback/reapply, failed
  rollback atomicity, Auth challenges, session expiry/revocation and RBAC.
- Generated frontend: all 57 tests, TypeScript checks and production builds.
- Live CLI/HTTP: environment profiles/redaction, service lifecycle, role grants
  and revocation, private storage upload/retrieval/deletion, public/protected
  functions, current user/logout, CORS and log-secret checks.
- Frontend-only composition schema-v2 validation/build with the sidecar removed.
- Diff inspection confirmed composition schema/renderer/catalog and existing
  Auth migration bytes remain unchanged. No secrets, generated applications,
  build outputs or database files were added to the repository; no commit was made.
