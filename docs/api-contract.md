# API Contract & Mocks (`api-contract.md`)

## 1. Overview
The `lol-telemetry` application communicates exclusively with the local **Live Client Data API** provided by the League of Legends game client. 

*   **Base URL:** `https://127.0.0.1:2999/liveclientdata`
*   **Protocol:** HTTPS (Self-signed certificate, requires skipping SSL validation in the HTTP client).
*   **Authentication:** None (Local API).

## 2. Core Endpoint Used in MVP
*   **Endpoint:** `/allgamedata`
*   **Full URL:** `https://127.0.0.1:2999/liveclientdata/allgamedata`
*   **Description:** Returns all available game data in a single JSON payload, including active player metadata, full player list, game time, and event list.

## 3. Mock Strategy & Test Data
To enable offline development, unit testing, and TDD without requiring an active League of Legends match, raw JSON responses must be stored locally.

*   **Mock File Path:** `testdata/mocks/allgamedata.json`
*   **Rule for AI/Developers:** Any DTO (Data Transfer Object) in `pkg/riotclient` MUST map strictly to the structure present in `testdata/mocks/allgamedata.json`. Do not infer fields outside of this contract.