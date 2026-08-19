package cli

import (
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/checker"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/config"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/domain"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/notifier"
)

// runCheck executes a single check for every monitor and exits.
func runCheck(cfg config.Config, checkers map[string]*checker.Checker, globalNotifiers []notifier.Notifier, store domain.Store, rawStore domain.RawStore) {
	for _, m := range cfg.Monitors {
		executeAndPrint(checkers[m.Name], m, false, resolveNotifiers(m, globalNotifiers, cfg), store, rawStore)
	}
}
