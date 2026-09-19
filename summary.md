# Issue 3: Goal dependency graph

Implemented manual goal status and ordered prerequisite edges with migration
`003_goal_dependencies.sql`. Existing goals use the proposed date/checked-step
backfill (including vacuously complete goals with no steps). Older archives without
these fields import as planned with no prerequisites. Explicit status and dependency
order round-trip exactly, including forward references.

Graph saves serialize inside their transaction, reject self-edges and cycles with
400 responses, and preserve optimistic revisions. Deletion cascades edges and bumps
dependent revisions. The editor offers status, cycle-safe prerequisite selection,
date/dropped warnings, manual status suggestions, and named dependent confirmation.
`/goals/graph` uses dagre with readiness/blocker badges, conflict edges, finished-goal
filtering, critical paths, and links into the editor. All goals are loaded, including
prerequisites outside the timeline window. Read-only MCP/context includes the fields
and explains derived readiness versus explicit completion.

Validation passed:
- Full PostgreSQL `go test -race ./...` using disposable schemas: cycle rejection,
  concurrent writers, SQL self-edge CHECK, cascade/revision invalidation, exact
  transfer, backfill, and missing payload handling.
- `npm test`: 33 tests, including dependency state, cycles, suggestions and
  hand-computed inclusive critical paths. MCP suite: 12 tests.
- Production build, TypeScript, Astro check (zero errors/warnings; existing hints),
  and `git diff --check`.
- Real Brave/CDP browser acceptance: form-created edges, disabled cyclic options,
  API 400 cycle/self-edge responses, graph/conflicts, critical target, finished
  toggle, dropped-prerequisite readiness/warning, editor links, and named deletion
  confirmation followed by cascade verification. Test lives in
  `tests/goal-dependencies.browser.mjs`; run against an initially empty disposable
  authenticated server (defaults: port 4337 and existing PWA test credentials).

Independent worktree/branch from main. Migration 003 is reserved; shared migration
runner discovers numbered gaps and later supports 001+002+003. Files overlapping
issue 2 need normal integration when combining branches; no issue 2 features were
copied. No commits, pushes, deployments, or shared-public-schema migrations performed.
Native mobile multi-select interaction and very large graph performance were not
benchmarked; desktop browser acceptance used the built application and isolated DB.
