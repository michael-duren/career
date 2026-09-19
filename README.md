# Project

Career website for users to track goals, books, companies, etc.

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

## Hosting

Production runs on the homelab k3s cluster: one Go service serves the static Astro
build and the `/api` routes, backed by PostgreSQL. Netlify is no longer used.
Deploy with `scripts/deploy-homelab.sh` (build, import images, migrate, and wait for
rollout). Kubernetes manifests live in `~/Code/home-infra/k8s/apps/career-strategy`;
the default kubeconfig is `~/Code/home-infra/ansible/kubeconfig-homelab`.
Override `HOME_INFRA`, `CAREER_MANIFESTS`, or `KUBECONFIG` as needed.
Use `scripts/deploy-homelab.sh rollback` to restore the previous deployment revision.

## Local development

Run `make postgres-up migrate check-db` to start PostgreSQL 17 and verify the schema.
See [local development](docs/migration/local-development.md) for Go API, seed,
import/export and testing commands. [migration.md](migration.md) is the historical
record of the Netlify → Go/PostgreSQL move.
