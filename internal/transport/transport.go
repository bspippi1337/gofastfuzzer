package transport

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var ErrNoDevices = errors.New("no devices found")

type Transport interface {
	ListDevices(ctx context.Context) ([]string, error)
	ADB(ctx context.Context, serial string, args ...string) (string, error)
	Fastboot(ctx context.Context, serial string, args ...string) (string, error)
}

type CmdTransport struct {
	ADBPath      string
	FastbootPath string
}

func NewCmdTransport(adbPath, fastbootPath string) *CmdTransport {
	if adbPath == "" {
		adbPath = "adb"
	}
	if fastbootPath == "" {
		fastbootPath = "fastboot"
	}
	return &CmdTransport{ADBPath: adbPath, FastbootPath: fastbootPath}
}

func (t *CmdTransport) ListDevices(ctx context.Context) ([]string, error) {
	// Prefer adb devices first, then fastboot devices.
	serials := map[string]struct{}{}

	if out, err := t.run(ctx, 6*time.Second, t.ADBPath, "devices"); err == nil {
		s := parseDeviceList(out)
		for _, d := range s {
			serials[d] = struct{}{}
		}
	}
	if out, err := t.run(ctx, 6*time.Second, t.FastbootPath, "devices"); err == nil {
		s := parseDeviceList(out)
		for _, d := range s {
			serials[d] = struct{}{}
		}
	}
	if len(serials) == 0 {
		return nil, ErrNoDevices
	}
	res := make([]string, 0, len(serials))
	for s := range serials {
		res = append(res, s)
	}
	sortStrings(res)
	return res, nil
}

func (t *CmdTransport) ADB(ctx context.Context, serial string, args ...string) (string, error) {
	full := []string{}
	if serial != "" {
		full = append(full, "-s", serial)
	}
	full = append(full, args...)
	return t.run(ctx, 20*time.Second, t.ADBPath, full...)
}

func (t *CmdTransport) Fastboot(ctx context.Context, serial string, args ...string) (string, error) {
	full := []string{}
	if serial != "" {
		full = append(full, "-s", serial)
	}
	full = append(full, args...)
	return t.run(ctx, 30*time.Second, t.FastbootPath, full...)
}

func (t *CmdTransport) run(ctx context.Context, timeout time.Duration, bin string, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, bin, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	if cctx.Err() == context.DeadlineExceeded {
		return buf.String(), fmt.Errorf("timeout: %s %s", bin, strings.Join(args, " "))
	}
	if err != nil {
		return buf.String(), fmt.Errorf("%s %s failed: %w\n%s", bin, strings.Join(args, " "), err, buf.String())
	}
	return buf.String(), nil
}

func parseDeviceList(out string) []string {
	var res []string
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "List of devices") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		// For both adb and fastboot the first token is serial.
		serial := fields[0]
		// Skip header-ish lines.
		if strings.EqualFold(serial, "device") {
			continue
		}
		res = append(res, serial)
	}
	return res
}

func sortStrings(s []string) {
	// small local sort to avoid importing sort everywhere
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}
