package engine

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bspippi1337/fastfuzzer-ng/internal/actions"
	"github.com/bspippi1337/fastfuzzer-ng/internal/rules"
	"github.com/bspippi1337/fastfuzzer-ng/internal/scorer"
	"github.com/bspippi1337/fastfuzzer-ng/internal/state"
	"github.com/bspippi1337/fastfuzzer-ng/internal/transport"
)

type Engine struct {
	T       transport.Transport
	Rules   []rules.Rule
	Scorer  *scorer.ScoreBoard
	Auto    bool
	Verbose bool
}

func New(t transport.Transport) *Engine {
	return &Engine{
		T:      t,
		Rules:  rules.DefaultRules(),
		Scorer: scorer.New(),
		Auto:   true,
	}
}

func (e *Engine) CollectState(ctx context.Context, serial string) state.DeviceState {
	s := state.DeviceState{
		Serial:    serial,
		Mode:      state.ModeUnknown,
		Battery:   -1,
		Props:     map[string]string{},
		Errors:    nil,
		Timestamp: time.Now(),
	}

	// Detect adb
	if out, err := e.T.ADB(ctx, serial, "get-state"); err == nil {
		out = strings.TrimSpace(out)
		s.Mode = state.ModeADB
		if strings.Contains(strings.ToLower(out), "recovery") {
			s.Mode = state.ModeRecovery
		}
		// Try to determine recovery mode via property
		if prop, err := e.T.ADB(ctx, serial, "shell", "getprop", "ro.bootmode"); err == nil {
			p := strings.ToLower(strings.TrimSpace(prop))
			s.Props["ro.bootmode"] = p
			if strings.Contains(p, "recovery") {
				s.Mode = state.ModeRecovery
			}
			if strings.Contains(p, "safe") {
				s.Mode = state.ModeSafeMode
			}
		}
		// Battery level
		if batt, err := e.T.ADB(ctx, serial, "shell", "dumpsys", "battery"); err == nil {
			s.Battery = parseBattery(batt)
		}
		return s
	} else {
		s.Errors = append(s.Errors, err.Error())
	}

	// Detect fastboot
	if out, err := e.T.Fastboot(ctx, serial, "getvar", "product"); err == nil {
		s.Mode = state.ModeFastboot
		// output often includes "product: xxx"
		if strings.TrimSpace(out) != "" {
			s.Props["fastboot_product"] = strings.TrimSpace(out)
		}
		return s
	} else {
		s.Errors = append(s.Errors, err.Error())
	}

	return s
}

func (e *Engine) Diagnose(s state.DeviceState) []rules.Diagnosis {
	var diags []rules.Diagnosis
	for _, r := range e.Rules {
		if d := r.Match(s); d != nil {
			diags = append(diags, *d)
		}
	}
	// Sort by severity, then by number of actions
	sortDiags(diags)
	return diags
}

func (e *Engine) AutoFix(ctx context.Context, serial string, s state.DeviceState) (rules.Diagnosis, actions.Action, error) {
	diags := e.Diagnose(s)
	if len(diags) == 0 {
		return rules.Diagnosis{Name: "ok", Severity: rules.SevInfo, Message: "No issues detected", When: time.Now()}, nil, nil
	}

	pick := diags[0]
	if len(pick.Actions) == 0 {
		return pick, nil, nil
	}

	// Choose an action
	var chosen actions.Action
	if e.Auto {
		keys := make([]string, 0, len(pick.Actions))
		for _, a := range pick.Actions {
			keys = append(keys, a.Name())
		}
		idx := e.Scorer.Pick(keys, func(i int) float64 {
			// Prefer cheaper actions slightly (higher weight).
			c := pick.Actions[i].Cost().Seconds()
			if c <= 0 {
				return 1
			}
			// 1/(1+c) keeps weights in (0,1]
			return 1.0 / (1.0 + c)
		})
		chosen = pick.Actions[idx]
	} else {
		chosen = pick.Actions[0]
	}

	if !chosen.CanApply(s) {
		return pick, chosen, fmt.Errorf("action %s cannot apply in mode %s", chosen.Name(), s.Mode)
	}

	err := chosen.Apply(ctx, e.T, serial)
	e.Scorer.Update(chosen.Name(), err == nil)
	_ = e.Scorer.Save(scorer.DefaultPath())
	return pick, chosen, err
}

var reBattery = regexp.MustCompile(`(?m)^\s*level\s*:\s*(\d+)\s*$`)

func parseBattery(out string) int {
	m := reBattery.FindStringSubmatch(out)
	if len(m) != 2 {
		return -1
	}
	var v int
	_, _ = fmt.Sscanf(m[1], "%d", &v)
	if v < 0 || v > 100 {
		return -1
	}
	return v
}

func sortDiags(diags []rules.Diagnosis) {
	// severity desc: error > warn > info
	sevRank := func(s rules.Severity) int {
		switch s {
		case rules.SevError:
			return 3
		case rules.SevWarn:
			return 2
		default:
			return 1
		}
	}
	for i := 0; i < len(diags); i++ {
		for j := i + 1; j < len(diags); j++ {
			if sevRank(diags[j].Severity) > sevRank(diags[i].Severity) {
				diags[i], diags[j] = diags[j], diags[i]
				continue
			}
			if sevRank(diags[j].Severity) == sevRank(diags[i].Severity) {
				if len(diags[j].Actions) > len(diags[i].Actions) {
					diags[i], diags[j] = diags[j], diags[i]
				}
			}
		}
	}
}
