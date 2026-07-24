# Default recipe: list available commands
default:
    @just --list

# Run muxly directly with go
run *args:
    go run . {{args}}

# Build and install the binary
install:
    go install

# Run all tests
test *args:
    go test ./... {{args}}

# Format all Go files
fmt:
    go fmt ./...

# Static analysis
vet:
    go vet ./...

# Clean up module dependencies
tidy:
    go mod tidy

# Run formatting, vet, and tests
check: fmt vet test

# Cut a release. Version comes from the git tag (goreleaser injects it via
# ldflags), so there is no file to edit: this tags HEAD and pushes, which triggers
# the goreleaser workflow (GitHub release + Homebrew cask + AUR).
#
#   just release              # auto-detect bump from conventional commits (svu)
#   just release minor        # force a specific bump (major graduates 0.x -> 1.0.0)
#   just release 1.2.3        # set an explicit version
#   just release -n           # dry-run any of the above
#
# Auto-detect uses `svu next --v0` so, while on 0.x, a breaking change bumps the
# minor instead of jumping to 1.0.0. If any commit since the last tag lacks a
# conventional type, auto-detect bails and shows the log so you can pick a bump.
release *args:
    #!/usr/bin/env bash
    set -euo pipefail

    # Locate svu (go install github.com/caarlos0/svu/v3@latest)
    svu="$(command -v svu || true)"
    [ -z "$svu" ] && svu="$(go env GOPATH)/bin/svu"
    if [ ! -x "$svu" ]; then
        echo "svu not found. Install it with:"
        echo "  go install github.com/caarlos0/svu/v3@latest"
        exit 1
    fi

    last="$("$svu" current)"
    branch="$(git rev-parse --abbrev-ref HEAD)"

    # Parse args: an optional bump (major|minor|patch) or explicit version, plus -n/--dry-run.
    bump=""
    explicit=""
    dry_run=false
    for arg in {{args}}; do
        case "$arg" in
            major|minor|patch) bump="$arg" ;;
            -n|--dry-run) dry_run=true ;;
            v[0-9]*|[0-9]*) explicit="${arg#v}" ;;
            *) echo "Unknown arg: $arg. Use major|minor|patch, a version (X.Y.Z), and/or -n/--dry-run."; exit 1 ;;
        esac
    done

    # Determine the target tag.
    if [ -n "$explicit" ]; then
        if [[ ! "$explicit" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            echo "Invalid version: $explicit (expected X.Y.Z)"; exit 1
        fi
        tag="v${explicit}"
    elif [ -n "$bump" ]; then
        # Forced bump: svu computes from the latest tag. `major` at 0.x graduates to 1.0.0.
        tag="$("$svu" "$bump")"
    else
        # Auto-detect from conventional commits since the last tag.
        range="${last}..HEAD"
        subjects="$(git log "$range" --no-merges --format='%s')"
        if [ -z "$subjects" ]; then
            echo "No commits since ${last}."; exit 1
        fi
        nonconv="$(echo "$subjects" | grep -vE '^[a-z]+(\([^)]+\))?!?: .+' || true)"
        if [ -n "$nonconv" ]; then
            echo "Found commit(s) without a conventional type since ${last}:"
            echo "$nonconv" | sed 's/^/  /'
            echo ""
            echo "Auto-detect needs conventional commits. Re-run with an explicit bump:"
            echo "  just release <patch|minor|major|X.Y.Z>"
            echo ""
            git log --oneline "$range"
            exit 1
        fi
        tag="$("$svu" next --v0)"
        if [ "$tag" = "$last" ]; then
            echo "No release-worthy commits (only chore/docs/refactor/etc.) since ${last}."
            echo ""
            git log --oneline "$range"
            echo ""
            echo "Re-run with an explicit bump to release anyway:"
            echo "  just release <patch|minor|major|X.Y.Z>"
            exit 1
        fi
    fi

    if git rev-parse "$tag" >/dev/null 2>&1; then
        echo "Error: tag ${tag} already exists"; exit 1
    fi

    if [ "$dry_run" = true ]; then
        echo "Dry run — nothing will be tagged or pushed"
        echo ""
        echo "Current: ${last}"
        echo "Next:    ${tag}"
        echo "Branch:  ${branch}"
        echo ""
        echo "Would run: git tag ${tag} && git push origin ${branch} && git push origin ${tag}"
        exit 0
    fi

    # Untracked files are fine (e.g. WIP); tracked changes must be committed so the
    # tag captures the intended state.
    if ! git diff --quiet || ! git diff --cached --quiet; then
        echo "Error: uncommitted changes present. Commit or stash before releasing."; exit 1
    fi

    git tag "$tag"
    git push origin "$branch"
    git push origin "$tag"
    echo "Tagged ${tag} and pushed — goreleaser will build and publish the release."
