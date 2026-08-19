package domain

// Fetcher is the port for retrieving a snapshot from an external source.
// Implementations live in internal/infra and are injected into the checker.
type Fetcher interface {
	// Fetch executes a request for the given monitor and returns a Snapshot.
	// Transport and protocol errors are captured inside Snapshot.Error;
	// this method never returns a non-nil error.
	Fetch(m Monitor) Snapshot
}

// Store is the port for persisting and retrieving snapshots.
// Implementations live in internal/infra and are injected into the checker.
type Store interface {
	// Save persists a snapshot, overwriting any previous one for the same monitor.
	Save(snap Snapshot) error
	// Load retrieves the latest snapshot for the given monitor name.
	// Returns an error wrapping os.ErrNotExist if no snapshot has been saved yet.
	Load(monitorName string) (Snapshot, error)
}
