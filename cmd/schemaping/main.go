package main

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/checker"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/config"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/diff"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/scheduler"
	filestore "github.com/rubensantoniorosa2704/schemaping-worker/internal/storage/file"
	"github.com/rubensantoniorosa2704/schemaping-worker/pkg/types"
)

var version = "dev"

var printMu sync.Mutex

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

const helpText = `SchemaPing — API schema drift monitor

USAGE:
  schemaping <command> [flags]

COMMANDS:
  run     Load config and run checks continuously on each monitor's interval
  check   Run a single check for all monitors and exit

FLAGS:
  --config <path>       Path to config file (default: ./config.yaml)
  --interval <duration> Override interval for all monitors (e.g. 30s, 2m)
  --help                Show this help message
  --version             Show version

EXAMPLES:
  schemaping run --config ./examples/config.yaml
  schemaping run --config ./examples/config.yaml --interval 30s
  schemaping check --config ./examples/config.yaml

CONFIG FORMAT (YAML):
  monitors:
    - name: payments-api
      url: https://api.example.com/v1/payments
      method: GET
      interval: 5m
      timeout: 10s
      expected_status: 200
      headers:
        Authorization: Bearer YOUR_TOKEN

SOURCE:
  https://github.com/rubensantoniorosa2704/schemaping-worker
`

func main() {
	args := os.Args[1:]

	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Print(helpText)
		return
	}

	if args[0] == "--version" || args[0] == "version" {
		fmt.Println("schemaping " + version)
		return
	}

	cmd := args[0]
	if cmd != "run" && cmd != "check" {
		fmt.Fprintf(os.Stderr, "unknown command: %q\nRun 'schemaping --help' for usage.\n", cmd)
		os.Exit(1)
	}

	configPath := "./config.yaml"
	var intervalOverride time.Duration

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--config":
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			}
		case "--interval":
			if i+1 < len(args) {
				d, err := time.ParseDuration(args[i+1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "invalid --interval value: %s\n", args[i+1])
					os.Exit(1)
				}
				intervalOverride = d
				i++
			}
		}
	}

	monitors, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	if intervalOverride > 0 {
		for i := range monitors {
			monitors[i].Interval = intervalOverride
		}
	}

	checkers := make(map[string]*checker.Checker, len(monitors))
	for _, m := range monitors {
		checkers[m.Name] = checker.New(m)
	}

	switch cmd {
	case "check":
		for _, m := range monitors {
			executeAndPrint(checkers[m.Name], m, false)
		}
	case "run":
		fmt.Printf("%sSchemaPing v%s%s starting — %d monitor(s) loaded\n\n", colorBold, version, colorReset, len(monitors))
		scheduler.Run(monitors, func(m types.Monitor) { executeAndPrint(checkers[m.Name], m, true) })
		fmt.Println("\nSchemaPing stopped.")
	}
}

// checkResult holds the outcome of a single monitor check.
type checkResult struct {
	prefix  string
	snap    types.Snapshot
	prev    types.Snapshot
	hasPrev bool
	diffs   []types.DiffResult
}

// runCheck executes a monitor check, persists the snapshot (only on success), and returns the result.
func runCheck(c *checker.Checker, m types.Monitor, showTimestamp bool) checkResult {
	prefix := fmt.Sprintf("[%s]", m.Name)
	if showTimestamp {
		prefix = fmt.Sprintf("%s [%s]", time.Now().Format("15:04:05"), m.Name)
	}

	snap := c.Run()

	prev, err := filestore.Load(m.Name)
	hasPrev := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "%s storage error: %s\n", prefix, err)
	}

	if snap.Error == "" {
		if saveErr := filestore.Save(snap); saveErr != nil {
			fmt.Fprintf(os.Stderr, "%s save error: %s\n", prefix, saveErr)
		}
	}

	return checkResult{
		prefix:  prefix,
		snap:    snap,
		prev:    prev,
		hasPrev: hasPrev,
		diffs:   diff.Compare(prev, snap),
	}
}

// printResult writes the check outcome to stdout.
func printResult(r checkResult) {
	printMu.Lock()
	defer printMu.Unlock()

	if r.snap.Error != "" {
		fmt.Printf("%s%s error:%s %s\n", colorRed, r.prefix, colorReset, r.snap.Error)
		return
	}

	if !r.hasPrev {
		fmt.Printf("%s%s%s first snapshot captured (status %d)\n", colorCyan, r.prefix, colorReset, r.snap.StatusCode)
		return
	}

	if len(r.diffs) == 0 {
		fmt.Printf("%s%s%s no changes detected\n", colorGreen, r.prefix, colorReset)
		return
	}

	fmt.Printf("%s%s%s change detected\n", colorYellow+colorBold, r.prefix, colorReset)
	for _, d := range r.diffs {
		switch d.Kind {
		case types.ChangeKindAdded:
			fmt.Printf("  %s+ %s added (%s)%s\n", colorGreen, d.Path, d.After, colorReset)
		case types.ChangeKindRemoved:
			fmt.Printf("  %s- %s removed (%s)%s\n", colorRed, d.Path, d.Before, colorReset)
		case types.ChangeKindTypeChanged:
			fmt.Printf("  %s~ %s changed: %s -> %s%s\n", colorYellow, d.Path, d.Before, d.After, colorReset)
		case types.ChangeKindNullabilityChanged:
			fmt.Printf("  %s~ %s nullability changed: %s -> %s%s\n", colorYellow, d.Path, d.Before, d.After, colorReset)
		case types.ChangeKindStatusChanged:
			fmt.Printf("  %s~ status changed: %s -> %s%s\n", colorYellow, d.Before, d.After, colorReset)
		}
	}
}

func executeAndPrint(c *checker.Checker, m types.Monitor, showTimestamp bool) {
	printResult(runCheck(c, m, showTimestamp))
}
