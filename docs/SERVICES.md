# Application services — BaseStack 0.3.2

BaseStack owns a local-first Go application runtime, PostgreSQL migrations, Auth,
RBAC, private storage and compiled server functions. It runs independently on
machines, VPS servers and containers. There is no required BaaS or cloud vendor.
Docker Compose is optional tooling for local PostgreSQL; use an external
PostgreSQL connection without Docker. The frontend and API run independently.

## Local workflow

```sh
basestack init my-app
cd my-app
npm install
cp .env.example .env
# In .env, enable BASESTACK_ENV=development and BASESTACK_AUTH_DELIVERY=local.
basestack services start
basestack db migrate
basestack api                   # foreground, leave this terminal open
# Another terminal:
basestack dev
basestack services list
basestack services status
basestack functions list
basestack functions run hello
basestack storage create-bucket files
basestack storage buckets
basestack services stop         # stops PostgreSQL; preserves its volume
```

Prerequisites: Go 1.23+, Node 22+, npm. Local PostgreSQL uses Docker Compose v2.
API defaults to `127.0.0.1:54321`, PostgreSQL to `127.0.0.1:54322`. `dev` reports
API health and runs Vite; it does not create background processes. Ctrl+C shuts
services down gracefully. `services start/stop` manage the local database only;
storage and functions run inside the API. `services list` reports configuration;
`status` probes API health and reports local container state. `db status` checks
actual database connectivity and migration history, including external databases.
Local Compose status/start/stop need Docker even when the API uses an external DB.

## Configuration and compatibility decision

Composition schema v2 in `basestack.json` is unchanged. Backend settings use a
separate, strict `basestack/services.json` so private configuration cannot enter
the browser's composition manifest:

```json
{
  "schemaVersion": 3,
  "database": {"provider": "postgresql", "enabled": true, "port": 54322},
  "api": {
    "enabled": true, "host": "127.0.0.1", "port": 54321,
    "corsOrigins": ["http://localhost:5173", "http://127.0.0.1:5173"]
  },
  "auth": {"enabled": true, "requireEmailVerification": true},
  "authorization": {"enabled": true},
  "storage": {"enabled": true, "provider": "local", "maxObjectBytes": 10485760},
  "functions": {"enabled": true}
}
```

All displayed fields are required for v3. Unknown/case-mismatched/duplicate
fields, null required values and unsupported providers fail validation. Ports
are 1–65535; host must be an IP address or localhost. CORS origins are unique
HTTP(S) origins with no credentials, paths or wildcards. An empty array disables
cross-origin access. Auth requires the database; authorization requires Auth.
Storage limits are 1 byte–1 GiB. Disabling a service preserves its data.

Services v1 still provides database/API settings without Auth. Services v2 keeps
Auth Core's credential-only login contract. New service fields require v3;
merely installing the CLI does not rewrite generated source or change sessions.
Projects without a sidecar remain frontend-only and keep their existing commands.
`basestack check` validates the sidecar if present.

To adopt services, generate a separate project and review/merge its Go source,
module files, Compose file, `.env.example`, `.gitignore`, service sidecar and SQL.
For v1/v2 service upgrades, preserve **every existing migration byte**. Copy the
new `auth/sessions.sql` and `rbac/schema.sql` into the next unused migration;
copy `auth/schema.sql` first if upgrading from a project without Auth. Do not
renumber existing SQL. Merge the frontend client files only if wanted. Never
blindly overwrite a customized renderer, Go module or runtime.

## Environments and secrets

```sh
basestack env list
basestack env show development
basestack env set development BASESTACK_DATABASE_URL < /private/path/database-url
basestack env unset development BASESTACK_DATABASE_URL
BASESTACK_ENV=test basestack db status
```

Supported profiles are `development`, `test`, `production`. Selection uses the
process `BASESTACK_ENV`, then dotenv selection, then **production**. Profiles
cannot override their own selection. Precedence, highest first:

1. Process environment (an empty value is an explicit override).
2. Selected `.basestack/environments/<name>.json` profile.
3. `.env`.
4. `.basestack/local.env`.
5. Public settings and defaults.

Profiles have private schema v1: `schemaVersion` and a string-valued `values`
object. Writes use 0600 temporary files, atomic replacement and an exclusive
per-profile lock; directories require 0700. Symlinks, unsafe permissions, invalid
keys, NUL values, oversized profiles and duplicates are rejected. `env show`
displays sorted stored keys with **every value redacted**, and never enumerates
the process environment. `env set` reads stdin, refusing terminal echo; one final
newline is removed. Do not put secrets in shell arguments or command history.
`VITE_*` profile keys are rejected. CLI updates the project ignore file before
writing private state, including on older projects. After an interrupted write,
inspect the profile's `.lock` file before removing a stale lock.

`config.SecretStore`, injected through `app.Options.Secrets`, is the future encrypted-provider boundary. The current
store is private plaintext, not encrypted. The loader does not modify global
process variables. Dotenv files retain literal `KEY=value` semantics with
optional enclosing quotes and whole-line comments; no expansion, export syntax
or inline-comment interpretation. Keep private state out of published artifacts
and backups intended for sharing. Never force-add it to Git.

| Variable | Purpose |
| --- | --- |
| `BASESTACK_ENV` | Select development, test or production |
| `BASESTACK_DATABASE_URL` | Explicit PostgreSQL DSN; externally managed DBs need no Docker |
| `BASESTACK_DB_PASSWORD` | Automatically generated local Compose credential |
| `BASESTACK_DATABASE_PORT` | Local DB port override |
| `BASESTACK_API_HOST`, `BASESTACK_API_PORT` | API binding overrides |
| `BASESTACK_CORS_ORIGINS` | Comma-separated explicit origins; empty disables CORS |
| `BASESTACK_AUTH_DELIVERY` | `none`, `local`, `external`; local requires explicit development |
| `BASESTACK_FUNCTION_TOKEN` | Optional bearer session for a private CLI function run |
| `BASESTACK_TEST_DATABASE_URL` | Disposable integration server with CREATEDB permission |

See [Auth](AUTH.md) for password and challenge settings. Test mode supports an
injected delivery provider; it deliberately does not enable development outbox
files. Enabled Auth requires a delivery provider even if verification is disabled.

The first local service start creates a random 256-bit password in
`.basestack/local.env`; it never replaces an existing password or volume. Local
credentials remain fallback values for compatibility, so inject an explicit
production DSN rather than relying on a developer machine's files. Changing a
credential file does not rotate the password of an initialized PostgreSQL volume.
Profiles select configuration; they do not create isolated databases or Compose
volumes. Use distinct database URLs for development, test and production.
Remote connections should use PostgreSQL TLS verification. Container administrators
can inspect container environment. Compose's raw output is never relayed by the CLI.

## Database and migrations

PostgreSQL is ordinary PostgreSQL: use SQL, pgx, psql, pg_dump and standard backups.
No ORM or public raw-SQL endpoint is introduced. The database provider boundary
is `migrations.Provider`/`Service`; it owns connection, health, migration semantics
and close. The bundled implementation exposes its pgx pool to PostgreSQL-specific
Auth/RBAC. A different database provider must also replace those repositories;
changing a provider name alone cannot translate PostgreSQL SQL.

```sh
basestack db create-migration create_notes
# Edit basestack/migrations/000004_create_notes.sql in a fresh project.
# The existing alias, basestack migration new create_notes, still works.
basestack db status
basestack db migrate
basestack db rollback
```

SQL names use six-digit positive sequences and lowercase underscore names.
Discovery rejects duplicate versions/names, malformed SQL filenames and symlinks.
Applied version, name and SHA-256 checksums are stored in
`basestack_internal.schema_migrations`. A PostgreSQL transaction advisory lock
serializes migration, rollback and status runners. A pending batch and its
metadata commit together or roll back together. Status never creates metadata.
Applied history must match a prefix of the local files exactly. Creation uses an
exclusive lock/file and never overwrites historical SQL.

For explicit rollback, write a companion such as
`000004_create_notes.down.sql` **before applying** its forward file:

```sql
-- 000004_create_notes.sql
CREATE TABLE notes (id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, body text NOT NULL);
```

```sql
-- 000004_create_notes.down.sql
DROP TABLE notes;
```

Rollback executes only the latest applied migration's recorded down SQL and
removes its history row in one transaction. Down SQL is checksummed too; adding
or changing it after application is rejected. A migration without down SQL
cannot be rolled back by the CLI. The shipped identity migrations deliberately
have no down files. Rollback runs owner-written SQL and can destroy data;
review it and maintain backups. It is not automatic disaster recovery.

Both directions forbid transaction-control statements. Function-body PL/pgSQL
blocks are supported; `CREATE INDEX CONCURRENTLY` is not supported inside this
transactional workflow. Migration operations have a five-minute deadline. Driver
errors omit credentials, SQL and data; safe SQLSTATE categories remain visible.

## Auth and authorization

See [Auth](AUTH.md) for signup, challenge delivery and the session contract.
New v3 projects have opaque bearer sessions, `POST /api/auth/logout` and
`GET /api/auth/me`. Sessions expire after 12 hours; password changes/resets and
account suspension invalidate them. The browser client keeps tokens in memory.

RBAC has users, named roles and exact permissions. `admin` and `member` are
initial empty roles; neither has implicit bypass privileges. Grant deliberately:

```sh
basestack roles create editor
basestack roles grant editor profile.read
basestack roles grant editor storage.files.read
basestack roles grant editor storage.files.write
basestack roles assign <user-uuid> editor
basestack roles check <user-uuid> profile.read
basestack roles revoke editor storage.files.write
basestack roles unassign <user-uuid> editor
```

Roles and grants persist in PostgreSQL. Checks read current assignments and active
user state; revocation has no stale cache. `rbac.Authorizer.Check` is also usable
inside server handlers. Management commands are **trusted operator operations**
with database credentials; there is no public role-management HTTP endpoint.
No user can self-assign roles. Organizations, tenants, role inheritance, wildcard
permissions and row-level policies are future work.

## Private local storage

```sh
basestack storage create-bucket files
basestack storage put files < ./document.pdf
# put returns generated ID, byte size and modification time.
basestack storage list files
basestack storage get files <object-id> > ./download.pdf
basestack storage delete files <object-id>
```

`storage.Store` supports bucket creation/listing and object upload/list/read/delete
with metadata. The local provider writes under `.basestack/storage/<bucket>/`.
Buckets are validated names; objects have random 256-bit hexadecimal IDs. A
caller-supplied filename never becomes a path. Complete uploads publish atomically
with exclusive links; failures/oversized uploads remove temporary bytes. Files
are 0600, directories 0700. Symlinks and traversal are rejected. Applications
can inject another store via `app.Options.Storage`, without changing HTTP clients.

HTTP uses `/api/storage/<bucket>` for GET listing and POST raw-byte upload;
`/api/storage/<bucket>/<id>` supports GET and DELETE. Every HTTP request needs a
session and the exact `storage.<bucket>.read` or `.write` grant. Downloads are
attachments with octet-stream content and nosniff. Bucket creation remains a
trusted CLI/server operation. There are no public buckets, signed URLs, multipart
uploads, custom metadata fields or S3 integration yet. Local metadata contains
ID/size/modification time; original filenames are deliberately not stored.

The local filesystem belongs to the application owner. Checks defend against
untrusted API paths and existing symlinks, not a hostile OS user concurrently
replacing the owner's directories. Use a private owned volume. Native Windows
permission/hard-link behavior is not runtime-verified; unsupported filesystems
fail cleanly. Reads/listings and trusted function code are not an OS sandbox.

## Compiled server functions

```sh
basestack functions list
basestack functions inspect hello
basestack functions run hello
printf '{"example":true}' | basestack functions run hello
```

Edit `internal/runtime/functions/example.go` in your generated application, or
supply definitions via `app.Options.Functions` in `cmd/server/main.go`. Definitions
are compiled Go with validated unique names, visibility and permission metadata.
`hello` is explicitly public; `whoami` requires `profile.read`. Discovery is a
sorted view of this compiled registry. No arbitrary shell scripts are executed.

Each handler receives a cancellation/deadline context, JSON body, verified user
(for private functions), and a private environment lookup. It returns a JSON
value or an error. Errors and panics produce generic responses without exposing
provider details. Input/output JSON are bounded to 1 MiB; request contexts have
an eight-second deadline. Trusted functions must honor cancellation; CPU-bound
code is not forcibly terminated or isolated. Handlers can capture application
providers in Go closures. Never return an environment value intended to stay secret.

POST `/api/functions/<name>` runs the same registry. Private functions require a
session and an exact permission. CLI execution runs generated code locally and
uses optional `BASESTACK_FUNCTION_TOKEN`, never a token CLI argument. Enabled
runtime dependencies must be available; list/inspect do not connect to the DB.
Function stdout is intentional application output, so function authors own its
privacy. Scheduler, queues, remote builds and serverless hosting are out of scope.

## Frontend client

`src/services.json` is an explicitly public schema v1 configuration containing
only `apiURL`; it imports no backend settings. Its schemaVersion is also required.
Use an HTTP(S) origin or `""` for same-origin hosting. Change it if your API port
changes. URLs with credentials, paths, queries or fragments are rejected.

```ts
import { base } from './services';
await base.auth.login(email, password);
const current = await base.auth.current();
const response = await base.functions.run('hello', {});
const uploaded = await base.storage.upload('files', new Blob(['hello']));
await base.auth.logout();
```

The typed client also provides signup, verification/reset helpers and storage
listing/download/delete. Bearer tokens stay in closure memory, with no cookies,
localStorage or sessionStorage; reload requires login. Requests refuse redirects
and omit ambient cookies. Errors expose only HTTP status, never server error
bodies. There is deliberately no `base.db.query` or public CRUD abstraction yet.
Existing composition rendering makes no service calls until application code
imports and uses this client.

## Verification and limits

`scripts/verify-generated.sh` exercises root/generated Go tests, race tests, vet,
real PostgreSQL migration/rollback/Auth/session/RBAC behavior, environment and
storage/function CLI workflows, live HTTP, frontend-only compatibility, frontend
tests/typecheck/build, shutdown and database persistence. CI runs the same script.
Without `BASESTACK_TEST_DATABASE_URL`, Go tests explicitly skip database integration.

TLS termination, backups, encrypted secret providers, email delivery, upload malware
scanning and deployment remain operator concerns. Auth abuse controls are local to
one process. There is no refresh-token rotation, multi-tenant policy, ORM, cloud
provider, realtime system or Studio in this release. See [architecture](ARCHITECTURE.md).
