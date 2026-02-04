# fastfuzzer-ng (Go rewrite)

A scratch rewrite of **fastfuzzer** in Go: a fast, rule-based Android transport diagnostician + auto-repair helper for **adb**, **fastboot**, and **recovery**.

This version keeps the spirit of your original tool (diagnose → pick plan → execute), but is:
- **faster** (compiled, minimal overhead)
- **more modular** (transport/rules/actions separated)
- **automagic** (learns which actions work over time via a simple scoreboard)

> Safety note: this rewrite intentionally avoids destructive actions (wipes/flashes) by default.

## Install / build

```bash
go build -o fastfuzzer ./cmd/fastfuzzer
```

You need `adb` and `fastboot` available in `PATH`, or pass paths with `-adb` and `-fastboot`.

## Usage

### Auto-fix (diagnose + execute best action)
```bash
./fastfuzzer --auto-fix
```

### Force a mode change
```bash
./fastfuzzer --set-mode fastboot
./fastfuzzer --set-mode recovery
./fastfuzzer --set-mode safemode
```

### Target a specific device
```bash
./fastfuzzer --serial <SERIAL> --auto-fix
```

### Disable automagic scoring
```bash
./fastfuzzer --auto-fix --no-auto
```

### Show top learned actions
```bash
./fastfuzzer --top 10
```

## Automagic scoring

The tool maintains a tiny scoreboard (JSON) of action success/failure and uses it to bias future choices.
Default path:
- `~/.fastfuzzer/scoreboard.json`

Override with:
```bash
./fastfuzzer --scoredb /path/to/scoreboard.json --auto-fix
```

## Architecture

- `internal/transport`: command transport for `adb`/`fastboot`
- `internal/state`: immutable device state snapshot
- `internal/rules`: pure diagnosis rules (no side effects)
- `internal/actions`: idempotent-ish actions (mode switches)
- `internal/scorer`: automagic scoreboard
- `internal/engine`: orchestration (collect → diagnose → choose → apply)

## Next extensions

If you want parity with a larger original repo, the natural next steps are:
- richer state collection (build fingerprint, slot, boot reason)
- recovery scripting (adb sideload hooks)
- raw USB transports (libusb)
- guarded destructive actions (behind an explicit `--allow-destructive`)

