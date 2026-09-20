#!/usr/bin/env bash
set -euo pipefail
set +x
cd "$(dirname "$0")/.."

case "${1:-}" in
  ''|--defaults) ;;
  *) echo "Usage: npm run setup -- [--defaults]" >&2; exit 1 ;;
esac

if [[ -e .env || -L .env ]]; then
  echo ".env already exists; keeping your settings. Edit it directly to make changes."
  exit 0
fi

username=admin
password=password123
if [[ "${1:-}" != --defaults && -t 0 ]]; then
  read -r -p "Use local defaults (admin / password123)? [Y/n] " answer
  case "$answer" in
    n|N|no|NO)
      read -r -p "Username [admin]: " username
      username="${username:-admin}"
      read -r -s -p "Password [password123]: " password
      echo
      password="${password:-password123}"
      ;;
  esac
fi

# Pass credentials through stdin, never command arguments or shell interpolation.
printf '%s\n%s' "$username" "$password" | node --input-type=module -e '
  import { readFileSync, writeFileSync } from "node:fs";
  import { randomBytes } from "node:crypto";
  import bcrypt from "bcryptjs";

  const input = readFileSync(0, "utf8");
  const separator = input.indexOf("\n");
  const username = input.slice(0, separator);
  const password = input.slice(separator + 1);
  if (!/^[a-zA-Z0-9_.@-]+$/.test(username)) {
    throw new Error("Username may contain letters, numbers, underscores, dots, @ and hyphens.");
  }
  if (Buffer.byteLength(password) > 72) throw new Error("Password must be at most 72 bytes.");
  const values = {
    AUTH_USERNAME: username,
    AUTH_PASSWORD_HASH: bcrypt.hashSync(password, 10),
    JWT_SECRET: randomBytes(48).toString("base64url"),
  };
  const template = readFileSync(".env.example", "utf8");
  const content = template.replace(/^(AUTH_USERNAME|AUTH_PASSWORD_HASH|JWT_SECRET)=.*$/gm,
    (_, key) => `${key}=${String.fromCharCode(39)}${values[key]}${String.fromCharCode(39)}`);
  writeFileSync(".env", content, { flag: "wx", mode: 0o600 });
'
unset password
echo "Created .env for local development (username: $username)."
echo "Next: make postgres-up && npm run build && make go-dev"
