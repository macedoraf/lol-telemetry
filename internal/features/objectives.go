package features

import (
	"fmt"
	"strings"

	"lol-telemetry/internal/types"
)

type objectivesTransformer struct{}

func (objectivesTransformer) Name() string { return "objectives" }

func (objectivesTransformer) Transform(w Window, fv *types.FeatureVector) {
	last, ok := w.Last()
	if !ok {
		return
	}
	active := findActive(last)
	if active == nil {
		return
	}
	allyTeam := active.Team

	for _, e := range w.Events() {
		switch e.EventName {
		case "DragonKill":
			if e.KillerName == "" {
				continue
			}
			team := teamOf(last.Players, e.KillerName)
			if team == "" {
				continue
			}
			obj := objectiveFor(fv, allyTeam, team)
			tf := teamFeaturesFor(fv, allyTeam, team)
			obj.Dragons++
			if e.Stolen == "True" {
				obj.Steals++
				tf.Spikes = append(tf.Spikes, fmt.Sprintf("%s dragon stolen by %s @%s", e.DragonType, team, formatGameTime(e.EventTime)))
			} else {
				tf.Spikes = append(tf.Spikes, fmt.Sprintf("%s dragon by %s @%s", e.DragonType, team, formatGameTime(e.EventTime)))
			}
		case "BaronKill":
			if e.KillerName == "" {
				continue
			}
			team := teamOf(last.Players, e.KillerName)
			if team == "" {
				continue
			}
			obj := objectiveFor(fv, allyTeam, team)
			tf := teamFeaturesFor(fv, allyTeam, team)
			obj.Barons++
			if e.Stolen == "True" {
				obj.Steals++
				tf.Spikes = append(tf.Spikes, fmt.Sprintf("Baron stolen by %s @%s", team, formatGameTime(e.EventTime)))
			} else {
				tf.Spikes = append(tf.Spikes, fmt.Sprintf("Baron by %s @%s", team, formatGameTime(e.EventTime)))
			}
		case "HeraldKill":
			if e.KillerName == "" {
				continue
			}
			team := teamOf(last.Players, e.KillerName)
			if team == "" {
				continue
			}
			obj := objectiveFor(fv, allyTeam, team)
			tf := teamFeaturesFor(fv, allyTeam, team)
			obj.Heralds++
			if e.Stolen == "True" {
				obj.Steals++
				tf.Spikes = append(tf.Spikes, fmt.Sprintf("Herald stolen by %s @%s", team, formatGameTime(e.EventTime)))
			} else {
				tf.Spikes = append(tf.Spikes, fmt.Sprintf("Herald by %s @%s", team, formatGameTime(e.EventTime)))
			}
		case "TurretKilled":
			team := structureOwnerTeam(e.TurretKilled)
			if team == "" {
				continue
			}
			creditTeam := oppositeTeam(team)
			objectiveFor(fv, allyTeam, creditTeam).Towers++
			teamFeaturesFor(fv, allyTeam, creditTeam).Spikes = append(
				teamFeaturesFor(fv, allyTeam, creditTeam).Spikes,
				fmt.Sprintf("Turret killed by %s @%s", creditTeam, formatGameTime(e.EventTime)),
			)
		case "InhibKilled":
			team := structureOwnerTeam(e.InhibKilled)
			if team == "" {
				continue
			}
			creditTeam := oppositeTeam(team)
			objectiveFor(fv, allyTeam, creditTeam).Inhibs++
			teamFeaturesFor(fv, allyTeam, creditTeam).Spikes = append(
				teamFeaturesFor(fv, allyTeam, creditTeam).Spikes,
				fmt.Sprintf("Inhibitor killed by %s @%s", creditTeam, formatGameTime(e.EventTime)),
			)
		case "FirstBrick":
			team := teamOf(last.Players, e.KillerName)
			if team == "" {
				continue
			}
			teamFeaturesFor(fv, allyTeam, team).Spikes = append(
				teamFeaturesFor(fv, allyTeam, team).Spikes,
				fmt.Sprintf("First turret by %s @%s", team, formatGameTime(e.EventTime)),
			)
		case "Ace":
			team := e.AcingTeam
			if team != allyTeam && team != oppositeTeam(allyTeam) {
				continue
			}
			teamFeaturesFor(fv, allyTeam, team).Spikes = append(
				teamFeaturesFor(fv, allyTeam, team).Spikes,
				fmt.Sprintf("Ace by %s @%s", team, formatGameTime(e.EventTime)),
			)
		case "Multikill":
			team := teamOf(last.Players, e.KillerName)
			if team == "" {
				continue
			}
			teamFeaturesFor(fv, allyTeam, team).Spikes = append(
				teamFeaturesFor(fv, allyTeam, team).Spikes,
				fmt.Sprintf("%s multikill x%d @%s", e.KillerName, e.KillStreak, formatGameTime(e.EventTime)),
			)
		case "ChampionKill":
			if e.KillerName == "" {
				continue
			}
			team := teamOf(last.Players, e.KillerName)
			if team == "" {
				continue
			}
			if last.GameTime-e.EventTime <= 60 {
				if team == allyTeam {
					fv.Team.Kills1m++
				} else {
					fv.Enemy.Kills1m++
				}
			}
		}
	}

	if fv.Team.Objectives.Dragons == 3 {
		fv.Team.Objectives.SoulPoint = true
	}
	if fv.Enemy.Objectives.Dragons == 3 {
		fv.Enemy.Objectives.SoulPoint = true
	}

	for i := range last.Players {
		p := &last.Players[i]
		if !p.IsDead || p.RespawnTimer <= 0 {
			continue
		}
		tf := teamFeaturesFor(fv, allyTeam, p.Team)
		tf.DeadNow++
		if p.RespawnTimer > tf.MaxRespawnSec {
			tf.MaxRespawnSec = p.RespawnTimer
		}
	}
}

// structureOwnerTeam parses Turret_T1_* / Barracks_T1_* as ORDER and _T2_* as CHAOS.
func structureOwnerTeam(id string) string {
	if strings.Contains(id, "_T1_") {
		return "ORDER"
	}
	if strings.Contains(id, "_T2_") {
		return "CHAOS"
	}
	return ""
}
