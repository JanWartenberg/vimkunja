// Package errors keeps an in-memory record of errors encountered during the
// current Vimkunja session.
package errors

import "sync"

var sessionLog struct {
	sync.RWMutex
	entries []string
}

// Log records msg for later display in the error pane.
func Log(msg string) {
	if msg == "" {
		return
	}

	sessionLog.Lock()
	defer sessionLog.Unlock()
	sessionLog.entries = append(sessionLog.entries, msg)
}

// GetAll returns a copy of all errors recorded during this session.
func GetAll() []string {
	sessionLog.RLock()
	defer sessionLog.RUnlock()

	return append([]string(nil), sessionLog.entries...)
}
