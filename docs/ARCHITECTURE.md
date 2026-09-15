# Architecture and V1 roadmap

## Product principle

BaseStack enables customers and developers to assemble applications without needing AI. The CLI and future drag-and-drop Studio share the versioned `basestack.json` application contract: pages contain ordered components with explicit variants and validated props. No AI or BaseStack server is involved in normal local development.

## Current implementation — 0.2 Composition Engine

| Location | Responsibility |
| --- | --- |
| `cmd/basestack/main.go` | Thin executable: call CLI, report errors, set exit status |
| `internal/cli/cli.go` | Dispatch, help and version |
| `internal/cli/{init,add,remove,page,theme,templates}.go` | Command-specific parsing, application changes and output |
| `internal/cli/options.go`, `frontend.go` | Strict options and shared dev/build npm invocation |
| `internal/project` | Schema v2 types, validation, page selection, strict reads and atomic writes |
| `internal/registry` | Embedded portable component definitions, property validation and theme tokens |
| `internal/scaffold` | Safe new-directory creation and embedded source copying |
| `internal/scaffold/template` | Independent React/TypeScript/Vite application and frontend tests |

Dependency direction is CLI → project/registry/scaffold; scaffold → project/registry; project → registry. The registry has no dependency on project or CLI. Go uses only the standard library. Dev/build share one file because their process invocation is identical.

### Composition contract

Schema v2 has `schemaVersion`, `name`, `theme` and `pages`. Each page has `id`, `path`, `title` and ordered `sections`. Each section has `id`, `type`, `variant` and `props`. IDs are page-scoped for sections; page IDs and paths are application-wide. The property schema belongs to the component definition. Nothing silently fills missing properties during reads.

The registry's JSON definitions are the source for variants, defaults, property requirements and theme tokens. `init` copies the exact embedded catalog into `src/registry.json`. Go tooling and generated JavaScript consume it; TypeScript uses a discriminated union for safe rendering. App structure validation exists in both runtimes, with corresponding negative tests. Changing the application schema requires updating both validators and types; it is not generated from a standards-based JSON Schema yet.

Registry packages can later supply this contract through a versioned loader. Remote package discovery, signing, dependency resolution and installation are not implemented. Adding a built-in component still requires a renderer and TypeScript type alongside its definition.

### Frontend

Six components each have two variants; shared CSS selectors produce distinct responsive layouts. Theme entries supply tokens for font, background, text, primary, secondary/muted, borders, radius, spacing and width. No external font service or routing dependency is used.

Routing matches `window.location.pathname` exactly. Standard anchors support normal keyboard navigation, browser back/forward, reloads and direct links. A missing route renders an explicit page list. Static hosts must fall back to `index.html`; subdirectory hosting, pre-rendering, server-side rendering and automated hosting configuration are outside 0.2.

Navigation can use explicit links or derive them from the page list. One hero per page supplies the h1; subsequent heroes use h2. Pages without heroes get a page-title h1. Links have visible focus, a skip link targets the main content, and decorative hero content is hidden from assistive technology. Tests cover rendered semantics and escaping, but do not constitute a complete accessibility audit.

### Filesystem safety and ownership

`init` creates the project directory exclusively and refuses existing directories, files or symlinks. Failed initialization cleans up only its newly created directory. Config mutations validate before writing, preserve permissions and replace through a temporary file and rename; non-regular config targets are rejected. Failed validation does not modify the manifest. CLI edits assume one writer at a time; no concurrent-editor locking is provided.

Generated projects are user-owned source snapshots. CLI upgrades never silently replace customizations. Projects can validate, test, develop and build using npm without Go or the CLI. Schema v1 is explicitly rejected; the documented manual migration preserves old projects until their source changes have been reviewed.

### Verification

Go tests exercise creation, existing-path protection, command usage, page/section lifecycles, deterministic selection, all variants/themes, strict JSON, nested props and atomic-write protection. Generated Node tests exercise registry validation, unsafe input, URL resolution, React rendering, escaping and heading/navigation semantics. CI builds a fresh multi-page application and checks its production artifact after frontend tests and TypeScript checking.

## Milestones toward V1

1. **0.1 foundation — complete:** Go CLI, six section types, single-page configuration, preview/build and initial tests/docs.
2. **0.2 composition — implemented:** multi-page schema, typed props, variants, token themes, registry, commands and expanded tests. Automatic v1 source migration is deferred.
3. **0.3 application services — next:** design and implement real Go HTTP services, PostgreSQL migrations, authentication/sessions, RBAC and CRUD modules, with integration tests and explicit schema/service versioning.
4. **0.4 Studio — planned:** visual editor using the same application schema, property/variant/theme editing, section ordering, preview and export.
5. **V1 release candidate — planned:** storage, deployment adapters, accessibility/security review, integration coverage and a safe upgrade/package versioning strategy.

Auth, RBAC, billing, licensing, PostgreSQL/database, storage, realtime, Studio and deployment are absent from 0.2. Contact is an email link and pricing is content; neither pretends to be a backend service. No production-readiness claim is made for future capabilities.
