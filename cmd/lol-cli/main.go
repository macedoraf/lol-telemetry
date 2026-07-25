// Package main is the CLI entrypoint. It wires the riotclient and the debugger
// renderer together with zero business logic.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"lol-telemetry/internal/hooks"
	"lol-telemetry/internal/judge"
	"lol-telemetry/internal/judge/openrouter"
	"lol-telemetry/internal/judge/payload"
	"lol-telemetry/internal/orchestrator"
	"lol-telemetry/internal/processor"
	"lol-telemetry/internal/renderer"
	"lol-telemetry/pkg/riotclient"
)

func main() {
	var mockPath string
	var smokeTest bool
	flag.StringVar(&mockPath, "mock", "", "path to a local allgamedata.json mock file")
	flag.BoolVar(&smokeTest, "smoke-test", false, "run one fetch/calculate cycle and exit without the TUI")
	flag.Parse()

	if mockPath != "" {
		if _, err := loadMockFile(mockPath); err != nil {
			fmt.Fprintf(os.Stderr, "mock error: %v\n", err)
			os.Exit(1)
		}
	}

	client := riotclient.NewClient()

	if smokeTest {
		if err := runSmokeTest(client, mockPath); err != nil {
			fmt.Fprintf(os.Stderr, "smoke test error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	model := renderer.NewDebugger(client)
	program := tea.NewProgram(model, tea.WithAltScreen())

	startJudgeLoop(program, client)

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

func startJudgeLoop(program *tea.Program, client *riotclient.Client) {
	llmClient := newLLMClient()
	if llmClient == nil {
		return
	}

	reg := hooks.NewRegistry()
	reg.Register(hooks.Periodic5MinHook{})

	j := judge.NewJudge(llmClient)
	b := payload.NewBuilder()
	orch := orchestrator.NewOrchestrator(client, reg, b, j)

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				resps, err := orch.Tick(context.Background())
				if err != nil {
					continue
				}
				for _, resp := range resps {
					program.Send(renderer.AdviceMsg{Advice: resp.Advice})
				}
			}
		}
	}()
}

func newLLMClient() judge.LLMClient {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil
	}
	return openrouter.NewClient(apiKey)
}

func runSmokeTest(client *riotclient.Client, mockPath string) error {
	data, err := fetchGameData(client, mockPath)
	if err != nil {
		return err
	}
	stats, err := processor.Calculate(data)
	if err != nil {
		return err
	}
	fmt.Printf("smoke test ok: CS/Min=%.2f, GPM=%.2f, Gold=%.0f\n",
		stats.CSPerMin, stats.GPM, stats.CurrentGold)
	return nil
}

func fetchGameData(client *riotclient.Client, mockPath string) (riotclient.AllGameData, error) {
	if mockPath != "" {
		return loadMockFile(mockPath)
	}
	return client.GetGameData()
}

func loadMockFile(path string) (riotclient.AllGameData, error) {
	var data riotclient.AllGameData
	content, err := os.ReadFile(path)
	if err != nil {
		return data, fmt.Errorf("read mock file: %w", err)
	}
	if err := json.Unmarshal(content, &data); err != nil {
		return data, fmt.Errorf("parse mock file: %w", err)
	}
	return data, nil
}
