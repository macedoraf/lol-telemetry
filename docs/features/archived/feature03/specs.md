# Feature 03 Specification: How to Generate a Windows Artifact

## 1. Objective
Provide clear, reproducible instructions and a lightweight build helper for producing a Windows executable (`lol-cli.exe`) from the Go CLI source. This enables developers and users on Windows to run `lol-telemetry` without installing a local Go toolchain or building from source manually.

## 2. Requirements

### 2.1 Build Instructions (`README.md`)
- Add a dedicated "Build a Windows Artifact" section to `README.md`.
- Explain the cross-compilation command: `GOOS=windows GOARCH=amd64 go build -o dist/lol-cli.exe ./cmd/lol-cli`.
- Mention that the binary is statically linked and does not require CGO.
- Include the output path and the expected file name.

### 2.2 Build Helper (`Makefile`)
- Add a `Makefile` at the project root with the following targets:
  - `build-windows` — cross-compiles `dist/lol-cli.exe` for Windows amd64.
  - `build` — compiles the native binary for the current platform (`dist/lol-cli`).
  - `clean` — removes the `dist/` directory.
  - `help` — prints a short description of each target.
- The `Makefile` must be portable and avoid shell-specific syntax beyond what GNU Make supports on macOS, Linux, and Windows (via `make` from MSYS2/WSL/Git Bash).

### 2.3 Artifact Directory
- The build helper must create a `dist/` directory if it does not exist.
- The `dist/` directory must be ignored by Git.
- Update `.gitignore` to include `dist/`.

### 2.4 Verification Script
- Add a `scripts/build-windows.sh` shell script (or equivalent in the `Makefile`) that:
  1. Runs `go build` with the Windows target.
  2. Verifies the output file exists and is a Windows PE executable.
  3. Prints the path and size.
- The script should be executable on Unix-like systems (macOS/Linux) and in WSL/Git Bash on Windows.

### 2.5 GitHub Actions Workflow
- Add `.github/workflows/build-windows.yml` that triggers on every push and pull request.
- The workflow runs on `ubuntu-latest` and uses the repository's `Makefile` target `build-windows`.
- It uploads the produced `dist/lol-cli.exe` as a workflow artifact named `lol-cli-windows` so the artifact can be downloaded from the GitHub Actions UI.
- The workflow also runs `go test ./...` to ensure the artifact is only built from a green test suite.

### 2.6 Semantic Versioning & Release
- Add `.github/workflows/release-windows.yml` that triggers on:
  - Tag pushes matching `v*.*.*` (semantic versioning, e.g., `v1.0.0`).
  - Pushes to the `main` branch.
- The workflow runs on `ubuntu-latest` and:
  1. Checks out the commit.
  2. Runs `go test ./...`.
  3. Runs `make build-windows` to produce `dist/lol-cli.exe`.
  4. Creates a GitHub Release using a generated release name.
  5. Attaches `dist/lol-cli.exe` as a release asset.
- For tag pushes, the release title should be the tag name (e.g., `v1.0.0`).
- For `main` branch pushes, the release name should include a timestamp or commit short SHA (e.g., `main-20260725-abc123`) so each push produces a unique, overwritable pre-release or snapshot release.
- The release body should list the attached artifact and a brief usage note.

### 2.7 Documentation Updates
- Update `README.md` with both the manual command and the `make build-windows` shortcut.
- Add a "GitHub Actions" subsection describing the automated Windows build artifact.
- Add a "Releasing" subsection explaining how to push a semantic version tag (`git tag v1.0.0 && git push origin v1.0.0`) to trigger the release workflow.
- Keep existing Docker Compose and mock-mode instructions intact.
- Optionally add a note that the produced Windows artifact runs in a terminal and supports both live mode and `--mock` mode.

## 3. Out of Scope
- Windows installer (`.msi`) or packaged app.
- Code signing for the Windows executable.
- 32-bit (`GOARCH=386`) or ARM builds.
- Publishing artifacts to a package registry outside GitHub Releases.

## 4. Acceptance Criteria
- [x] `README.md` contains a "Build a Windows Artifact" section with both the manual `go build` command and the `make build-windows` shortcut.
- [x] `Makefile` exists with `build-windows`, `build`, `clean`, and `help` targets.
- [x] `scripts/build-windows.sh` exists and produces `dist/lol-cli.exe` when executed.
- [x] `.gitignore` ignores the `dist/` directory.
- [x] Running `make build-windows` on a clean repository succeeds and creates a non-empty `dist/lol-cli.exe`.
- [x] Running `make clean` removes the `dist/` directory.
- [x] `.github/workflows/build-windows.yml` exists and builds the Windows artifact using `make build-windows`.
- [x] The GitHub Actions workflow runs `go test ./...` before building the artifact.
- [x] `.github/workflows/release-windows.yml` exists and triggers on `v*.*.*` tags.
- [x] The release workflow creates a GitHub Release and attaches `dist/lol-cli.exe`.
