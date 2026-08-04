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
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/notifier"
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
  run            Load config and run checks continuously on each monitor's interval
  check          Run a single check for all monitors and exit
  test-webhooks  Send a test notification to all configured webhooks and exit

FLAGS:
  --config <path>       Path to config file (default: ./config.yaml)
  --interval <duration> Override interval for all monitors (e.g. 30s, 2m)
  --help                Show this help message
  --version             Show version

EXAMPLES:
  schemaping run --config ./examples/config.yaml
  schemaping run --config ./examples/config.yaml --interval 30s
  schemaping check --config ./examples/config.yaml
  schemaping test-webhooks --config ./examples/config.yaml

CONFIG FORMAT (YAML):
  monitors:
    - name: payments-api
      url: https://api.example.com/v1/payments
      method: GET
      interval: 5m
      timeout: 10s
      expected_status: 200
      headers:
        Authorization: Bearer ${API_TOKEN}

  webhooks:
    - type: discord
      url: ${DISCORD_WEBHOOK_URL}
    - type: telegram
      url: https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage
      chat_id: ${TELEGRAM_CHAT_ID}

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
	if cmd != "run" && cmd != "check" && cmd != "test-webhooks" {
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

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	if intervalOverride > 0 {
		for i := range cfg.Monitors {
			cfg.Monitors[i].Interval = intervalOverride
		}
	}

	notifiers := buildNotifiers(cfg.Webhooks)

	checkers := make(map[string]*checker.Checker, len(cfg.Monitors))
	for _, m := range cfg.Monitors {
		captured := m // capture for closure
		checkers[m.Name] = checker.New(m, func(e checker.RetryEvent) {
			printMu.Lock()
			defer printMu.Unlock()
			fmt.Fprintf(os.Stderr, "%s[%s]%s retrying (attempt %d/%d, reason: %s, next in %s)\n",
				colorYellow, captured.Name, colorReset,
				e.Attempt, e.MaxAttempts, e.Reason, e.NextIn)
		})
	}

	switch cmd {
	case "check":
		for _, m := range cfg.Monitors {
			executeAndPrint(checkers[m.Name], m, false, resolveNotifiers(m, notifiers, cfg))
		}
	case "run":
		fmt.Printf("%sSchemaPing v%s%s starting — %d monitor(s) loaded\n\n", colorBold, version, colorReset, len(cfg.Monitors))
		scheduler.Run(cfg.Monitors, func(m types.Monitor) {
			executeAndPrint(checkers[m.Name], m, true, resolveNotifiers(m, notifiers, cfg))
		})
		fmt.Println("\nSchemaPing stopped.")
	case "test-webhooks":
		runTestWebhooks(notifiers)
	}
}

// runTestWebhooks sends a test notification to every global notifier and
// reports success or failure for each one.
func runTestWebhooks(notifiers []notifier.Notifier) {
	if len(notifiers) == 0 {
		fmt.Println("No webhooks configured.")
		return
	}
	fmt.Printf("Testing %d webhook(s)...\n", len(notifiers))
	ok := true
	for i, n := range notifiers {
		t, isTester := n.(notifier.Tester)
		if !isTester {
			fmt.Printf("  [%d] skipped (does not support test)\n", i+1)
			continue
		}
		if err := t.Test(); err != nil {
			fmt.Printf("  [%d] %sFAIL%s — %s\n", i+1, colorRed, colorReset, err)
			ok = false
		} else {
			fmt.Printf("  [%d] %sOK%s\n", i+1, colorGreen, colorReset)
		}
	}
	if !ok {
		os.Exit(1)
	}
}

// buildNotifiers constructs a list of Notifier instances from webhook configs.
// Unknown types are logged and skipped.
func buildNotifiers(cfgs []types.WebhookConfig) []notifier.Notifier {
	var result []notifier.Notifier
	for _, wh := range cfgs {
		switch wh.Type {
		case "discord":
			result = append(result, notifier.NewDiscord(wh.URL))
		case "telegram":
			result = append(result, notifier.NewTelegram(wh.URL, wh.ChatID))
		default:
			fmt.Fprintf(os.Stderr, "[notifier] unknown webhook type %q — skipping\n", wh.Type)
		}
	}
	return result
}

// resolveNotifiers returns the notifiers to use for a given monitor.
// If the monitor defines its own webhooks, those are used (override).
// Otherwise the global notifiers are used.
// An explicit empty webhooks list on the monitor silences all notifications.
func resolveNotifiers(m types.Monitor, global []notifier.Notifier, cfg config.Config) []notifier.Notifier {
	if m.Webhooks == nil {
		return global
	}
	return buildNotifiers(m.Webhooks)
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

func executeAndPrint(c *checker.Checker, m types.Monitor, showTimestamp bool, notifiers []notifier.Notifier) {
	r := runCheck(c, m, showTimestamp)
	printResult(r)

	// Fire webhook notifications only when a real change is detected
	// (i.e., there was a previous snapshot to compare against).
	if r.hasPrev && len(r.diffs) > 0 {
		notifier.NotifyAll(notifiers, m.Name, r.diffs)
	}
}
