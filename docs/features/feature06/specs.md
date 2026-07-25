# Feature 06 Specification: Usage Examples and Overlay Documentation

## 1. Objective
Provide clear, practical documentation and code examples in the `README.md` to demonstrate how third-party developers can integrate the SDK into their own applications, specifically focusing on a game overlay use case.

## 2. Requirements

### 2.1 README Restructuring
- Add a new top-level section: `## Examples & Use Cases`.
- Add a subsection: `### Building a Game Overlay`.

### 2.2 Overlay Code Example
- Provide a fully self-contained Go code snippet in the README demonstrating:
  1. Instantiating the SDK client.
  2. A polling loop (e.g., using `time.Ticker`) to fetch data safely without spamming the local server.
  3. Fetching `GetActivePlayerName()` once at startup.
  4. Fetching `GetGameStats()` and `GetEventData()` periodically.
  5. Printing the extracted data to standard output (simulating data being sent to a UI layer like Wails or a web socket).
- Keep the snippet in sync with a real, compilable example file under `examples/overlay/main.go`.

### 2.3 Best Practices Documentation
- Document the recommended polling rate (e.g., maximum of 1 request per second or as recommended by Riot) to prevent performance degradation in the LoL Client.
- Add a brief explanation of handling the TLS certificate (e.g., `InsecureSkipVerify` necessity).
- Mention graceful shutdown (handling `ctx.Done()` / `os.Signal`) so the polling loop exits cleanly.

## 3. Out of Scope
- Developing an actual, runnable overlay application (e.g., a Wails/Electron app). Only the integration snippet is required.

## 4. Acceptance Criteria
- [x] `README.md` contains the new "Examples & Use Cases" section with a "Building a Game Overlay" subsection.
- [x] The provided Go code example is syntactically correct, compiles, and is kept in sync with `examples/overlay/main.go`.
- [x] Best practices regarding polling rates, TLS, and graceful shutdown are documented.