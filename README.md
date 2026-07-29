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

### Runtime Configuration API

The daemon exposes `GET /api/config` and `PATCH /api/config` to inspect and modify the judge language, hook enable states, and hook parameters at runtime.

```bash
curl http://localhost:8080/api/config
curl -X PATCH http://localhost:8080/api/config \
  -H 'Content-Type: application/json' \
  -d '{"judge":{"language":"pt-BR"},"hooks":[{"name":"periodic-5min","enabled":false}]}'
```

### Judge Configuration (Bring Your Own Key)
The optional LLM Judge is provider-pluggable and speaks the OpenAI chat-completions format. Select the provider with `JUDGE_PROVIDER` and configure the corresponding variables.

| Variable | Required | Description |
| :--- | :--- | :--- |
| `JUDGE_PROVIDER` | No | `openrouter` (default), `deepinfra`, or `openai`. |
| `OPENROUTER_API_KEY` | For `openrouter` | Your OpenRouter API key. |
| `OPENROUTER_MODEL` | No | Model to use. Defaults to `openai/gpt-4o-mini`. |
| `DEEPINFRA_BASE_URL` | No | DeepInfra API base URL. Defaults to `https://api.deepinfra.com/v1/openai`. |
| `DEEPINFRA_API_KEY` | For `deepinfra` | Your DeepInfra API token. |
| `DEEPINFRA_MODEL` | No | Model to use. Defaults to `deepseek-ai/DeepSeek-V3`. |
| `OPENAI_BASE_URL` | No | OpenAI API base URL. Defaults to `https://api.openai.com/v1`. |
| `OPENAI_API_KEY` | For `openai` | Your OpenAI API key. |
| `OPENAI_MODEL` | No | Model to use. Defaults to `gpt-4o-mini`. |
| `JUDGE_LANGUAGE` | No | Language for Judge advice tips. One of `en`, `pt-BR`, `es`. Defaults to `en`. |
| `LOL_RECORD_ENABLED` | No | Persist raw Live Client Data API snapshots to disk as append-only JSONL. Defaults to `false`. |
| `LOL_FEATURES_ENABLED` | No | Compute time-series features (gold/min, XP/min, objectives, death timers, matchup diffs) and send them to the Judge. Writes `features.jsonl` when recording is enabled. Defaults to `false`. |
| `LOL_RECORDINGS_DIR` | No | Directory for recorded sessions. Defaults to `./recordings`. |

Example:
```bash
export OPENROUTER_API_KEY="sk-..."
export OPENROUTER_MODEL="anthropic/claude-3.5-sonnet"
go run ./cmd/lol-daemon
```

When the key is absent, the Judge loop is disabled.

### Runtime Judge Prompt Editing

You can change the Judge system prompt at runtime without restarting the daemon:

```bash
curl -X PATCH http://localhost:8080/api/config \
  -H 'Content-Type: application/json' \
  -d '{"judge":{"prompt":"Focus only on objective control and rotations."}}'
```

Rules:

* The prompt must be non-empty and ≤ 4000 characters.
* `{"judge":{"prompt":""}}` restores the default prompt.
* The language directive from `JUDGE_LANGUAGE` is always appended to the effective prompt.

From the test TUI (`go run ./cmd/lol-cli`), open the **Config** tab and type `/prompt` followed by Enter to edit the prompt in `$EDITOR`.

### Telemetry Recording

You can persist every raw `/allgamedata` snapshot to disk for later analysis or for training future Judge features:

```bash
export LOL_RECORD_ENABLED=true
export LOL_RECORDINGS_DIR=./recordings

go run ./cmd/lol-daemon
```

Each match gets its own directory under `<recordingsDir>/<sessionID>/`:

```
recordings/
  20260727-143012-a1b2c3/
    telemetry.jsonl
```

Each line is a self-contained JSON object:

```json
{"v":1,"type":"telemetry","ts":1750000000000,"session":"20260727-143012-a1b2c3","gameTime":612.4,"data":{...allgamedata bruto...}}
```

Read it back with `jq`:

```bash
jq -c 'select(.gameTime > 300)' recordings/20260727-143012-a1b2c3/telemetry.jsonl
```

When the Judge fires, a tip is also recorded in the same session directory:

```
recordings/
  20260727-143012-a1b2c3/
    telemetry.jsonl
    tips.jsonl
```

Join tips to telemetry by `(session, gameTime)`:

```bash
SESSION=20260727-143012-a1b2c3
TIP_TIME=612.4
jq -c "select(.session==\"$SESSION\" and (.gameTime - $TIP_TIME | . * . < 1))" recordings/$SESSION/telemetry.jsonl
```

Recording is fully asynchronous: if disk I/O falls behind, frames are dropped and counted, but the live poll loop is never blocked.

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
