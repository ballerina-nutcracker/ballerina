#!/usr/bin/env bash
# Setup common agent tooling: link Claude-specific paths to the canonical
# AGENTS.md / .agents/skills sources so Claude Code shares one source of truth.

set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

link() {
  local src="$1" dst="$2"

  if ! (cd "$(dirname "$dst")" && [[ -e "$src" ]]); then
    echo "error: source $src (relative to $dst) missing" >&2
    return 1
  fi

  if [[ -L "$dst" ]]; then
    if [[ "$(readlink "$dst")" == "$src" ]]; then
      echo "$dst already linked"
      return 0
    fi
    rm "$dst"
  elif [[ -e "$dst" ]]; then
    echo "error: $dst exists and is not a symlink" >&2
    return 1
  fi

  ln -s "$src" "$dst"
  echo "linked $dst -> $src"
}

mkdir -p .claude

link AGENTS.md CLAUDE.md
link ../.agents/skills .claude/skills
