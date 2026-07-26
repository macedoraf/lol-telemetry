// Package main is the CLI entrypoint. It wires the riotclient, the feature menu,
// the SDK route debugger, and the Judge tips panel together with zero business logic.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"lol-telemetry/internal/config"
	"lol-telemetry/internal/hooks"
	"lol-telemetry/internal/judge"
	"lol-telemetry/internal/judge/openrouter"
	"lol-telemetry/internal/judge/payload"
	"lol-telemetry/internal/menu"
	"lol-telemetry/internal/orchestrator"
	"lol-telemetry/internal/processor"
	"lol-telemetry/internal/renderer"
	"lol-telemetry/internal/tips"
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

	cfg := config.LoadEnvConfig()
	app := newAppModel(client, cfg)
	program := tea.NewProgram(app, tea.WithAltScreen())

	startJudgeLoop(program, client, cfg)

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

// appModel is the top-level Bubble Tea model that routes between the feature
// menu, the SDK route debugger, and the Judge tips panel.
type appModel struct {
	activeView string // "menu", "routes", "tips"
	menu       menu.Model
	debugger   renderer.Model
	tips       tips.Model
}

func newAppModel(client *riotclient.Client, cfg config.EnvConfig) appModel {
	return appModel{
		activeView: "menu",
		menu:       menu.NewModel(),
		debugger:   renderer.NewDebugger(client),
		tips:       tips.NewModel(cfg),
	}
}

func (m appModel) Init() tea.Cmd {
	return m.menu.Init()
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		menuModel, _ := m.menu.Update(msg)
		debuggerModel, _ := m.debugger.Update(msg)
		tipsModel, _ := m.tips.Update(msg)
		m.menu = menuModel.(menu.Model)
		m.debugger = debuggerModel.(renderer.Model)
		m.tips = tipsModel.(tips.Model)
		return m, nil

	case menu.SelectMsg:
		switch msg.Choice {
		case menu.ChoiceRoutes:
			m.activeView = "routes"
		case menu.ChoiceTips:
			m.activeView = "tips"
		}
		return m, nil

	case tips.BackMsg:
		m.activeView = "menu"
		return m, nil

	case renderer.AdviceMsg:
		debuggerModel, _ := m.debugger.Update(msg)
		tipsModel, _ := m.tips.Update(tips.UpdateAdviceMsg{Advice: msg.Advice})
		m.debugger = debuggerModel.(renderer.Model)
		m.tips = tipsModel.(tips.Model)
		return m, nil
	}

	switch m.activeView {
	case "menu":
		newMenu, cmd := m.menu.Update(msg)
		m.menu = newMenu.(menu.Model)
		cmds = append(cmds, cmd)
	case "routes":
		newDebugger, cmd := m.debugger.Update(msg)
		m.debugger = newDebugger.(renderer.Model)
		cmds = append(cmds, cmd)
	case "tips":
		newTips, cmd := m.tips.Update(msg)
		m.tips = newTips.(tips.Model)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m appModel) View() string {
	switch m.activeView {
	case "routes":
		return m.debugger.View()
	case "tips":
		return m.tips.View()
	default:
		return m.menu.View()
	}
}

func startJudgeLoop(program *tea.Program, client *riotclient.Client, cfg config.EnvConfig) {
	if !cfg.Enabled() {
		return
	}

	llmClient := openrouter.NewClientWithModel(cfg.APIKey, cfg.Model)
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
