package cli

import (
	"fmt"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/checker"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/config"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/domain"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/notifier"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/scheduler"
)

// runRun starts SchemaPing in continuous mode, scheduling each monitor on its own interval.
func runRun(cfg config.Config, checkers map[string]*checker.Checker, globalNotifiers []notifier.Notifier, stores *storeSet) {
	fmt.Printf("%sSchemaPing v%s%s starting — %d monitor(s) loaded\n\n", colorBold, Version, colorReset, len(cfg.Monitors))
	scheduler.Run(cfg.Monitors, func(m domain.Monitor) {
		executeAndPrint(checkers[m.Name], m, true, resolveNotifiers(m, globalNotifiers, cfg), stores)
	})
	fmt.Println("\nSchemaPing stopped.")
}
