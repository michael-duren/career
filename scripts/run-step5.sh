#!/usr/bin/env bash
set -euo pipefail

ANSIBLE_DIR=/home/mduren/Code/home-infra/infra/ansible
SCRIPT_PATH=$(readlink -f "$0")
usage() {
  echo "Usage: $0 {prepare|database|backup|verify-backup|pod-check|restore}" >&2
  echo "  prepare    Install the required Ansible collections" >&2
  echo "  database   Syntax/check/apply the DB playbook twice" >&2
  echo "  backup     Syntax/check/apply backup twice, then verify it" >&2
  echo "  verify-backup  Verify backup, capacity, timer, and Prometheus only" >&2
  echo "  pod-check  Verify DB access from an existing app pod" >&2
  echo "  restore    Restore a selected archive into a disposable DB" >&2
  exit 2
}

cd "$ANSIBLE_DIR"

case "${1:-}" in
  prepare)
    ansible-galaxy collection install community.postgresql ansible.posix
    ;;

  database)
    read -r -s -p 'Career Strategy database password: ' CAREER_STRATEGY_DB_PASSWORD
    echo
    [[ ${#CAREER_STRATEGY_DB_PASSWORD} -ge 20 ]] || {
      unset CAREER_STRATEGY_DB_PASSWORD
      echo 'Password must contain at least 20 characters.' >&2
      exit 1
    }
    export CAREER_STRATEGY_DB_PASSWORD
    trap 'unset CAREER_STRATEGY_DB_PASSWORD' EXIT
    ansible-playbook -i inventory.ini career-strategy-db.yml --syntax-check
    make career-strategy-db ANSIBLE_ARGS=--check
    make career-strategy-db
    make career-strategy-db
    ;;

  backup)
    ansible-playbook -i inventory.ini career-strategy-backup.yml --syntax-check
    ansible-playbook -i inventory.ini career-strategy-backup.yml --check
    make career-strategy-backup
    make career-strategy-backup
    "$SCRIPT_PATH" verify-backup
    ;;

  verify-backup)
    ansible pve_worker -i inventory.ini -m ansible.builtin.shell -a '
      set -eu
      df -h /mnt/duckbackup
      systemctl start career-strategy-db-backup.service
      test "$(systemctl show career-strategy-db-backup.service -p Result --value)" = success
      test "$(systemctl show career-strategy-db-backup.service -p ExecMainStatus --value)" = 0
      systemctl show career-strategy-db-backup.service -p Result -p ExecMainStatus
      systemctl list-timers career-strategy-db-backup.timer --no-pager
      ls -lh /mnt/duckbackup/pgdump/career-strategy/
    '
    ansible prometheus -i inventory.ini -b -m ansible.builtin.command \
      -a 'promtool check config /etc/prometheus/prometheus.yml'
    ;;

  pod-check)
    pod_name=${POD_NAME:-}
    [[ -n "$pod_name" ]] || { echo "Set POD_NAME to the Career Strategy pod." >&2; exit 1; }
    kubectl -n career-strategy exec -it "$pod_name" -- \
      psql 'postgres://career_strategy@192.168.20.103:5432/career_strategy?sslmode=disable' \
      -c 'select current_database(), current_user, inet_client_addr();'
    ;;

  restore)
    archive=${BACKUP_ARCHIVE:-}
    [[ "$archive" =~ ^career_strategy-[0-9]{4}-[0-9]{2}-[0-9]{2}_[0-9]{4}\.dump$ ]] || {
      echo "Set BACKUP_ARCHIVE to a filename such as career_strategy-2026-09-14_0350.dump." >&2
      exit 1
    }
    ansible db -i inventory.ini -b --become-user postgres \
      -m community.postgresql.postgresql_db \
      -a 'name=career_strategy_restore_test owner=career_strategy state=present'
    ssh root@192.168.20.100 \
      "cat /mnt/duckbackup/pgdump/career-strategy/$archive" |
      ssh ops@192.168.20.103 \
        'sudo -n -u postgres pg_restore --exit-on-error --no-owner --role=career_strategy -d career_strategy_restore_test'
    echo "Restore complete. Perform the documented count, representative-row, and app-read checks."
    echo "The disposable database is intentionally retained until those checks are recorded."
    ;;

  *) usage ;;
esac
