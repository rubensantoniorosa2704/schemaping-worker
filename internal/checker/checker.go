package checker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/httpclient"
	"github.com/rubensantoniorosa2704/schemaping-worker/pkg/types"
)

// Run executes a check for the given monitor and returns a Snapshot.
// Failures are captured in Snapshot.Error; this function never returns an error.
func Run(m types.Monitor) types.Snapshot {
	snap := types.Snapshot{
		MonitorName: m.Name,
		CapturedAt:  time.Now().UTC(),
	}

	statusCode, body, err := httpclient.Do(m)
	if err != nil {
		snap.Error = err.Error()
		return snap
	}

	snap.StatusCode = statusCode

	if statusCode != m.ExpectedStatus {
		snap.Error = fmt.Sprintf("unexpected status: got %d, want %d", statusCode, m.ExpectedStatus)
	}

	if len(body) > 0 {
		if err := json.Unmarshal(body, &snap.Body); err != nil && snap.Error == "" {
			snap.Error = fmt.Sprintf("parse body: %s", err.Error())
		}
	}

	return snap
}
