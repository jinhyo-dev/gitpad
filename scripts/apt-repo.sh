#!/usr/bin/env bash
# Build (or update) a signed apt repository from a directory of .deb files.
#
#   scripts/apt-repo.sh <repo-dir> <deb-dir>
#
# Env: APT_GPG_KEY_ID (required), APT_GPG_PASSPHRASE (optional).
# Layout produced under <repo-dir>:
#   key.gpg                                   public key (armored)
#   pool/main/<arch>/*.deb
#   dists/stable/main/binary-<arch>/Packages(.gz)
#   dists/stable/{Release,Release.gpg,InRelease}
set -euo pipefail

REPO_DIR=${1:?repo dir}
DEB_DIR=${2:?deb dir}
KEY_ID=${APT_GPG_KEY_ID:?APT_GPG_KEY_ID is required}
ARCHES="amd64 arm64"

command -v apt-ftparchive >/dev/null || { echo "apt-ftparchive missing (apt-get install apt-utils)"; exit 1; }

mkdir -p "$REPO_DIR"
for arch in $ARCHES; do
  mkdir -p "$REPO_DIR/pool/main/$arch" "$REPO_DIR/dists/stable/main/binary-$arch"
  for f in "$DEB_DIR"/*_"$arch".deb; do
    [ -e "$f" ] && cp -f "$f" "$REPO_DIR/pool/main/$arch/"
  done
done

cd "$REPO_DIR"
for arch in $ARCHES; do
  apt-ftparchive packages "pool/main/$arch" > "dists/stable/main/binary-$arch/Packages"
  gzip -9kf "dists/stable/main/binary-$arch/Packages"
done

cat > /tmp/apt-release.conf <<CONF
APT::FTPArchive::Release::Origin "gitpad";
APT::FTPArchive::Release::Label "gitpad";
APT::FTPArchive::Release::Suite "stable";
APT::FTPArchive::Release::Codename "stable";
APT::FTPArchive::Release::Architectures "$ARCHES";
APT::FTPArchive::Release::Components "main";
APT::FTPArchive::Release::Description "gitpad apt repository";
CONF
apt-ftparchive -c /tmp/apt-release.conf release dists/stable > dists/stable/Release

GPG=(gpg --batch --yes --pinentry-mode loopback -u "$KEY_ID")
if [ -n "${APT_GPG_PASSPHRASE:-}" ]; then
  GPG+=(--passphrase "$APT_GPG_PASSPHRASE")
fi
"${GPG[@]}" -abs -o dists/stable/Release.gpg dists/stable/Release
"${GPG[@]}" --clearsign -o dists/stable/InRelease dists/stable/Release
gpg --batch --yes --armor --export "$KEY_ID" > key.gpg

echo "apt repository updated in $REPO_DIR"
