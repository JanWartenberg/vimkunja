package tui

import "time"

// DefaultLeaderTimeout is kept with TUI configuration so a future user-facing
// config file can expose it without changing the leader state machine.
const DefaultLeaderTimeout = 500 * time.Millisecond
