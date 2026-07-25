package processor

import (
	"testing"

	"lol-telemetry/internal/types"
	"lol-telemetry/pkg/riotclient"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name     string
		input    riotclient.AllGameData
		expected types.PlayerStats
		wantErr  bool
	}{
		{
			name: "CS/Min and GPM for 10 minute game",
			input: riotclient.AllGameData{
				ActivePlayer: riotclient.ActivePlayer{
					SummonerName: "Player",
					Level:        12,
					CurrentGold:  3000,
				},
				AllPlayers: []riotclient.AllPlayer{
					{
						SummonerName: "Player",
						ChampionName: "Ashe",
						Scores: riotclient.PlayerScores{
							CreepScore: 120,
						},
					},
				},
				GameData: riotclient.GameData{GameTime: 600},
			},
			expected: types.PlayerStats{
				SummonerName: "Player",
				ChampionName: "Ashe",
				Level:        12,
				CurrentGold:  3000,
				GameTime:     600,
				CSPerMin:     12.0,
				GPM:          300.0,
			},
			wantErr: false,
		},
		{
			name: "GPM for 5 minute game",
			input: riotclient.AllGameData{
				ActivePlayer: riotclient.ActivePlayer{
					SummonerName: "Player",
					Level:        6,
					CurrentGold:  3000,
				},
				AllPlayers: []riotclient.AllPlayer{
					{
						SummonerName: "Player",
						ChampionName: "Lux",
						Scores: riotclient.PlayerScores{
							CreepScore: 50,
						},
					},
				},
				GameData: riotclient.GameData{GameTime: 300},
			},
			expected: types.PlayerStats{
				SummonerName: "Player",
				ChampionName: "Lux",
				Level:        6,
				CurrentGold:  3000,
				GameTime:     300,
				CSPerMin:     10.0,
				GPM:          600.0,
			},
			wantErr: false,
		},
		{
			name: "zero game time returns error",
			input: riotclient.AllGameData{
				ActivePlayer: riotclient.ActivePlayer{
					CurrentGold: 1000,
				},
				GameData: riotclient.GameData{GameTime: 0},
			},
			expected: types.PlayerStats{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Calculate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Calculate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.expected {
				t.Errorf("Calculate() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}
