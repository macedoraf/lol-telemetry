# Feature 07 Specification: Interactive CLI Monitor and Debugger

## 1. Objective
Improove an interactive Command Line Interface (CLI) tool that continuously monitors the League of Legends Live Client Data API. The CLI must display real-time connection status, allow developers to navigate between different API routes to inspect raw JSON data for debugging, and remain completely resilient to connection drops or API errors.

## 2. Requirements

### 2.1 UI Layout & Connection Status
- Implement a Terminal User Interface (TUI) (e.g., using Charmbracelet's Bubble Tea).
- **Header/Status Bar**: Must display the current API connection status.
  - When the API is reachable: Show a Green indicator with the text "ONLINE".
  - When the API is unreachable (game closed, loading screen): Show a Red indicator with the text "OFFLINE".
- **Sidebar/Menu**: A list of available API routes (e.g., `allgamedata`, `activeplayer`, `playerlist`, `eventdata`, etc.).
- **Main Viewport**: A scrollable area displaying the raw, pretty-printed `.json` response of the currently selected route.

### 2.2 Short Polling Engine
- Implement a background polling mechanism that ticks every `1s` (1000ms).
- The polling should update the "Online/Offline" status.
- To save resources, the polling should only fetch the data for the **currently selected route** in the UI, updating the main viewport with the fresh JSON payload automatically every second.

### 2.3 Interactive Navigation
- Users must be able to use the keyboard (e.g., `Up/Down` arrows, `j/k`, or `Tab`) to navigate between the different routes listed in the sidebar.
- Changing the selected route must immediately trigger a fetch for that specific route and update the main viewport with the new JSON structure.
- If the JSON is larger than the terminal height, the main viewport must be scrollable.

### 2.4 Resilience & Error Handling
- The CLI **must not crash or panic** under any circumstances (e.g., connection refused, EOF, invalid JSON, 404 Not Found).
- If an HTTP request fails or times out:
  - The status bar should update to "OFFLINE".
  - The main viewport should elegantly display the error message (e.g., `"Error: Connection refused - Please ensure the League of Legends client is running in-game."`) instead of crashing the application.
  - The polling loop must continue attempting to reconnect every 1s.

### 2.5 User Controls & Graceful Exit
- The CLI must be easy to close. Pressing `q`, `Esc`, or `Ctrl+C` must instantly and gracefully terminate the application and stop all background polling goroutines.

## 3. Out of Scope
- Advanced JSON filtering (e.g., `jq` style queries) inside the CLI.
- Saving/exporting the JSON payloads to local files.
- Modifying or sending POST requests to the API (this is a read-only telemetry debugger).

## 4. Acceptance Criteria
- [x] The CLI starts up and displays the TUI correctly.
- [x] Connection status indicator dynamically changes to Green ("ONLINE") when the API is available and Red ("OFFLINE") when it drops.
- [x] Polling occurs every 1 second, updating the displayed JSON in real-time.
- [x] The user can navigate between different routes and see the specific raw JSON for each.
- [x] If the API becomes unreachable, the CLI survives, shows an error in the viewport, and recovers automatically when the API returns.
- [x] The smoke test still validates the non-TUI path against the existing mock.
- [x] Pressing `q`, `Esc`, or `Ctrl+C` closes the CLI safely without orphaned processes.