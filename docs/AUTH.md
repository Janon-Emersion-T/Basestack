# Auth Core — BaseStack 0.3.1

Auth is a module in the editable Go modular monolith. It provides accounts, password credential verification, email verification and password reset. **Successful login does not establish a session or authorize subsequent requests.**

## Architecture and ownership

`internal/runtime/auth/http.go` handles transport, strict JSON and rate limits. `service.go` coordinates validation, hashing, delivery and account policy. `repository.go` owns parameterized PostgreSQL queries and transactional transitions. Passwords, challenges, public models and delivery have focused source files. `app.RunWithDelivery` wires the module into the existing server. There is no separate Auth deployment, identity vendor or proprietary database interface.

Generated projects receive editable source, tests, pinned Go dependencies and SQL. CLI upgrades never replace this source. The composition schema and frontend registry remain unchanged; functional Auth components belong to 0.3.5.

### Database

Fresh projects apply the unchanged `000001_initial.sql`, then `000002_auth_core.sql`. Its authoritative source is `internal/runtime/auth/schema.sql`, embedded and copied by the scaffold. The existing migration runner preserves checksums, ordered history, advisory locking and transactional application.

| Table in `basestack_auth` | Contents |
| --- | --- |
| `users` | Random UUIDv4 ID; email and unique normalized email; encoded password hash; explicit status; verification, creation, update, last-login and password-change timestamps |
| `challenges` | SHA-256 token digest; user foreign key; purpose; expiry; consumed timestamp |
| `events` | Event ID, nullable user ID, event type and timestamp |

Email normalization trims surrounding whitespace and lowercases ASCII addresses. It does not remove dots or plus tags. Internationalized addresses and display-name syntax are currently rejected. PostgreSQL enforces normalized uniqueness with C collation and additional value/status constraints; concurrent signup cannot create duplicate accounts. Timestamps use `timestamptz`.

States are `unverified`, `active`, `suspended` and `disabled`. Required verification starts accounts unverified; successful verification activates eligible accounts. With verification explicitly disabled, signup creates active accounts but does not claim their email is verified. Suspended/disabled accounts cannot log in or redeem challenges. Disabling is terminal in the current service API. Internal find, suspend, disable and password-update operations are for trusted application code; no administrative HTTP route is exposed.

UUID identity is independent of email. Later identity tables can reference this stable user ID; moving password credentials to an identity table can use an additive migration/backfill. OAuth, MFA and identity-linking tables are deliberately deferred.

For an existing 0.3.0 project, merge reviewed source/dependency/configuration changes. Add the Auth SQL at the **next unused migration sequence** if 000002 is occupied. Never overwrite, rename or reformat applied migrations. Disabling Auth removes routes, not database contents.

## HTTP contract

All routes below use POST and `Content-Type: application/json`. JSON is strict: unknown/case-mismatched fields, duplicates, trailing values and malformed payloads fail. Bodies are limited to 8192 bytes. Secrets belong in request bodies, never URLs. Use HTTPS outside loopback development.

| Route | Body | Success |
| --- | --- | --- |
| `/api/auth/signup` | `email`, `password` | 201, `data.user` |
| `/api/auth/login` | `email`, `password` | 200, `data.credentialsVerified: true`, `data.sessionIssued: false`, `data.user` |
| `/api/auth/verify-email` | `token` | 200, `data.user` |
| `/api/auth/verify-email/request` | `email` | 202, generic `data.message` |
| `/api/auth/password/forgot` | `email` | 202, identical generic `data.message` for existing/missing/ineligible accounts |
| `/api/auth/password/reset` | `token`, `password` | 200, `data.passwordChanged: true`, `data.sessionIssued: false` |

The explicit public user contains only `id`, `email`, `emailVerified`, `status`, `createdAt`. Internal user fields are excluded from JSON serialization; mapping to the public model is explicit. No endpoint returns a hash, raw challenge, cookie, session or access token.

Errors use `{"error":{"code":"invalid_credentials","message":"Credentials could not be verified."}}`. Wrong credentials, missing accounts, unverified accounts and suspended/disabled accounts share that 401 error. Invalid, expired, wrong-purpose and consumed challenges share 400 `token_invalid`. Other codes include 400 `invalid_request`, 409 `signup_unavailable`, 413 `request_too_large`, 415 `unsupported_media_type`, 429 `rate_limited`, and 503 `auth_busy`/`auth_unavailable`. Errors omit raw Go/SQL/driver details. Rate/busy responses include `Retry-After`.

**Signup privacy limitation:** new signup returns a public user with 201; duplicate signup returns 409. This contract exposes address availability. Rate limiting reduces abuse but does not remove that disclosure. Applications requiring private account membership need a different asynchronous signup contract.

## Passwords and credential verification

Argon2id uses the maintained `golang.org/x/crypto/argon2` implementation. Defaults are 64 MiB memory, three iterations and one lane, with a fresh cryptographically random 16-byte salt and 32-byte result. The PHC encoding stores algorithm/version, parameters, salt and result. Verification bounds encoded input and resource parameters before allocation and compares results in constant time. Successful login rehashes when configured parameters differ, inside a transaction that rechecks the previous hash and account state.

Passwords allow passphrases: at least 15 Unicode characters, at most 1024 bytes; no composition rules or silent trimming. Supported stored/configured parameters are 19–256 MiB memory, 2–6 iterations and 1–4 lanes. Hashes outside these bounds are rejected. A service permits at most two concurrent Argon2 operations, returning busy rather than accumulating an unbounded queue. Argon2 work already running cannot be interrupted by request cancellation.

Missing-account login verifies a dummy hash to reduce timing differences. This is not a strict constant-time network guarantee. Defaults follow the [OWASP password storage guidance](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html) and passphrase policy follows the [OWASP authentication guidance](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html). Benchmark configuration on deployment hardware within the enforced bounds.

## Verification and password reset

Tokens contain 32 cryptographically random bytes, encoded as unpadded base64url. Only their SHA-256 digest is stored in PostgreSQL. Purpose binding prevents a reset challenge from verifying an email. Verification expires after 24 hours by default; reset after 30 minutes. Database time determines validity on redemption.

Issuing a replacement invalidates the previous open challenge for that purpose. User-row locks and transactions make redemption single-use even under concurrent requests. Verification updates verification time/status and consumes its challenge together. Reset updates the password hash and `password_changed_at`, consumes reset challenges and records its event together. Suspension/disabling also invalidates open challenges. Password reset does not verify an email or reactivate an account.

`password_changed_at` and the service transaction boundary give 0.3.2 a place to attach real session invalidation. No session invalidation is claimed now.

Forgot/resend return the same accepted response even if delivery or repository work fails, apart from input validation. A 300 ms response floor reduces simple timing differences, but slower synchronous delivery can still differ. Delivery occurs after commit; failure records a safe event, and verification resend allows recovery. There is no durable delivery queue, retry worker or delivery-success guarantee. A future provider should use bounded calls and a reviewed queue/timing policy. These choices and their limits are informed by [OWASP forgot-password guidance](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html).

## Configuration and delivery

Fresh `basestack/services.json` uses services schema v2 with:

```json
"auth": {"enabled": true, "requireEmailVerification": true}
```

This is a field within the full services object. Auth requires enabled PostgreSQL. Legacy services schema v1 remains supported with Auth absent; adding even `auth: null` to v1 is rejected. Browser-facing `basestack.json` remains composition schema v2 and contains no backend secrets.

Environment precedence is process variables, `.env`, `.basestack/local.env`, defaults. Values are literal; see [services](SERVICES.md).

| Variable | Default / allowed values |
| --- | --- |
| `BASESTACK_ENV` | `production`; only `production` or `development` |
| `BASESTACK_AUTH_DELIVERY` | `none`; `none`, `local`, `external` |
| `BASESTACK_AUTH_ARGON2_MEMORY_KIB` | 65536; 19456–262144 |
| `BASESTACK_AUTH_ARGON2_ITERATIONS` | 3; 2–6 |
| `BASESTACK_AUTH_ARGON2_PARALLELISM` | 1; 1–4 |
| `BASESTACK_AUTH_VERIFY_TTL_SECONDS` | 86400; 1–86400 |
| `BASESTACK_AUTH_RESET_TTL_SECONDS` | 1800; 1–3600 |

Short TTLs are useful for tests; use appropriate delivery windows in real applications. No provider credentials are generated or committed.

### Explicit local workflow

1. Generate a project and copy `.env.example` to `.env`.
2. Uncomment `BASESTACK_ENV=development` and `BASESTACK_AUTH_DELIVERY=local`.
3. Run `basestack services start`, `basestack db migrate`, `basestack auth status`, then `basestack api`.
4. Submit JSON signup to the API. Read the resulting private message in `.basestack/auth-outbox/` using a local editor; send its token in the verification request body.
5. Forgot-password creates a reset message in the same outbox. Submit that token and a new password to reset.

The outbox stores raw delivery messages, including recipient and token, exclusively for local development. It is ignored by Git, is not served over HTTP and never prints secrets to normal logs. Directories require 0700 and files use exclusive creation with 0600; unsafe permissions/symlinks fail closed. Remove old messages when no longer needed and exclude local state from published artifacts/backups. POSIX permissions were tested on Linux; native Windows outbox permissions have not been runtime-verified and may fail closed.

### Production provider boundary

`auth.Delivery` implements `Deliver(context.Context, Message) error` and `DevelopmentOnly() bool`. Owners supply a reviewed provider through `app.RunWithDelivery` in their generated runtime entry point. Setting `external` alone does not install a provider. There is no bundled SMTP/vendor integration. An enabled Auth server with no provider refuses startup, even if verification is optional, because reset still needs delivery.

Production rejects `local` configuration and injected development-only providers. Providers must accurately declare their safety, protect message secrets, honor contexts, avoid credential logging and use a trusted configured destination/link origin. The application owner is responsible for deployment secrets, TLS, database privileges/backups and provider delivery. `auth status` reports configuration, not proof of provider reachability or database readiness.

## Abuse protection and audit events

The replaceable `Limiter` interface currently uses a mutex-protected fixed 60-second window per remote IP and route. Login and verification allow 10 attempts; signup, forgot, reset and resend allow 5. The map is bounded at 10,000 keys and rejects new keys when full until expiry. It does not key limits by account existence. Forwarded headers are ignored to avoid spoofing.

Limits reset on process restart and are not shared between instances. Clients behind a proxy/NAT share its IP quota; distributed attackers can use multiple IPs. Multi-instance deployment needs an external enforcement strategy or distributed limiter and an explicit trusted-proxy policy. No Redis is introduced.

Events cover `user_created`, `email_verified`, `login_succeeded`, `login_failed`, `password_reset_requested`, `password_reset_completed`, `user_suspended`, `user_disabled`, plus `password_changed`, `verification_requested`, `delivery_failed`. State-changing events commit with their database transition. Unknown-account events have no user ID. Events contain no password, hash, raw token, email address or IP. They are minimal security records, not comprehensive HTTP access logging; validation/rate-limit failures need not create database events. Retention/pruning of events, consumed challenges and local outbox files remains owner-managed.

## Security review and verification

The implementation review covered cryptographic randomness, bounded Argon2 parsing/concurrency, public serialization, parameterized SQL, normalized database uniqueness, transactional lifecycle checks, challenge purpose/expiry/reuse, generic credential errors, body limits, rate limits, delivery isolation and secret-free runtime logging. Automated tests cover concurrent duplicate signup, concurrent verification/reset redemption, password-change/login state checks, actual PostgreSQL constraints and safe audit values.

`scripts/verify-generated.sh` provisions a fresh app and disposable PostgreSQL, runs root and generated Go tests/race/vet with real integration tests, all frontend tests/typechecking/build, and live API signup/verification/login/forgot/reset flows. The live smoke test checks invalid, expired and reused tokens and scans responses/logs for its actual secret values. Repository review also checks diffs and logging patterns for accidental secrets/artifacts. This is an engineering review, not an independent penetration test or security certification.

Remaining limits include signup availability disclosure, imperfect timing uniformity, synchronous non-durable delivery, single-process abuse controls, owner-managed retention and untested native Windows outbox behavior. Internal administrative methods require a future authorization boundary before public exposure. CORS is an explicit-origin browser policy, not authorization.

## Milestone boundary

0.3.1 does **not** provide persistent authenticated sessions, access tokens, refresh tokens, OAuth/social login, MFA, OTP login, magic links, OIDC, SSO/SAML, passkeys, RBAC, anonymous accounts, identity linking or Auth UI. There are no placeholder endpoints claiming these capabilities.

**Next: 0.3.2 — Sessions, Tokens & Auth Security.** Design and implement real session issuance after credential verification, transport/storage policy, expiration, rotation/replay protection, revocation, logout, password-change invalidation and the corresponding security/integration tests. RBAC remains 0.3.3, Data/CRUD 0.3.4 and Auth UI Integration 0.3.5.
