package cli

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/checker"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/config"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/diff"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/domain"
	infrahttp "github.com/rubensantoniorosa2704/schemaping-worker/internal/infra/http"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/infra/storage"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/notifier"
)

// Version is set at build time via -ldflags.
var Version = "dev"

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

// Execute is the main entry point for the CLI. It parses os.Args and
// dispatches to the appropriate command handler.
func Execute() {
	args := os.Args[1:]

	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Print(helpText)
		return
	}

	if args[0] == "--version" || args[0] == "version" {
		fmt.Println("schemaping " + Version)
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

	globalNotifiers := buildNotifiers(cfg.Webhooks)
	store := storage.New()
	rawStore := storage.NewRawStore()

	checkers := make(map[string]*checker.Checker, len(cfg.Monitors))
	for _, m := range cfg.Monitors {
		captured := m
		fetcher := infrahttp.New(m)
		checkers[m.Name] = checker.New(m, fetcher, func(e checker.RetryEvent) {
			printMu.Lock()
			defer printMu.Unlock()
			fmt.Fprintf(os.Stderr, "%s[%s]%s retrying (attempt %d/%d, reason: %s, next in %s)\n",
				colorYellow, captured.Name, colorReset,
				e.Attempt, e.MaxAttempts, e.Reason, e.NextIn)
		})
	}

	switch cmd {
	case "check":
		runCheck(cfg, checkers, globalNotifiers, store, rawStore)
	case "run":
		runRun(cfg, checkers, globalNotifiers, store, rawStore)
	case "test-webhooks":
		runTestWebhooks(globalNotifiers)
	}
}

// buildNotifiers constructs a list of Notifier instances from webhook configs.
// Unknown types are logged and skipped.
func buildNotifiers(cfgs []domain.WebhookConfig) []notifier.Notifier {
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

// resolveNotifiers returns the notifiers for a given monitor.
// If the monitor defines its own webhooks list, that overrides the global one.
// An explicit empty list on the monitor silences all notifications.
func resolveNotifiers(m domain.Monitor, global []notifier.Notifier, cfg config.Config) []notifier.Notifier {
	if m.Webhooks == nil {
		return global
	}
	return buildNotifiers(m.Webhooks)
}

// resolveDiffStrategy returns the appropriate DiffStrategy for the monitor's type.
func resolveDiffStrategy(m domain.Monitor) domain.DiffStrategy {
	switch m.Type {
	case domain.MonitorTypeOpenAPI:
		return diff.OpenAPI{}
	default:
		return diff.JSONSchema{}
	}
}

// checkResult holds the outcome of a single monitor check.
type checkResult struct {
	prefix  string
	snap    domain.Snapshot
	prev    domain.Snapshot
	hasPrev bool
	diffs   []domain.DiffResult
}

// executeCheck runs a monitor check, persists the snapshot on success, and returns the result.
func executeCheck(c *checker.Checker, m domain.Monitor, showTimestamp bool, store domain.Store, rawStore domain.RawStore) checkResult {
	prefix := fmt.Sprintf("[%s]", m.Name)
	if showTimestamp {
		prefix = fmt.Sprintf("%s [%s]", time.Now().Format("15:04:05"), m.Name)
	}

	snap := c.Run()

	prev, err := store.Load(m.Name)
	hasPrev := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "%s storage error: %s\n", prefix, err)
	}

	if snap.Error == "" {
		if saveErr := store.Save(snap); saveErr != nil {
			fmt.Fprintf(os.Stderr, "%s save error: %s\n", prefix, saveErr)
		}

		// Persist raw response body when raw mode is enabled for this monitor.
		if m.Raw && len(snap.RawBody) > 0 {
			if rawErr := rawStore.Save(m.Name, snap.CapturedAt, snap.RawBody); rawErr != nil {
				fmt.Fprintf(os.Stderr, "%s raw save error: %s\n", prefix, rawErr)
			}
		}
	}

	return checkResult{
		prefix:  prefix,
		snap:    snap,
		prev:    prev,
		hasPrev: hasPrev,
		diffs:   resolveDiffStrategy(m).Diff(prev, snap),
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
		case domain.ChangeKindAdded:
			fmt.Printf("  %s+ %s added (%s)%s\n", colorGreen, d.Path, d.After, colorReset)
		case domain.ChangeKindRemoved:
			fmt.Printf("  %s- %s removed (%s)%s\n", colorRed, d.Path, d.Before, colorReset)
		case domain.ChangeKindTypeChanged:
			fmt.Printf("  %s~ %s changed: %s -> %s%s\n", colorYellow, d.Path, d.Before, d.After, colorReset)
		case domain.ChangeKindNullabilityChanged:
			fmt.Printf("  %s~ %s nullability changed: %s -> %s%s\n", colorYellow, d.Path, d.Before, d.After, colorReset)
		case domain.ChangeKindStatusChanged:
			fmt.Printf("  %s~ status changed: %s -> %s%s\n", colorYellow, d.Before, d.After, colorReset)
		}
	}
}

// executeAndPrint runs a check, prints the result, and fires webhooks if a change is detected.
func executeAndPrint(c *checker.Checker, m domain.Monitor, showTimestamp bool, notifiers []notifier.Notifier, store domain.Store, rawStore domain.RawStore) {
	r := executeCheck(c, m, showTimestamp, store, rawStore)
	printResult(r)

	if r.hasPrev && len(r.diffs) > 0 {
		notifier.NotifyAll(notifiers, m.Name, r.diffs)
	}
}
