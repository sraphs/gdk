#!/usr/bin/env bash

# This script checks to see if there are any incompatible API changes on the
# current branch relative to the upstream branch.
# It fails if it finds any, unless there is a commit with BREAKING_CHANGE_OK
# in the first line of the commit message.
#
# It checks all modules listed in allmodules, and skips packages with
# "internal" or "test" in their name.
#
# It expects to be run at the root of the repository, and that HEAD is pointing
# to a commit that merges between the pull request and the upstream branch
# GITHUB_BASE_REF).

set -euo pipefail

UPSTREAM_BRANCH="${GITHUB_BASE_REF:-master}"
echo "Checking for incompatible API changes relative to ${UPSTREAM_BRANCH}..."

MASTER_CLONE_DIR="$(mktemp -d)"
PKGINFO_BRANCH=$(mktemp)
PKGINFO_MASTER=$(mktemp)

function cleanup() {
  rm -f "$PKGINFO_BRANCH" "$PKGINFO_MASTER"
}
trap cleanup EXIT

# Install apidiff.
go install golang.org/x/exp/cmd/apidiff@latest

git clone -b "$UPSTREAM_BRANCH" . "$MASTER_CLONE_DIR" &>/dev/null

# Run the following checks in the master directory
ORIG_DIR="$(pwd)"
cd "$MASTER_CLONE_DIR"

incompatible_change_pkgs=()
while read -r path || [[ -n "$path" ]]; do
  echo "  checking packages in module $path"
  pushd "$path" &>/dev/null

  PKGS=$(go list ./...)
  for pkg in $PKGS; do
    if [[ "$pkg" =~ "test" ]] || [[ "$pkg" =~ "internal" ]] || [[ "$pkg" =~ "samples" ]]; then
      continue
    fi
    echo "    checking ${pkg}..."

    # Compute export data for the current branch.
    package_deleted=0
    (cd "$ORIG_DIR/$path" && apidiff -w "$PKGINFO_BRANCH" "$pkg") || package_deleted=1
    if [[ $package_deleted -eq 1 ]]; then
      echo "    package ${pkg} was deleted! Recording as an incompatible change."
      incompatible_change_pkgs+=("${pkg}")
      continue
    fi

    # Compute export data for master@HEAD.
    apidiff -w "$PKGINFO_MASTER" "$pkg"

    # Print all changes for posterity.
    apidiff "$PKGINFO_MASTER" "$PKGINFO_BRANCH"

    # Note if there's an incompatible change.
    ic=$(apidiff -incompatible "$PKGINFO_MASTER" "$PKGINFO_BRANCH")
    if [ -n "$ic" ]; then
      incompatible_change_pkgs+=("$pkg")
    fi
  done
  popd &>/dev/null
done < <(sed -e '/^#/d' -e '/^$/d' allmodules | awk '{print $1}')

if [ ${#incompatible_change_pkgs[@]} -eq 0 ]; then
  # No incompatible changes, we are good.
  echo "OK: No incompatible changes found."
  exit 0
fi
echo "Found breaking API change(s) in: ${incompatible_change_pkgs[*]}."

# Found incompatible changes; see if they were declared as OK via a commit.
cd "$ORIG_DIR"
if git cherry -v master | grep -q "BREAKING_CHANGE_OK"; then
  echo "Allowing them due to a commit message with BREAKING_CHANGE_OK."
  exit 0
fi

echo "FAIL. If this is expected and OK, you can pass this check by adding a commit with BREAKING_CHANGE_OK in the first line of the message."
exit 1
