# lol-telemetry

A real-time telemetry SDK and CLI for League of Legends, built in Go. Fully compliant with Riot Games Vanguard anti-cheat policies (zero memory reading).

## Quickstart

### Prerequisites
* Go 1.22+ installed.

### Running in Mock Mode (Offline Development)
Test the TUI dashboard instantly without an active game client:
```bash
go run cmd/lol-cli/main.go --mock testdata/mocks/allgamedata.json
```

## Testing with Docker Compose

A containerized test stack is provided for reproducible builds and integration tests without installing a local Go toolchain or running the League of Legends client.

### Prerequisites
* Docker and Docker Compose installed.

### Run the Full Test Suite
```bash
docker compose up --build
```

This command:
1. Builds the Go test runner image.
2. Builds and starts the mock Live Client Data API on `https://localhost:2999`.
3. Runs unit tests (`go test ./...`).
4. Runs the integration test against the mock API.
5. Runs the CLI smoke test in mock mode (`--mock --smoke-test`) without requiring an interactive terminal.

### Start the Mock API for Ad-Hoc Testing
```bash
docker compose up mock-api -d
```

The mock API exposes:
* `https://localhost:2999/liveclientdata/allgamedata` (self-signed certificate)
* `http://localhost:8080/health` (health check endpoint)

You can point the `riotclient` integration tests at it by setting `MOCK_API_URL=https://mock-api:2999/liveclientdata` from within the compose network.

## Build a Windows Artifact

The CLI can be cross-compiled to a Windows executable (`lol-cli.exe`) on any platform with Go installed.

### Manual cross-compilation
```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o dist/lol-cli.exe ./cmd/lol-cli
```

### Using Make
```bash
make build-windows
```

This creates `dist/lol-cli.exe`. The produced binary is statically linked and does not require CGO.

### Verify the artifact
```bash
./scripts/build-windows.sh
```

### Clean build artifacts
```bash
make clean
```

## Releasing

A GitHub Actions workflow automatically builds and releases the Windows artifact.

### On every push to `main`
The `.github/workflows/release-windows.yml` workflow runs tests, builds `dist/lol-cli.exe`, creates a lightweight tag (e.g., `main-20260725-abc123`), and publishes a GitHub prerelease with the artifact attached.

### On semantic version tags
Push a tag following [SemVer](https://semver.org/) to create a stable release:
```bash
git tag v1.0.0
git push origin v1.0.0
```

The workflow uses the tag name as the release title and attaches `dist/lol-cli.exe` as a release asset.

### Pull request CI
The `.github/workflows/build-windows.yml` workflow runs on every pull request to `main`, ensuring tests pass and the Windows artifact builds successfully.
