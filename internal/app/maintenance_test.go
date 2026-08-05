package app

import "testing"

func TestMaintenanceState(t *testing.T) {
	t.Parallel()

	state := &MaintenanceState{}
	if state.Active() {
		t.Fatal("new maintenance state is active")
	}
	state.Enter()
	if !state.Active() {
		t.Fatal("maintenance state did not activate")
	}
	state.Exit()
	if state.Active() {
		t.Fatal("maintenance state did not clear")
	}
}
