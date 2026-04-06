#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPOS="$SCRIPT_DIR/repos"
REMOTES="$SCRIPT_DIR/remotes"

if [ -d "$REPOS" ]; then
    echo "repos/ already exists. Run reset.sh to recreate."
    exit 1
fi

mkdir -p "$REPOS" "$REMOTES"

# Fixed dates for reproducibility
export GIT_AUTHOR_DATE="2025-01-15T10:00:00+00:00"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00+00:00"

# Allow local file:// protocol for submodules
export GIT_CONFIG_COUNT=1
export GIT_CONFIG_KEY_0="protocol.file.allow"
export GIT_CONFIG_VALUE_0="always"

# Helper: init a repo with standard config and an initial commit
init_repo() {
    local dir="$1"
    mkdir -p "$dir"
    git -C "$dir" init -b main -q
    git -C "$dir" config user.name "Test User"
    git -C "$dir" config user.email "test@fossor.dev"
    echo "# $(basename "$dir")" > "$dir/README.md"
    git -C "$dir" add README.md
    git -C "$dir" commit -q -m "Initial commit"
}

# Helper: create a bare remote, push main to it, set up tracking
setup_remote() {
    local repo="$1"
    local name="$(basename "$repo")"
    local bare="$REMOTES/${name}.git"
    git init --bare -b main -q "$bare"
    git -C "$repo" remote add origin "$bare"
    git -C "$repo" push -q -u origin main
}

# Helper: commit to a bare remote directly (simulates someone else pushing)
commit_to_bare() {
    local bare="$1"
    local msg="$2"
    local tmp
    tmp=$(mktemp -d)
    git clone -q "$bare" "$tmp"
    git -C "$tmp" config user.name "Other User"
    git -C "$tmp" config user.email "other@fossor.dev"
    echo "$msg" >> "$tmp/remote-change.txt"
    git -C "$tmp" add remote-change.txt
    GIT_AUTHOR_DATE="2025-01-15T11:00:00+00:00" GIT_COMMITTER_DATE="2025-01-15T11:00:00+00:00" \
        git -C "$tmp" commit -q -m "$msg"
    git -C "$tmp" push -q
    rm -rf "$tmp"
}

echo "Creating 20 test repositories..."

# --- 1. clean ---
echo "  1/20 clean"
init_repo "$REPOS/clean"
setup_remote "$REPOS/clean"
git -C "$REPOS/clean" fetch -q

# --- 2. dirty-unstaged ---
echo "  2/20 dirty-unstaged"
init_repo "$REPOS/dirty-unstaged"
setup_remote "$REPOS/dirty-unstaged"
git -C "$REPOS/dirty-unstaged" fetch -q
echo "modified content" >> "$REPOS/dirty-unstaged/README.md"

# --- 3. dirty-staged ---
echo "  3/20 dirty-staged"
init_repo "$REPOS/dirty-staged"
setup_remote "$REPOS/dirty-staged"
git -C "$REPOS/dirty-staged" fetch -q
echo "staged change" >> "$REPOS/dirty-staged/README.md"
git -C "$REPOS/dirty-staged" add README.md

# --- 4. dirty-mixed ---
echo "  4/20 dirty-mixed"
init_repo "$REPOS/dirty-mixed"
setup_remote "$REPOS/dirty-mixed"
git -C "$REPOS/dirty-mixed" fetch -q
echo "staged" > "$REPOS/dirty-mixed/staged.txt"
git -C "$REPOS/dirty-mixed" add staged.txt
echo "unstaged" >> "$REPOS/dirty-mixed/README.md"

# --- 5. ahead ---
echo "  5/20 ahead"
init_repo "$REPOS/ahead"
setup_remote "$REPOS/ahead"
git -C "$REPOS/ahead" fetch -q
echo "local change" > "$REPOS/ahead/local.txt"
git -C "$REPOS/ahead" add local.txt
GIT_AUTHOR_DATE="2025-01-15T12:00:00+00:00" GIT_COMMITTER_DATE="2025-01-15T12:00:00+00:00" \
    git -C "$REPOS/ahead" commit -q -m "Add local change"

# --- 6. behind ---
echo "  6/20 behind"
init_repo "$REPOS/behind"
setup_remote "$REPOS/behind"
commit_to_bare "$REMOTES/behind.git" "Remote update"
git -C "$REPOS/behind" fetch -q

# --- 7. diverged ---
echo "  7/20 diverged"
init_repo "$REPOS/diverged"
setup_remote "$REPOS/diverged"
commit_to_bare "$REMOTES/diverged.git" "Remote diverge commit"
git -C "$REPOS/diverged" fetch -q
echo "local diverge" > "$REPOS/diverged/local.txt"
git -C "$REPOS/diverged" add local.txt
GIT_AUTHOR_DATE="2025-01-15T12:30:00+00:00" GIT_COMMITTER_DATE="2025-01-15T12:30:00+00:00" \
    git -C "$REPOS/diverged" commit -q -m "Local diverge commit"

# --- 8. non-default ---
echo "  8/20 non-default"
init_repo "$REPOS/non-default"
setup_remote "$REPOS/non-default"
git -C "$REPOS/non-default" fetch -q
git -C "$REPOS/non-default" checkout -q -b feature/new-feature

# --- 9. non-default-dirty ---
echo "  9/20 non-default-dirty"
init_repo "$REPOS/non-default-dirty"
setup_remote "$REPOS/non-default-dirty"
git -C "$REPOS/non-default-dirty" fetch -q
git -C "$REPOS/non-default-dirty" checkout -q -b feature/wip
echo "work in progress" > "$REPOS/non-default-dirty/wip.txt"

# --- 10. non-default-ahead ---
echo "  10/20 non-default-ahead"
init_repo "$REPOS/non-default-ahead"
setup_remote "$REPOS/non-default-ahead"
git -C "$REPOS/non-default-ahead" fetch -q
git -C "$REPOS/non-default-ahead" checkout -q -b feature/ready
echo "feature code" > "$REPOS/non-default-ahead/feature.txt"
git -C "$REPOS/non-default-ahead" add feature.txt
GIT_AUTHOR_DATE="2025-01-15T13:00:00+00:00" GIT_COMMITTER_DATE="2025-01-15T13:00:00+00:00" \
    git -C "$REPOS/non-default-ahead" commit -q -m "Implement feature"

# --- 11. ahead-dirty ---
echo "  11/20 ahead-dirty"
init_repo "$REPOS/ahead-dirty"
setup_remote "$REPOS/ahead-dirty"
git -C "$REPOS/ahead-dirty" fetch -q
echo "committed" > "$REPOS/ahead-dirty/committed.txt"
git -C "$REPOS/ahead-dirty" add committed.txt
GIT_AUTHOR_DATE="2025-01-15T14:00:00+00:00" GIT_COMMITTER_DATE="2025-01-15T14:00:00+00:00" \
    git -C "$REPOS/ahead-dirty" commit -q -m "Committed change"
echo "uncommitted" >> "$REPOS/ahead-dirty/README.md"

# --- 12. untracked-files ---
echo "  12/20 untracked-files"
init_repo "$REPOS/untracked-files"
setup_remote "$REPOS/untracked-files"
git -C "$REPOS/untracked-files" fetch -q
echo "new file 1" > "$REPOS/untracked-files/new-file.txt"
echo "new file 2" > "$REPOS/untracked-files/another-new.txt"
echo "temp data" > "$REPOS/untracked-files/temp.log"

# --- 13. untracked-dir ---
echo "  13/20 untracked-dir"
init_repo "$REPOS/untracked-dir"
setup_remote "$REPOS/untracked-dir"
git -C "$REPOS/untracked-dir" fetch -q
mkdir -p "$REPOS/untracked-dir/new-feature"
echo "component code" > "$REPOS/untracked-dir/new-feature/component.go"
echo "test code" > "$REPOS/untracked-dir/new-feature/component_test.go"
echo "docs" > "$REPOS/untracked-dir/new-feature/README.md"

# --- 14. stashed ---
echo "  14/20 stashed"
init_repo "$REPOS/stashed"
setup_remote "$REPOS/stashed"
git -C "$REPOS/stashed" fetch -q
# Stash 1
echo "stash work 1" > "$REPOS/stashed/work1.txt"
git -C "$REPOS/stashed" add work1.txt
GIT_AUTHOR_DATE="2025-01-15T15:00:00+00:00" GIT_COMMITTER_DATE="2025-01-15T15:00:00+00:00" \
    git -C "$REPOS/stashed" stash push -q -m "WIP: first feature"
# Stash 2
echo "stash work 2" >> "$REPOS/stashed/README.md"
GIT_AUTHOR_DATE="2025-01-15T15:30:00+00:00" GIT_COMMITTER_DATE="2025-01-15T15:30:00+00:00" \
    git -C "$REPOS/stashed" stash push -q -m "WIP: readme updates"
# Stash 3
echo "stash work 3" > "$REPOS/stashed/experiment.txt"
git -C "$REPOS/stashed" add experiment.txt
GIT_AUTHOR_DATE="2025-01-15T16:00:00+00:00" GIT_COMMITTER_DATE="2025-01-15T16:00:00+00:00" \
    git -C "$REPOS/stashed" stash push -q -m "WIP: experiment"

# --- 15. staged-delete-rename ---
echo "  15/20 staged-delete-rename"
init_repo "$REPOS/staged-delete-rename"
# Create files to delete and rename
echo "will be deleted" > "$REPOS/staged-delete-rename/to-delete.txt"
echo "will be renamed" > "$REPOS/staged-delete-rename/old-name.txt"
echo "normal file" > "$REPOS/staged-delete-rename/normal.txt"
git -C "$REPOS/staged-delete-rename" add .
GIT_AUTHOR_DATE="2025-01-15T10:30:00+00:00" GIT_COMMITTER_DATE="2025-01-15T10:30:00+00:00" \
    git -C "$REPOS/staged-delete-rename" commit -q -m "Add files"
setup_remote "$REPOS/staged-delete-rename"
git -C "$REPOS/staged-delete-rename" fetch -q
git -C "$REPOS/staged-delete-rename" rm -q to-delete.txt
git -C "$REPOS/staged-delete-rename" mv old-name.txt new-name.txt

# --- 16. large-history ---
echo "  16/20 large-history"
init_repo "$REPOS/large-history"
for i in $(seq 2 60); do
    echo "Change $i" >> "$REPOS/large-history/log.txt"
    git -C "$REPOS/large-history" add log.txt
    ts=$(printf "2025-01-%02dT%02d:00:00+00:00" $(( (i / 24) + 15 )) $(( i % 24 )))
    GIT_AUTHOR_DATE="$ts" GIT_COMMITTER_DATE="$ts" \
        git -C "$REPOS/large-history" commit -q -m "Change $i: $([ $((i % 3)) -eq 0 ] && echo 'fix bug' || ([ $((i % 3)) -eq 1 ] && echo 'add feature' || echo 'refactor code'))"
done
setup_remote "$REPOS/large-history"
git -C "$REPOS/large-history" fetch -q

# --- 17. merge-conflict ---
echo "  17/20 merge-conflict"
init_repo "$REPOS/merge-conflict"
echo "original line" > "$REPOS/merge-conflict/shared.txt"
git -C "$REPOS/merge-conflict" add shared.txt
GIT_AUTHOR_DATE="2025-01-15T10:30:00+00:00" GIT_COMMITTER_DATE="2025-01-15T10:30:00+00:00" \
    git -C "$REPOS/merge-conflict" commit -q -m "Add shared file"
# Create conflicting branch
git -C "$REPOS/merge-conflict" checkout -q -b conflict-branch
echo "branch version" > "$REPOS/merge-conflict/shared.txt"
GIT_AUTHOR_DATE="2025-01-15T11:00:00+00:00" GIT_COMMITTER_DATE="2025-01-15T11:00:00+00:00" \
    git -C "$REPOS/merge-conflict" commit -q -am "Branch change"
# Modify main
git -C "$REPOS/merge-conflict" checkout -q main
echo "main version" > "$REPOS/merge-conflict/shared.txt"
GIT_AUTHOR_DATE="2025-01-15T11:30:00+00:00" GIT_COMMITTER_DATE="2025-01-15T11:30:00+00:00" \
    git -C "$REPOS/merge-conflict" commit -q -am "Main change"
# Trigger conflict
git -C "$REPOS/merge-conflict" merge conflict-branch -q --no-edit 2>/dev/null || true

# --- 18. empty-repo ---
echo "  18/20 empty-repo"
mkdir -p "$REPOS/empty-repo"
git -C "$REPOS/empty-repo" init -b main -q
git -C "$REPOS/empty-repo" config user.name "Test User"
git -C "$REPOS/empty-repo" config user.email "test@fossor.dev"

# --- 19. with-submodule-modified ---
echo "  19/20 with-submodule-modified"
# Create the submodule source repo
init_repo "$REMOTES/submodule-source"
setup_remote_bare="$REMOTES/submodule-source-bare.git"
git init --bare -b main -q "$setup_remote_bare"
git -C "$REMOTES/submodule-source" remote add origin "$setup_remote_bare"
git -C "$REMOTES/submodule-source" push -q -u origin main
# Add another commit to source
echo "v2" >> "$REMOTES/submodule-source/README.md"
GIT_AUTHOR_DATE="2025-01-15T17:00:00+00:00" GIT_COMMITTER_DATE="2025-01-15T17:00:00+00:00" \
    git -C "$REMOTES/submodule-source" commit -q -am "Update submodule source"
git -C "$REMOTES/submodule-source" push -q

# Create parent repo with submodule at old commit
init_repo "$REPOS/with-submodule-modified"
git -C "$REPOS/with-submodule-modified" submodule add -q "$setup_remote_bare" libs/shared
GIT_AUTHOR_DATE="2025-01-15T17:30:00+00:00" GIT_COMMITTER_DATE="2025-01-15T17:30:00+00:00" \
    git -C "$REPOS/with-submodule-modified" commit -q -m "Add submodule"
# Update submodule to new commit (creates a modified submodule state)
git -C "$REPOS/with-submodule-modified/libs/shared" pull -q origin main

# --- 20. with-submodule-dirty ---
echo "  20/20 with-submodule-dirty"
init_repo "$REPOS/with-submodule-dirty"
git -C "$REPOS/with-submodule-dirty" submodule add -q "$setup_remote_bare" libs/shared
GIT_AUTHOR_DATE="2025-01-15T18:00:00+00:00" GIT_COMMITTER_DATE="2025-01-15T18:00:00+00:00" \
    git -C "$REPOS/with-submodule-dirty" commit -q -m "Add submodule"
# Make submodule dirty
echo "local submodule edit" >> "$REPOS/with-submodule-dirty/libs/shared/README.md"

echo ""
echo "Done! Created 20 repos in $REPOS"
echo "Run fossor with: fossor $REPOS"
