package main

import "sync"

// Both native entrypoints must wake the socket hook after HarmonyOS protects
// the descriptor. Updating the legacy map alone leaves the hook waiting.
func acknowledgeSocketProtection(id int64, protected, waiters *sync.Map) {
	protected.Store(id, struct{}{})
	if value, ok := waiters.Load(id); ok {
		if ch, ok := value.(chan struct{}); ok {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}
}
