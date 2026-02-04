package rules

import (
	"fmt"
	"time"

	"github.com/bspippi1337/fastfuzzer-ng/internal/actions"
	"github.com/bspippi1337/fastfuzzer-ng/internal/state"
)

type Severity int

const (
	SevInfo Severity = iota
	SevWarn
	SevError
)

type Diagnosis struct {
	Name     string
	Severity Severity
	Message  string
	Actions  []actions.Action
	When     time.Time
}

type Rule interface {
	Name() string
	Match(s state.DeviceState) *Diagnosis
}

// Default rule-set: light-touch, no destructive actions.
func DefaultRules() []Rule {
	return []Rule{
		RuleFastbootNudge{},
		RuleLowBattery{},
		RuleUnknownMode{},
	}
}

type RuleUnknownMode struct{}

func (r RuleUnknownMode) Name() string { return "unknown_mode" }
func (r RuleUnknownMode) Match(s state.DeviceState) *Diagnosis {
	if s.Mode != state.ModeUnknown {
		return nil
	}
	return &Diagnosis{
		Name:     r.Name(),
		Severity: SevError,
		Message:  "Device mode is unknown (no adb/fastboot session detected). Check drivers/cable or power.",
		Actions:  nil,
		When:     time.Now(),
	}
}

type RuleFastbootNudge struct{}

func (r RuleFastbootNudge) Name() string { return "fastboot_detected" }
func (r RuleFastbootNudge) Match(s state.DeviceState) *Diagnosis {
	if s.Mode != state.ModeFastboot {
		return nil
	}
	return &Diagnosis{
		Name:     r.Name(),
		Severity: SevWarn,
		Message:  "Device is in fastboot mode. You can reboot to system or go to recovery.",
		Actions: []actions.Action{
			actions.SetMode{Target: state.ModeADB},
			actions.SetMode{Target: state.ModeRecovery},
		},
		When: time.Now(),
	}
}

type RuleLowBattery struct{}

func (r RuleLowBattery) Name() string { return "low_battery" }
func (r RuleLowBattery) Match(s state.DeviceState) *Diagnosis {
	if s.Battery >= 0 && s.Battery < 10 {
		return &Diagnosis{
			Name:     r.Name(),
			Severity: SevWarn,
			Message:  fmt.Sprintf("Battery is low (%d%%). Prefer charging before heavy operations.", s.Battery),
			Actions:  nil,
			When:     time.Now(),
		}
	}
	return nil
}
