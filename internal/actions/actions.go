package actions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bspippi1337/fastfuzzer-ng/internal/state"
	"github.com/bspippi1337/fastfuzzer-ng/internal/transport"
)

type Action interface {
	Name() string
	CanApply(s state.DeviceState) bool
	Apply(ctx context.Context, t transport.Transport, serial string) error
	Cost() time.Duration
}

type SetMode struct {
	Target state.Mode
}

func (a SetMode) Name() string        { return fmt.Sprintf("set_mode_%s", a.Target) }
func (a SetMode) Cost() time.Duration { return 10 * time.Second }

func (a SetMode) CanApply(s state.DeviceState) bool {
	// We can always attempt a mode transition, but we try to be sensible:
	// - safemode needs adb
	if a.Target == state.ModeSafeMode {
		return s.Mode == state.ModeADB || s.Mode == state.ModeRecovery
	}
	return s.Mode != state.ModeUnknown
}

func (a SetMode) Apply(ctx context.Context, t transport.Transport, serial string) error {
	switch a.Target {
	case state.ModeADB:
		// Best effort: from fastboot -> reboot
		_, err := t.Fastboot(ctx, serial, "reboot")
		if err == nil {
			return nil
		}
		// From adb/recovery: normal reboot
		_, err2 := t.ADB(ctx, serial, "reboot")
		return err2
	case state.ModeFastboot:
		_, err := t.ADB(ctx, serial, "reboot", "bootloader")
		return err
	case state.ModeRecovery:
		// ADB can reboot recovery; fastboot might not have universal recovery reboot
		if _, err := t.ADB(ctx, serial, "reboot", "recovery"); err == nil {
			return nil
		}
		// Try fastboot reboot recovery if supported
		_, err := t.Fastboot(ctx, serial, "reboot", "recovery")
		return err
	case state.ModeSafeMode:
		// Requires an ADB session. Mark safemode then reboot.
		// Note: property key mirrors your README note.
		if _, err := t.ADB(ctx, serial, "shell", "setprop", "persist.sys.safemode", "1"); err != nil {
			return err
		}
		_, err := t.ADB(ctx, serial, "reboot")
		return err
	default:
		return fmt.Errorf("unknown target mode: %s", a.Target)
	}
}

// Helper to normalize CLI values.
func ParseMode(s string) (state.Mode, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "adb":
		return state.ModeADB, nil
	case "fastboot":
		return state.ModeFastboot, nil
	case "recovery":
		return state.ModeRecovery, nil
	case "safemode", "safe", "safe-mode":
		return state.ModeSafeMode, nil
	default:
		return state.ModeUnknown, fmt.Errorf("unknown mode: %q", s)
	}
}
