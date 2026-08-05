package app

import "sync/atomic"

// MaintenanceState is shared by the HTTP surface and future recovery work.
// Enter and Exit are safe to call from different goroutines.
type MaintenanceState struct {
	active atomic.Bool
}

func (state *MaintenanceState) Enter() {
	state.active.Store(true)
}

func (state *MaintenanceState) Exit() {
	state.active.Store(false)
}

func (state *MaintenanceState) Active() bool {
	return state.active.Load()
}
