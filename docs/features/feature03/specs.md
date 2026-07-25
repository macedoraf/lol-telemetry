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

### 2.5 Documentation Updates
- Update `README.md` with both the manual command and the `make build-windows` shortcut.
- Keep existing Docker Compose and mock-mode instructions intact.
- Optionally add a note that the produced Windows artifact runs in a terminal and supports both live mode and `--mock` mode.

## 3. Out of Scope
- Windows installer (`.msi`) or packaged app.
- GitHub Actions CI/CD pipeline for automated releases.
- Code signing for the Windows executable.
- 32-bit (`GOARCH=386`) or ARM builds.

## 4. Acceptance Criteria
- [ ] `README.md` contains a "Build a Windows Artifact" section with both the manual `go build` command and the `make build-windows` shortcut.
- [ ] `Makefile` exists with `build-windows`, `build`, `clean`, and `help` targets.
- [ ] `scripts/build-windows.sh` exists and produces `dist/lol-cli.exe` when executed.
- [ ] `.gitignore` ignores the `dist/` directory.
- [ ] Running `make build-windows` on a clean repository succeeds and creates a non-empty `dist/lol-cli.exe`.
- [ ] Running `make clean` removes the `dist/` directory.
