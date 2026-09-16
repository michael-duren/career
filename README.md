# Project github.com/michael-duren/career-strategy

One Paragraph of project description goes here

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

## MakeFile

Run build make command with tests
```bash
make all
```

Build the application
```bash
make build
```

Run the application
```bash
make run
```
Create DB container
```bash
make docker-run
```

Shutdown DB Container
```bash
make docker-down
```

DB Integrations Test:
```bash
make itest
```

Live reload the application:
```bash
make watch
```

Run the test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```

## Go/PostgreSQL migration development

Steps 1–2 of [migration.md](migration.md) have a local relational backend. Run
`make postgres-up migrate check-db` to start PostgreSQL 17 and verify the schema.
See [local development](docs/migration/local-development.md) for Go API, seed,
import/export and testing commands, and [captured migration inputs](docs/migration/current-behavior.md)
for the current browser contracts and production source. The existing Astro UI
continues to use Netlify until the browser/static-build migration steps are implemented.
