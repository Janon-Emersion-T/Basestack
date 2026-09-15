# Architecture and V1 roadmap

## Product principle

Customers and developers should assemble applications using documented building blocks without needing AI. Templates cover entire applications, including navigation, content, marketing, commerce, authentication, dashboards and footers. A visual editor and a CLI should ultimately share the same versioned project schema.

## Current implementation

- `cmd/basestack`: executable entry point.
- `internal/cli`: commands, page operations, variant catalogue, validation, file operations and embedded templates.
- `internal/cli/scaffold`: independent React/TypeScript project copied during `init`.
- `basestack.json` in generated projects: ordered Home sections and optional additional pages.
- `src/config.ts` in generated projects: frontend parser with matching validation fixtures shared with Go tests.
- GitHub Actions: Go tests, race detection, frontend type checking, validation parity, demo production build and Chromium integration checks at desktop/mobile sizes.

The CLI never executes AI calls or downloads remote templates. `init` refuses existing directories; add/remove update the manifest through a temporary file and rename. CLI edits assume one writer at a time. Projects remain usable without the CLI, using npm. Templates are copied at creation: later CLI upgrades do not silently overwrite user customisations.

## Milestones toward V1

1. **0.1 foundation:** CLI, six section types, configuration, preview/build, tests and documentation.
2. **0.2 composition (implemented):** multiple pages, scoped section editing, two layouts per type, backward-compatible configuration, frontend/CLI validation fixtures and browser integration tests. Rich properties, a generated shared schema and theme tokens remain a follow-up.
3. **0.3 application services:** Go HTTP API, PostgreSQL migrations, authentication, sessions, RBAC and CRUD modules; functional services behind dashboard/auth templates.
4. **0.4 Studio:** React visual editor backed by the same schema, section ordering, property editing, preview and export.
5. **V1 release candidate:** storage, deployment adapters, integration coverage, accessibility/security review, versioning and upgrade strategy.

Future direction: Go for CLI and services; React/TypeScript for UI and Studio; PostgreSQL for persistence; Tailwind/design tokens once the shared component system grows. A DSL/compiler is not required for this product to work. Account login, billing and balance must only be added alongside real services, not simulated responses.

This repository's 0.2.0 milestone is not yet a full backend-as-a-service, authentication system, commerce engine or drag-and-drop editor. No production-readiness claim is made for those future capabilities.

## Routing decision

The initial page implementation selects a view using `?page=<slug>` and ordinary browser links. Static hosts serve the same index document; Vite uses relative asset URLs for hosting under subdirectories. Navigation reloads the document and supports refresh/back/forward without history-API rewrite configuration. Unknown views render a not-found message, though the HTTP response still comes from the static index. Prerendering, page-specific server metadata and clean-path routing are not included in this milestone.
