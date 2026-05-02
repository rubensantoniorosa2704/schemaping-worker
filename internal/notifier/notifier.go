// Package notifier defines the Notifier interface and the dispatcher that
// fans out schema-change alerts to all configured webhook targets.
package notifier

import (
	"fmt"
	"os"

	"github.com/rubensantoniorosa2704/schemaping-worker/pkg/types"
)

// Notifier sends a schema-change alert for a given monitor.
type Notifier interface {
	Notify(monitorName string, diffs []types.DiffResult) error
}

// Tester is implemented by notifiers that support sending a test message
// to verify the webhook is reachable and correctly configured.
type Tester interface {
	Test() error
}

// ExpandEnv replaces ${VAR} or $VAR occurrences in s with the corresponding
// environment variable values, identical to os.ExpandEnv.
func ExpandEnv(s string) string {
	return os.ExpandEnv(s)
}

// NotifyAll dispatches the alert to every notifier, printing errors to stderr
// without stopping the remaining ones.
func NotifyAll(notifiers []Notifier, monitorName string, diffs []types.DiffResult) {
	for _, n := range notifiers {
		if err := n.Notify(monitorName, diffs); err != nil {
			fmt.Fprintf(os.Stderr, "[notifier] %s: %s\n", monitorName, err)
		}
	}
}
