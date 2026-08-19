package cli

import (
	"fmt"
	"os"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/notifier"
)

// runTestWebhooks sends a test notification to every configured notifier and
// reports success or failure for each one. Exits with code 1 if any fails.
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
