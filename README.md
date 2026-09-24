# Project

Career website for users to track goals, books, companies, etc.

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

With Node.js, Go, and Docker installed:

```bash
make dev
```

`make dev` installs dependencies, sets up `.env`, starts PostgreSQL, builds the
frontend, applies migrations, and starts the Go server.

Open http://localhost:8080 and log in with `admin` / `password123` if you accepted
the local defaults. Setup generates a JWT secret and keeps existing `.env` files.
Use `npm run setup -- --defaults` to skip prompts, or answer `n` to choose login
credentials. The database defaults match the local Docker Compose stack.

Run `make help` to see all available commands.

## Hosting

Production runs on the homelab k3s cluster: one Go service serves the static Astro
build and the `/api` routes, backed by PostgreSQL. Netlify is no longer used.
Stable version tags publish a verified GHCR image and dispatch its immutable digest
to home-infra for GitOps promotion. See [release and rollback instructions](docs/releases.md)
for credential setup, first publication, and Argo adoption. Deployment templates live
in `~/Code/home-infra/k8s/apps/career-strategy`; released manifests live on its `gitops` branch.
`scripts/deploy-homelab.sh build` and `check` remain available locally; direct deployment
and rollback are disabled to protect GitOps ownership.

## Local development

Run `make postgres-up migrate check-db` to start PostgreSQL 17 and verify the schema.
See [local development](docs/migration/local-development.md) for Go API, seed,
import/export and testing commands. [migration.md](migration.md) is the historical
record of the Netlify → Go/PostgreSQL move.
