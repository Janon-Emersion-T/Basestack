# Working from your phone

1. Open the BaseStack repository in GitHub.
2. Browse `internal/cli/scaffold/src/sections.tsx` for templates or `docs/ARCHITECTURE.md` for the roadmap.
3. Use GitHub's file editor to make small changes and commit them. Template edits affect newly generated projects.
4. Open **Actions → BaseStack CI** and inspect the run for that commit. Green means CLI checks and the generated frontend build passed.
5. After a successful run, download `basestack-demo` to inspect or deploy the static build using your preferred hosting workflow. It must be served over HTTP; opening the HTML directly as a local phone file is not a supported preview.

GitHub's file editor edits source code; it does not run Go or npm on your phone. When a terminal environment is available, follow the root README. You do not need to manage a mobile VS Code setup to continue making commits and checking CI here.

BaseStack does not automatically deploy a public website or create cloud resources. CI artifacts provide build outputs only.
