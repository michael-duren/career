# Homelab database operations

## Step 6: application release

Work in `/home/mduren/Code/career-strategy/migration-step-6`. The Astro skill informed retaining the existing static output and copying only `dist/` into the runtime. The allowlisted Docker context excludes seed content, local credentials, and private archives. Node is build-only.

Observed 2026-09-14: kubeconfig `/home/mduren/Code/home-infra/infra/ansible/kubeconfig-homelab`, context `default`, API `https://192.168.20.3:6443`; Ready amd64 nodes `k8s-cp-1=192.168.20.3`, `k8s-w-1=192.168.20.101`, `k8s-w-2=192.168.20.102`. TCP 5432 on `192.168.20.103` accepted a connection. This does not prove authenticated DB/schema access. No existing ingress claims career.homelab; DNS allocation still needs confirmation.

### First validation handoff

Run `./scripts/deploy-homelab.sh check`. Prerequisites: Node 22.12+ (or supported newer Node), npm, Go 1.27, and the existing migrated local development PostgreSQL at 127.0.0.1:5433. This installs locked dependencies, runs Astro/TypeScript/domain/Go checks, builds static output and Go, and runs HTTP integration against that local database. It creates/removes disposable test entities. Return the final PASS line and exit status, or the first failing check. Do not supply production database credentials.

Run `./scripts/deploy-homelab.sh check-postgres` against the same disposable development database for PostgreSQL race coverage. Expected exit 0, no skipped database suite. Builds and suites are user-run under migration.md's two-second rule.

### Build and release phases

After reviewing and committing the topic worktree, run these script phases in order. Each invocation uses the working directory above. The image tag is `career-strategy:<full-Git-commit>`; builds/releases reject a dirty worktree. Do not commit on the migration branch as a substitute. A locally existing tag is reused, never overwritten.

| Invocation | Prerequisites / expected result |
| --- | --- |
| `./scripts/deploy-homelab.sh inspect` | Homelab kubeconfig; confirms cluster endpoint, node addresses and eligibility. |
| `./scripts/deploy-homelab.sh build` | Docker with network access; builds linux/amd64 image and checks non-root runtime inventory; prints image tag and PASS. |
| `./scripts/check-container.sh` | Built image, Docker, curl, free local port 18089; verifies unavailable DB gives readiness 503 while health stays 200, then SIGTERM exit 0. |
| `./scripts/deploy-homelab.sh secrets` | Completed step 5 DB provisioning and npm dependencies; securely prompts for DB password and browser credentials, generates session secret. Only career namespace is affected. |
| `./scripts/deploy-homelab.sh validate` | Namespace exists; server-side dry-run must succeed. |
| `./scripts/deploy-homelab.sh deploy` | Docker image, SSH ops access and passwordless sudo k3s on every discovered eligible node; imports images, runs migration Job, waits for rollout. No data import or DNS switch. |
| `./scripts/deploy-homelab.sh logs` | Deployed workload; inspect locally and redact before sharing. |

Secret values are never command arguments or tracked YAML. Restricted temporary files are removed on exit; server-side apply avoids embedding a last-applied copy. Re-running secrets rotates JWT credentials and expires existing sessions after rollout. ConfigMap holds only nonsecret settings. No app RBAC, service account token, or NetworkPolicy is added.

Keep the local current and previous images; do not prune them from Docker or node containerd during the rollback window. Before uncordoning a new eligible amd64 node, run `./scripts/deploy-homelab.sh sync-images` from the current release checkout. It imports current and previous images on all eligible nodes and fails if a required local image is missing. Different-architecture nodes are excluded by the matching Deployment and Job selectors. Node eligibility is rediscovered on each release.

### Routing and HTTPS acceptance

Reserve career.homelab in LAN DNS or a local hosts file pointing at a verified Traefik node address. The LAN HTTP ingress is for reachability diagnostics: production cookies are Secure and PUBLIC_ORIGIN is https://career.michaelduren.com, so authenticated use requires that HTTPS origin.

Use a dedicated remotely managed Cloudflare tunnel for this app. Configure its public hostname route to `http://career.career-strategy.svc.cluster.local:80`, preserve the public Host, use HTTPS at the edge, redirect HTTP to HTTPS, and disable cache for HTML/API/auth responses. The tunnel and Kubernetes Secret are career-specific; do not reuse Rubber Duck's token. Select a released cloudflared digest from the official image publisher and record/test it before claiming 6.7 complete. `./scripts/deploy-homelab.sh tunnel` prompts for that digest and token; the checked-in placeholder is intentionally not deployable directly.

Do not activate the final production hostname route until step 8 authorizes cutover. Step 7 can use a separately allocated staging origin and DB, with matching PUBLIC_ORIGIN. A tested digest, provider route configuration, DNS allocation, and external acceptance are still pending; no tunnel or public routing was changed by this implementation.

Through the actual HTTPS path, acceptance must prove login sets Secure/HttpOnly/SameSite cookies; private HTML/API are no-store; unauthenticated HTML redirects to login and API returns 401; mismatched Origin mutations return 403; authenticated disposable note create/read/delete succeeds; logout expires the session. Record host, image, tunnel digest and results without cookies/passwords. The existing local integration script does not count as this evidence.

### Outage, rollback and backup acceptance

The container outage check above proves startup liveness independent of DB access and graceful shutdown. In step 7 staging, also interrupt and restore DB access for only the staging workload; observe readiness fail/recover, unchanged restart count, and successful saved reads after recovery. Do not stop the shared database VM. Deployed outage, rescheduling and HTTPS acceptance still require a staging validation script once step 7 allocates that environment.

`./scripts/deploy-homelab.sh rollback` restores the recorded previous image after importing it on all eligible nodes. It does not reverse schema migrations or restore data; use only when the previous binary is compatible with the migrated schema. The release records the previous image before rollout so failures retain rollback information. On first release there is no previous image. Preserve step 8's distinct Netlify/data reconciliation procedure after any Go writes.

Use the step 5 wrapper's `pod-check`, `verify-backup`, and `restore` phases described above. Backup restoration must target the disposable restore database, never the live app DB. Complete populated-restore and pre-cutover backup evidence under 5.6. The runtime lacks psql and a writable import archive directory; perform diagnostic/import tooling from the prepared step 5/7 workflow, not by installing tools in the app container.

Implementation references: [Docker multi-stage builds](https://docs.docker.com/build/building/multi-stage/) and [Kubernetes probe semantics](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/). Runtime/rollout results remain unverified until the user runs the phases.


These procedures affect only PostgreSQL database/login `career_strategy`; they do not run general PostgreSQL provisioning or modify `duckserver`.

The commands below are also packaged in `deploy/homelab/run-step5.sh`. Run its phases in order:

```sh
./deploy/homelab/run-step5.sh prepare
./deploy/homelab/run-step5.sh database
./deploy/homelab/run-step5.sh backup
```

The `pod-check` phase requires `POD_NAME`; the `restore` phase requires `BACKUP_ARCHIVE`. Both are intentionally separate because the pod and a populated backup do not exist during initial provisioning.

If provisioning completed but its final status display returned code 3 for an inactive successful oneshot service, run only `./deploy/homelab/run-step5.sh verify-backup`.

## Prerequisites and secret

The wrapper installs `community.postgresql` and `ansible.posix`. During the database phase it prompts invisibly for a password of at least 20 characters, exports it only to child Ansible processes, and unsets it on exit. Store that password using your existing password-management workflow; no Ansible Vault is required.

```sh
cd /home/mduren/Code/career-strategy/migration-step-5
./deploy/homelab/run-step5.sh prepare
./deploy/homelab/run-step5.sh database
```

The password is never placed on the command line or written into either repository. At deployment, inject the same value into k3s through step 6's Secret workflow.

## Provisioning and TLS

The playbook discovers the live HBA path, adds a database/user-specific SCRAM rule for the cluster SNAT subnet `192.168.20.0/24`, and reloads PostgreSQL only when needed. It reports non-secret HBA and TLS settings.

The chosen initial LAN mode is `sslmode=disable`, matching Rubber Duck, until live inspection proves a trusted server certificate and CA exist. Do not require `verify-full` until the certificate names the database host and its CA is mounted in the app pod.

```sh
cd /home/mduren/Code/home-infra/infra/ansible
ansible-playbook -i inventory.ini career-strategy-db.yml --syntax-check
make career-strategy-db ANSIBLE_ARGS=--check
make career-strategy-db
make career-strategy-db
```

The second apply must report `changed=0 failed=0`. Record the reported `hba_file` and TLS values in `migration.md`. After step 6 creates an app pod, verify the real source address from it (supply the password interactively):

```sh
kubectl -n career-strategy exec -it POD_NAME -- sh
psql "postgres://career_strategy@192.168.20.103:5432/career_strategy?sslmode=disable" -c 'select current_database(), current_user, inet_client_addr();'
```

Both names must be `career_strategy`; `inet_client_addr()` must fall within the recorded HBA subnet.

## Backups and monitoring

`career-strategy-backup.yml` installs `career-strategy-db-backup.timer` on `pve-worker`. Daily at 03:50 plus jitter, it pulls `pg_dump -Fc`, verifies the archive TOC, atomically stores `/mnt/duckbackup/pgdump/career-strategy/career_strategy-*.dump`, and prunes only those files after 90 days. Duck archives remain untouched.

The script exposes success and last-success time through node exporter's textfile collector. Prometheus scrapes `pve-worker:9100` and loads failure/staleness rules. This makes failures visible in Prometheus; notification routing remains part of the broader Alertmanager workstream.

```sh
cd /home/mduren/Code/home-infra/infra/ansible
ansible-playbook -i inventory.ini career-strategy-backup.yml --syntax-check
ansible-playbook -i inventory.ini career-strategy-backup.yml --check
make career-strategy-backup
make career-strategy-backup
/home/mduren/Code/career-strategy/migration-step-5/deploy/homelab/run-step5.sh verify-backup
ansible prometheus -i inventory.ini -b -m ansible.builtin.command -a 'promtool check config /etc/prometheus/prometheus.yml'
```

Confirm capacity comfortably exceeds 90 projected dumps. The recovery point objective is daily backups and at most 24 hours of data loss outside cutover. Start the service immediately before cutover.

## Disposable restore drill

Recreate the role first with the database playbook if the DB host was rebuilt. Select a real archive; never target the live database:

```sh
cd /home/mduren/Code/home-infra/infra/ansible
ansible db -i inventory.ini -b --become-user postgres -m community.postgresql.postgresql_db -a 'name=career_strategy_restore_test owner=career_strategy state=present'
ssh root@192.168.20.100 'cat /mnt/duckbackup/pgdump/career-strategy/career_strategy-YYYY-MM-DD_HHMM.dump' | ssh ops@192.168.20.103 'sudo -n -u postgres pg_restore --exit-on-error --no-owner --role=career_strategy -d career_strategy_restore_test'
```

Compare every table count, inspect representative note/company/document/journal/goal rows, and run read-only application checks with a temporary restore-database URL. Record results, then remove only the disposable database:

```sh
ansible db -i inventory.ini -b --become-user postgres -m community.postgresql.postgresql_db -a 'name=career_strategy_restore_test state=absent force=true'
```

Acceptance evidence includes archive verification, counts, representative records, application reads, Prometheus metric/rules, capacity, and the pre-cutover backup timestamp.
