#!/usr/bin/env bash
# One-time: create the GPG key that signs the apt repository and print the
# values to store as GitHub Actions secrets.
#
#   scripts/apt-keygen.sh            (asks for a passphrase)
set -euo pipefail

read -rsp "Passphrase for the new signing key: " PASS; echo
export GNUPGHOME=$(mktemp -d)
trap 'rm -rf "$GNUPGHOME"' EXIT

gpg --batch --pinentry-mode loopback --passphrase "$PASS" --quick-generate-key \
  "gitpad apt signing <jinhyo-dev@users.noreply.github.com>" rsa4096 sign 5y
KEY_ID=$(gpg --list-secret-keys --with-colons | awk -F: '/^fpr/ {print $10; exit}')

OUT=${1:-apt-signing-key.asc}
gpg --batch --pinentry-mode loopback --passphrase "$PASS" --armor --export-secret-keys "$KEY_ID" > "$OUT"
chmod 600 "$OUT"

cat <<MSG

Key created. Add these repository secrets (Settings → Secrets and variables → Actions):

  APT_GPG_KEY_ID       $KEY_ID
  APT_GPG_PASSPHRASE   <the passphrase you just typed>
  APT_GPG_PRIVATE_KEY  <contents of $OUT>

Keep $OUT somewhere safe (password manager) and delete it from disk afterwards.
MSG
