# lol-telemetry

A real-time telemetry SDK and CLI for League of Legends, built in Go. Fully compliant with Riot Games Vanguard anti-cheat policies (zero memory reading).

## Quickstart

### Prerequisites
* Go 1.22+ installed.

### Running locally
Start the WebSocket daemon first (it reads the LoL Live Client Data API), then connect the TUI:

```bash
# Terminal 1
go run ./cmd/lol-daemon

# Terminal 2
go run ./cmd/lol-cli
```

### Judge Configuration (Bring Your Own Key)
The optional LLM Judge is powered by [OpenRouter](https://openrouter.ai) and configured through environment variables:

| Variable | Required | Description |
| :--- | :--- | :--- |
| `OPENROUTER_API_KEY` | Yes | Your OpenRouter API key. |
| `OPENROUTER_MODEL` | No | Model to use. Defaults to `openai/gpt-4o-mini`. |

Example:
```bash
export OPENROUTER_API_KEY="sk-..."
export OPENROUTER_MODEL="anthropic/claude-3.5-sonnet"
go run ./cmd/lol-daemon
```

When the key is absent, the Judge loop is disabled.

## Examples & Use Cases

### Building a Game Overlay

The SDK can be used directly in your own Go application to power a real-time overlay, a web dashboard, or any other tool that needs live telemetry. The example below polls the Live Client Data API once per second and prints the latest game stats and event. In a real overlay, you would push this data to a UI layer (Wails, Electron, a web socket, etc.) instead of printing to stdout.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lol-telemetry/pkg/riotclient"
)

func main() {
	// The SDK client is preconfigured with InsecureSkipVerify so it can talk
	// to the self-signed certificate served by the LoL client on 127.0.0.1:2999.
	client := riotclient.NewClient()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	name, err := client.GetActivePlayerName()
	if err != nil {
		log.Printf("Unable to fetch active player name (is a game running?): %v", err)
	} else {
		fmt.Printf("Active player: %s\n", name)
	}

	// Poll the local server at a conservative rate. Riot recommends keeping
	// Live Client Data API traffic low to avoid impacting the LoL client.
	// 1 request per second is a reasonable maximum for a lightweight overlay.
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Shutting down overlay...")
			return
		case <-ticker.C:
			stats, err := client.GetGameStats()
			if err != nil {
				log.Printf("game stats error: %v", err)
				continue
			}
			fmt.Printf("[%s] Game time: %.2f\n", stats.GameMode, stats.GameTime)

			events, err := client.GetEventData()
			if err != nil {
				log.Printf("event data error: %v", err)
				continue
			}
			if len(events.Events) > 0 {
				last := events.Events[len(events.Events)-1]
				fmt.Printf("Last event: %s (t=%.2f)\n", last.EventName, last.EventTime)
			}
		}
	}
}
```

A fully compilable version of this example is available in [`examples/overlay/main.go`](examples/overlay/main.go).

### Best practices

* **Polling rate:** Keep the request rate low. A maximum of **1 request per second** is recommended for a lightweight overlay. Higher rates can degrade the performance of the League of Legends client.
* **TLS certificate:** The LoL client serves the Live Client Data API over HTTPS with a self-signed certificate. The SDK client is preconfigured with `InsecureSkipVerify: true` to connect without manual certificate management.
* **Graceful shutdown:** Use a `context.Context` driven by `os.Interrupt`/`SIGTERM` so the polling loop exits cleanly when the user closes the overlay.
* **Error handling:** Always handle API errors. The LoL client only exposes the API while a game is running, so transient failures are expected between matches.

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

The project ships two Windows executables:

- `lol-daemon.exe` — background WebSocket daemon that reads the LoL Live Client Data API.
- `lol-cli.exe` — interactive TUI / raw-mode client that connects to the daemon.

Both must be present. Running `lol-cli.exe` without `lol-daemon.exe` first will fail with a WebSocket connection error.

### Manual cross-compilation
```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o dist/lol-daemon.exe ./cmd/lol-daemon
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o dist/lol-cli.exe ./cmd/lol-cli
```

### Using Make
```bash
make build-windows
```

This creates `dist/lol-daemon.exe` and `dist/lol-cli.exe`. Both binaries are statically linked and do not require CGO.

### Verify the artifact
```bash
./scripts/build-windows.sh
```

### Clean build artifacts
```bash
make clean
```

## Releasing

A GitHub Actions workflow automatically builds and releases the Windows artifacts.

### On every push to `main`
The `.github/workflows/release-windows.yml` workflow runs tests, builds `dist/lol-daemon.exe` and `dist/lol-cli.exe`, creates a lightweight tag (e.g., `main-20260725-abc123`), and publishes a GitHub prerelease with both executables attached.

### On semantic version tags
Push a tag following [SemVer](https://semver.org/) to create a stable release:
```bash
git tag v1.0.0
git push origin v1.0.0
```

The workflow uses the tag name as the release title and attaches `dist/lol-daemon.exe` and `dist/lol-cli.exe` as release assets.

### Pull request CI
The `.github/workflows/build-windows.yml` workflow runs on every pull request to `main`, ensuring tests pass and both Windows artifacts build successfully. Both `lol-daemon.exe` and `lol-cli.exe` are uploaded as separate artifacts.
