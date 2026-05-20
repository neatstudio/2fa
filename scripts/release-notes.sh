#!/usr/bin/env sh
set -eu

VERSION="${1:-${VERSION:-$(tr -d '[:space:]' < VERSION)}}"

cat <<EOF
# 2FA CLI $VERSION

## Highlights

- Local TOTP/2FA CLI with explicit \`list\`/\`ls\` commands.
- Authenticated local Web UI via \`2fa serve\`.
- Realtime browser updates for current codes and countdowns.
- Account CRUD with groups and notes.
- Safe default Web UI binding to localhost and local \`100.64.0.0/10\` addresses.
- Plaintext local JSON storage protected by filesystem permissions.

## Install

Download the archive for your platform, extract it, and place \`2fa\` on your \`PATH\`.

Example:

\`\`\`sh
tar -xzf 2fa-$VERSION-darwin-arm64.tar.gz
install -m 0755 2fa-$VERSION-darwin-arm64/2fa ~/.local/bin/2fa
\`\`\`

## Basic Usage

\`\`\`sh
2fa
2fa add --name github --secret JBSWY3DPEHPK3PXP --group work --note "GitHub admin"
2fa list
2fa list work
2fa serve
\`\`\`

## Security Notes

- Secrets are stored as plaintext base32 values in \`~/.2fa/accounts.json\`.
- The default storage directory is \`0700\`; the accounts file is \`0600\`.
- The Web UI uses a process-local token and does not expose raw secrets through HTML/API/SSE responses.
- Do not expose \`2fa serve\` to untrusted networks.

## Checksums

Verify downloads with \`SHA256SUMS\` from this release.
EOF
