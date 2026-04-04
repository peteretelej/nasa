# Release Checklist

Steps for cutting a stable release of the NASA Go SDK. This checklist is for maintainers only.

## Pre-Release Verification

Complete every item before creating the tag.

### 1. CI Status

- [ ] The latest commit on `master` has a passing CI run (lint, build, test).
- [ ] Review the [Actions tab](https://github.com/peteretelej/nasa/actions/workflows/ci.yml) for the most recent result.

### 2. Local Verification

Run all three checks locally on a clean checkout of `master`:

```bash
go test -race -count=1 ./...
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run ./...
go build ./...
```

- [ ] All tests pass.
- [ ] Lint reports 0 issues.
- [ ] Build succeeds for all packages.

### 3. Documentation

- [ ] `README.md` reflects current install commands, API coverage, and feature list.
- [ ] `docs/getting-started.md` examples compile and match current SDK behavior.
- [ ] `docs/authentication.md` key handling guidance is accurate.
- [ ] `docs/rate-limiting.md` describes the current rate limiter behavior.
- [ ] `docs/cli/installation.md` matches the CLI's actual flags and output.
- [ ] `CONTRIBUTING.md` contains only durable guidance (no branch-specific or transient notes).

### 4. Docs Deployment Workflow

- [ ] `.github/workflows/docs.yml` triggers on push to `master` for `docs/**` and `README.md`.
- [ ] GitHub Pages is enabled in repo settings (Settings > Pages > Source > GitHub Actions).

## Creating the Release

Only proceed after every pre-release item above is checked.

### Tag the Release

Create an annotated tag on `master` and push it:

```bash
git checkout master
git pull origin master
git tag -a v1.0.0 -m "v1.0.0 - first stable release"
git push origin v1.0.0
```

Do not use a lightweight tag. The annotated tag ensures GitHub creates a proper release entry and pkg.go.dev indexes the version.

## Post-Release Verification

### 1. GitHub Release

- [ ] Visit [Releases](https://github.com/peteretelej/nasa/releases) and confirm the tag appears.
- [ ] Optionally add release notes summarizing what the release includes.

### 2. pkg.go.dev

- [ ] Visit [pkg.go.dev/github.com/peteretelej/nasa](https://pkg.go.dev/github.com/peteretelej/nasa) and confirm the tagged version is indexed.
- [ ] If the version does not appear within a few minutes, request indexing at `https://pkg.go.dev/github.com/peteretelej/nasa@v1.0.0`.

### 3. Documentation Site

- [ ] Confirm the docs workflow ran after the most recent push to `master`.
- [ ] Visit [peteretelej.github.io/nasa](https://peteretelej.github.io/nasa) and verify the site is live and current.

### 4. Install Verification

Verify that users can install the SDK and CLI at the tagged version:

```bash
go get github.com/peteretelej/nasa@v1.0.0
go install github.com/peteretelej/nasa/cmd/nasa@v1.0.0
nasa --version
```
