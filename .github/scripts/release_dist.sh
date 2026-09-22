#!/usr/bin/env bash
# Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
#
# WSO2 LLC. licenses this file to you under the Apache License,
# Version 2.0 (the "License"); you may not use this file except
# in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied. See the License for the
# specific language governing permissions and limitations
# under the License.

set -euo pipefail

module_prefix="github.com/ballerina-nutcracker/ballerina"
mode="release"
if [[ "${1:-}" == "--check-tags" ]]; then
    mode="check-tags"
    shift
fi

version="${1:-}"
remote_or_commit="${2:-}"

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

branch=""
remote="${remote_or_commit:-origin}"
if [[ "$mode" == "release" ]]; then
    if [[ -n "$(git status --porcelain)" ]]; then
        echo "refusing to release from a dirty worktree" >&2
        exit 1
    fi
    branch="$(git symbolic-ref --quiet --short HEAD || true)"
    if [[ -z "$branch" ]]; then
        echo "refusing to release from a detached HEAD" >&2
        exit 1
    fi

    if [[ -z "$version" ]]; then
        version="v$("$(dirname "$0")/resolve_next_version.sh" "$branch")"
    fi
fi

if [[ ! "$version" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$ ]]; then
    echo "usage: $0 [--check-tags] vMAJOR.MINOR.PATCH[-PRERELEASE] [remote|commit]" >&2
    exit 1
fi

workspace_dirs="$(go list -m -f '{{if .Main}}{{.Dir}}{{end}}' all | sed '/^$/d' | sort)"
if [[ -z "$workspace_dirs" ]]; then
    echo "no modules found in go.work" >&2
    exit 1
fi

expected_tags() {
    local module_dir relative_dir
    while IFS= read -r module_dir; do
        if [[ "$module_dir" == "$repo_root" ]]; then
            printf '%s\n' "$version"
        elif [[ "$module_dir" == "$repo_root/"* ]]; then
            relative_dir="${module_dir#"$repo_root/"}"
            printf '%s\n' "$relative_dir/$version"
        else
            echo "workspace module directory $module_dir is outside $repo_root" >&2
            return 1
        fi
    done <<< "$workspace_dirs"
}

check_tags() {
    local target_commit="$1" tag actual
    while IFS= read -r tag; do
        actual="$(git rev-parse -q --verify "refs/tags/$tag^{commit}" || true)"
        if [[ "$actual" != "$target_commit" ]]; then
            echo "tag $tag does not point to $target_commit" >&2
            return 1
        fi
        printf '%s\n' "$tag"
    done < <(expected_tags)
}

if [[ "$mode" == "check-tags" ]]; then
    target_commit="$(git rev-parse --verify "${remote_or_commit:-HEAD}^{commit}")"
    check_tags "$target_commit"
    exit 0
fi

modules_file="$(mktemp)"
trap 'rm -f "$modules_file"' EXIT
go list -m -f '{{if .Main}}{{.Path}}{{"\t"}}{{.Dir}}{{end}}' all |
    awk -F '\t' -v prefix="$module_prefix" '$1 == prefix || index($1, prefix "/") == 1' > "$modules_file"
if [[ ! -s "$modules_file" ]]; then
    echo "no distribution modules found in the Go workspace" >&2
    exit 1
fi

internal_requirements() {
    awk -v prefix="$module_prefix" '
        ($1 == prefix || index($1, prefix "/") == 1) && $2 ~ /^v[0-9]/ { print $1, $2 }
        $1 == "require" && ($2 == prefix || index($2, prefix "/") == 1) && $3 ~ /^v[0-9]/ { print $2, $3 }
    ' "$1"
}

existing_versions="$({
    while IFS=$'\t' read -r _ module_dir; do
        internal_requirements "$module_dir/go.mod" | awk '{ print $2 }'
    done < "$modules_file"
    grep -Eo 'v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?' go.work
} | sort -u)"
version_count="$(printf '%s\n' "$existing_versions" | sed '/^$/d' | wc -l | tr -d ' ')"
if [[ "$version_count" -ne 1 ]]; then
    echo "internal module references do not use one shared version:" >&2
    printf '%s\n' "$existing_versions" >&2
    exit 1
fi
old_version="$(printf '%s\n' "$existing_versions" | sed '/^$/d')"

if [[ "$old_version" != "$version" ]]; then
    while IFS=$'\t' read -r _ module_dir; do
        dependencies="$(internal_requirements "$module_dir/go.mod")"
        while read -r dependency _; do
            [[ -z "$dependency" ]] && continue
            go mod edit "-require=$dependency@$version" "$module_dir/go.mod"
        done < <(printf '%s\n' "$dependencies")
    done < "$modules_file"
    python3 - "$old_version" "$version" go.work <<'PY'
from pathlib import Path
import sys

old_version, new_version, *files = sys.argv[1:]
for name in files:
    path = Path(name)
    content = path.read_text()
    updated = content.replace(old_version, new_version)
    if updated == content:
        raise SystemExit(f"{name}: did not contain {old_version}")
    path.write_text(updated)
PY
fi

failed=false
while IFS=$'\t' read -r _ module_dir; do
    while read -r dependency dependency_version; do
        [[ -z "$dependency" ]] && continue
        if [[ "$dependency_version" != "$version" ]]; then
            echo "$module_dir/go.mod: $dependency is $dependency_version, expected $version" >&2
            failed=true
        fi
    done < <(internal_requirements "$module_dir/go.mod")
done < "$modules_file"
while read -r found_version; do
    [[ -z "$found_version" ]] && continue
    if [[ "$found_version" != "$version" ]]; then
        echo "go.work: found $found_version, expected $version" >&2
        failed=true
    fi
done < <(grep -Eo 'v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?' go.work | sort -u)
if [[ "$failed" == true ]]; then
    exit 1
fi

make build
while IFS=$'\t' read -r _ module_dir; do
    git add "$module_dir/go.mod"
done < "$modules_file"
git add go.work
if ! git diff --cached --quiet; then
    git commit -m "chore(release): prepare $version"
fi
if [[ -n "$(git status --porcelain)" ]]; then
    echo "release preparation left uncommitted changes" >&2
    exit 1
fi

target_commit="$(git rev-parse --verify HEAD^{commit})"
git push "$remote" "HEAD:refs/heads/$branch"

is_final_release=false
if [[ "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    is_final_release=true
fi
release_branch="release-$(echo "${version#v}" | cut -d. -f1,2).x"

if [[ "$is_final_release" == true ]]; then
    # Anchors this version line for future patches. Only done on a final
    # release — each prerelease has its own commit, so anchoring on every
    # rc.N would collide with the first one.
    existing_release_branch="$(git rev-parse -q --verify "refs/heads/$release_branch^{commit}" || true)"
    if [[ -n "$existing_release_branch" && "$existing_release_branch" != "$target_commit" ]]; then
        echo "branch $release_branch already exists and points to $existing_release_branch" >&2
        exit 1
    fi
    if [[ -z "$existing_release_branch" ]]; then
        git branch "$release_branch" "$target_commit"
    fi
fi

tags=()
while IFS= read -r tag; do
    [[ -n "$tag" ]] && tags+=("$tag")
done < <(expected_tags)
for tag in "${tags[@]}"; do
    existing="$(git rev-parse -q --verify "refs/tags/$tag^{commit}" || true)"
    if [[ -n "$existing" && "$existing" != "$target_commit" ]]; then
        echo "tag $tag already points to $existing" >&2
        exit 1
    fi
    if [[ -z "$existing" ]]; then
        git tag "$tag" "$target_commit"
    fi
done
module_tags=()
for tag in "${tags[@]}"; do
    [[ "$tag" != "$version" ]] && module_tags+=("$tag")
done
if [[ "$is_final_release" == true ]]; then
    if [[ "${#module_tags[@]}" -gt 0 ]]; then
        git push --atomic "$remote" "refs/heads/$release_branch" "${module_tags[@]/#/refs/tags/}"
    else
        git push "$remote" "refs/heads/$release_branch"
    fi
elif [[ "${#module_tags[@]}" -gt 0 ]]; then
    git push --atomic "$remote" "${module_tags[@]/#/refs/tags/}"
fi
# Pushed alone: an atomic multi-tag push never fires GitHub's tag webhook,
# which would silently break release.yml's trigger.
git push "$remote" "refs/tags/$version"
check_tags "$target_commit"
