# Leetgrinder

Open `/leetgrinder` after signing in. The curriculum contains 84 numbered sessions across 12 weeks, with 252 core problems and 48 optional problems. Sessions follow your pace; there are no calendar deadlines.

Allow two hours per session: 90 minutes for attempts and debugging, and 30 minutes for reading and review. Optional problems are for sessions where core work finishes early. Finishing a day advances to the earliest unfinished session without marking its problems solved.

Record each attempt as solved, struggled, or unfinished, with minutes spent and whether you used a hint or reviewed a solution. A solve means the solution passed on LeetCode. The overview counts distinct solved problems and independent solves separately. History retains repeated attempts. Use “Correct this attempt” to fix an entry; concurrent corrections cannot silently replace one another.

Use “Export attempt history” to download Leetgrinder attempts and completed sessions as JSON. This is separate from the existing workspace archive. There is no Leetgrinder import UI yet; PostgreSQL backups remain the full restore mechanism.

## Development

The feature uses Go templ, not Astro or a client-side editor. Original lessons are authored in `internal/leetgrinder/lessons*.templ`; assignments and reading references live beside them. The private source files in `data/algomonster` are not required to build or serve the feature.

The templ generator and runtime are pinned together in `go.mod`. Run:

```sh
make generate
make build
make test
make test-postgres
```

`make dev` generates templates before starting Go. Docker builds generate them too. Generated `_templ.go` files remain ignored, so run `make generate` before invoking `go test ./...` directly in a fresh checkout. Air watches `.templ` files and excludes generated files from triggering rebuild loops.

Leetgrinder pages require the existing login. Writes use same-origin HTML forms with server validation. Failed attempt saves display the submitted draft for retry. Attempt IDs prevent duplicate creates; revisions protect corrections. Tracking belongs to the app's existing single configured account.

Future notification, automatic repetition, extension, and variable-time scheduling features are intentionally deferred. Stable problem slugs and attempt history provide the data those features will need.

See [sources and verification](leetgrinder-sources.md) for the reading bibliography and the command that checks all 300 assignments against LeetCode.

Migration 010 renames the existing tracking tables, indexes, and constraints to `leetgrinder` names while preserving saved attempts and completed days. Migration 009 keeps its historical SQL unchanged so previously applied checksums remain valid.

The shared local database already reserves migration 008 for note creation timestamps. Leetgrinder uses migrations 009 and 010. Run `make go-dev` to apply them; existing notes and learning history are retained.
