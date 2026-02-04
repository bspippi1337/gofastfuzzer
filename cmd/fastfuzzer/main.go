package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bspippi1337/fastfuzzer-ng/internal/actions"
	"github.com/bspippi1337/fastfuzzer-ng/internal/engine"
	"github.com/bspippi1337/fastfuzzer-ng/internal/scorer"
	"github.com/bspippi1337/fastfuzzer-ng/internal/transport"
)

func main() {
	var (
		autoFix  = flag.Bool("auto-fix", false, "auto-diagnose and execute best action")
		setMode  = flag.String("set-mode", "", "set device mode: adb|fastboot|recovery|safemode")
		serial   = flag.String("serial", "", "target a specific device serial")
		adbPath  = flag.String("adb", "", "path to adb binary (default: adb)")
		fbPath   = flag.String("fastboot", "", "path to fastboot binary (default: fastboot)")
		noAuto   = flag.Bool("no-auto", false, "disable automagic scoring (pick first action)")
		verbose  = flag.Bool("v", false, "verbose output")
		timeout  = flag.Duration("timeout", 60*time.Second, "overall command timeout")
		scoreDB  = flag.String("scoredb", "", "path to scoreboard.json (default: ~/.fastfuzzer/scoreboard.json)")
		printTop = flag.Int("top", 0, "print top N learned actions and exit (0 disables)")
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *scoreDB == "" {
		*scoreDB = scorer.DefaultPath()
	}

	t := transport.NewCmdTransport(*adbPath, *fbPath)
	e := engine.New(t)
	e.Auto = !*noAuto
	e.Verbose = *verbose
	_ = e.Scorer.Load(*scoreDB) // best-effort

	if *printTop > 0 {
		for _, row := range e.Scorer.Top(*printTop) {
			fmt.Printf("%s\t%.2f\n", row.Action, row.Score)
		}
		return
	}

	cctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	// If no explicit serial provided, pick the first detected device.
	ser := strings.TrimSpace(*serial)
	if ser == "" {
		devs, err := t.ListDevices(cctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "devices:", err)
			os.Exit(1)
		}
		ser = devs[0]
		if *verbose {
			fmt.Fprintln(os.Stderr, "using device:", ser)
		}
	}

	if strings.TrimSpace(*setMode) != "" {
		mode, err := actions.ParseMode(*setMode)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		s := e.CollectState(cctx, ser)
		a := actions.SetMode{Target: mode}
		if !a.CanApply(s) {
			fmt.Fprintf(os.Stderr, "cannot apply %s in mode %s\n", a.Name(), s.Mode)
			os.Exit(1)
		}
		fmt.Printf("[+] %s (%s -> %s)\n", a.Name(), s.Mode, mode)
		if err := a.Apply(cctx, t, ser); err != nil {
			fmt.Fprintln(os.Stderr, "failed:", err)
			os.Exit(1)
		}
		fmt.Println("[+] done")
		_ = e.Scorer.Save(*scoreDB)
		return
	}

	if *autoFix {
		s := e.CollectState(cctx, ser)
		d, a, err := e.AutoFix(cctx, ser, s)
		fmt.Printf("diagnosis: %s (%v)\n", d.Name, d.Severity)
		fmt.Printf("message: %s\n", d.Message)
		if a == nil {
			fmt.Println("action: (none)")
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			_ = e.Scorer.Save(*scoreDB)
			return
		}
		fmt.Printf("action: %s\n", a.Name())
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed:", err)
			os.Exit(1)
		}
		fmt.Println("[+] done")
		_ = e.Scorer.Save(*scoreDB)
		return
	}

	flag.Usage()
	os.Exit(2)
}
