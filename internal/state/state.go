package state

import "time"

type Mode string

const (
	ModeUnknown  Mode = "unknown"
	ModeADB      Mode = "adb"
	ModeFastboot Mode = "fastboot"
	ModeRecovery Mode = "recovery"
	ModeSafeMode Mode = "safemode"
)

type DeviceState struct {
	Serial    string
	Mode      Mode
	Battery   int
	Props     map[string]string
	Errors    []string
	Timestamp time.Time
}
