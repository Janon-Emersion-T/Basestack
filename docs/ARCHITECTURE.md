# BaseStack architecture

BaseStack is an independent application platform. Composition and application
services share inspectable, versioned files. Generated source, SQL and data belong
to the application owner. No hosted BaseStack service, BaaS or AI is required.

## Application Services — 0.3.2

The repository audit found an existing 0.3.1 Auth Core implementation. This release
extends it without rewriting the working 0.2 Composition Engine. See the
[audit record](APPLICATION-SERVICES-AUDIT.md) and [service guide](SERVICES.md).

| Package | Responsibility |
| --- | --- |
| `cmd/basestack`, `internal/cli` | Argument validation, composition/service commands, foreground process management |
| `internal/project` | Composition schema v2, validation, atomic writes and private-state ignore handling |
| `internal/registry`, `internal/strictjson` | Shared frontend catalog and exact/duplicate-free JSON parsing |
| `internal/runtime/config` | Services schemas v1/v2/v3, environment selection, private profile store |
| `internal/runtime/database` | PostgreSQL connection/pool and sanitized driver errors |
| `internal/runtime/migrations` | Database provider boundary, discovery, checksums, locking, apply/status/rollback |
| `internal/runtime/auth` | Identities, Argon2id, challenges, delivery, opaque sessions and HTTP |
| `internal/runtime/rbac` | Persistent roles, grants, assignments and exact permission checks |
| `internal/runtime/storage` | Provider interface, private local buckets, metadata and authorized HTTP |
| `internal/runtime/functions` | Compiled registry, metadata, execution context, authorized HTTP |
| `internal/runtime/privatefs` | Private directory/file checks shared by new local state providers |
| `internal/runtime/server` | HTTP routing, CORS, response headers, lifecycle and shutdown |
| `internal/runtime/app` | Configuration, provider wiring, runtime commands and resource ownership |
| `internal/services` | Optional local PostgreSQL Compose lifecycle and private credentials |
| `internal/scaffold` | Exclusive project creation; embeds tested runtime and frontend sources |

This is a modular monolith. Functions/storage/Auth do not require separate
microservices. The dependency direction is CLI → project/scaffold/services;
app → runtime modules; storage/functions HTTP → Auth session interface and RBAC
interface; PostgreSQL adapters → pgx. Runtime modules do not import the CLI or
React. Generated sources have their module path rewritten and run independently.
Dependencies/checksums are embedded and checked for drift.

`app.Options` allows injecting database, storage and secret providers, delivery, and
function definitions. The database provider owns health, migration semantics and
close; built-in Auth/RBAC explicitly need its PostgreSQL pool. Other database
engines will require matching repositories, not a misleading configuration alias.
`auth.Sessions`, `rbac.Authorizer`, `storage.Store` and `config.SecretStore` describe
real consumer boundaries, without creating an interface for every implementation.

## Versioned configuration boundaries

Composition remains schema v2 (`name`, `theme`, `pages`, ordered typed sections).
The browser never reads `basestack/services.json` or private environment profiles.
Services v1 is database/API only; v2 retains Auth Core's credential-only behavior;
v3 adds session behavior, provider names, authorization, storage and functions.
Unknown new fields in legacy versions are rejected, including null fields.
Projects without services keep their frontend workflow. No automatic source or
historical SQL rewriting occurs.

Private profile schema v1 stores bounded string values under ignored `.basestack/`.
Process variables override profiles, dotenv and generated local credentials.
Defaults are production-safe; local email delivery requires explicit development.
Public browser settings use their own tiny `src/services.json` schema v1. The
client exposes Auth/storage/function HTTP operations; it never exposes SQL or
internal Go provider types.

## Lifecycle and security boundaries

Startup validates configuration, constructs providers and verifies enabled DB/Auth
state before opening the listener. The runtime owns pool shutdown. HTTP has bounded
timeouts and context cancellation. CLI runtime commands build the application's
own source into unique private temporary directories and run foreground children;
cancellation signals the owned process tree and waits with a bounded fallback.
Docker manages only the optional local PostgreSQL instance and never deletes its
volume on stop. Containers can be replaced with externally managed PostgreSQL.

Credentials use established Argon2id and cryptographic randomness. Session tokens
are opaque, database-backed and stored only as digests. Protected routes resolve
a session and then check live exact permissions. Roles have no special-name bypass.
Role and local storage CLI operations trust the operator's filesystem/database
access; there are no unauthenticated role-management APIs. Generated object IDs,
private directories, atomic publication, size limits and path checks protect local
storage. Private profiles use exclusive locks and atomic writes.

Compiled function code is trusted application source. It receives context,
validated routing metadata, JSON input and private environment access, and must
honor cancellation and avoid returning secrets. It is not a sandbox or remote
serverless platform. Generic boundary errors omit secrets; Compose logs are not
relayed. All deployed services still need TLS, private volumes, backup/retention
policy and reviewed production delivery. See [Auth](AUTH.md) for residual limits.

## Composition preserved

Six component types, twelve variants and three themes use the same portable catalog
and React renderer as 0.2. Page links use full navigation and need static-host
fallback to index.html. Generated frontend tests verify schema rules, rendering,
escaping and navigation. Service client imports are opt-in; composition rendering
does not make new network calls.

## Verification and next scope

CI runs `scripts/verify-generated.sh`: root/generated Go tests/race/vet, real
PostgreSQL migrations and rollback, Auth/session/RBAC, live API/CLI storage and
functions, private environments, frontend-only compatibility, frontend tests,
typechecking, production build, shutdown and volume persistence. Integration tests
use randomly named disposable databases and remove only their owned test databases.
CI publishes CLI/frontend build artifacts, never secrets, logs or database files.

0.4 should focus on integration and operational polish: usable Auth/application UI,
provider conformance tests, encrypted secret adapters, session rotation and retention,
reviewable source upgrades, and cross-platform verification. Studio can build on
these same schemas when that foundation is ready. Cloud hosting, billing, an ORM,
queues, realtime subscriptions, multi-tenancy and a marketplace remain separate
milestones rather than prerequisites for a complete local application.
