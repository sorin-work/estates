package domain

import "testing"

func TestMaintenanceTransitions(t *testing.T) {
	allowed := [][2]string{
		{MaintenanceOpen, MaintenanceInProgress},
		{MaintenanceOpen, MaintenanceCompleted},
		{MaintenanceInProgress, MaintenanceOnHold},
		{MaintenanceOnHold, MaintenanceInProgress},
		{MaintenanceCompleted, MaintenanceOpen},
		{MaintenanceCancelled, MaintenanceOpen},
	}
	for _, tr := range allowed {
		if !CanTransitionMaintenance(tr[0], tr[1]) {
			t.Errorf("%s -> %s should be allowed", tr[0], tr[1])
		}
	}

	denied := [][2]string{
		{MaintenanceCompleted, MaintenanceInProgress},
		{MaintenanceCompleted, MaintenanceCancelled},
		{MaintenanceCancelled, MaintenanceCompleted},
		{MaintenanceOnHold, MaintenanceCompleted},
		{MaintenanceOpen, "bogus"},
		{"bogus", MaintenanceOpen},
	}
	for _, tr := range denied {
		if CanTransitionMaintenance(tr[0], tr[1]) {
			t.Errorf("%s -> %s should not be allowed", tr[0], tr[1])
		}
	}
}

func TestEveryStatusHasAWayOut(t *testing.T) {
	for _, s := range MaintenanceStatuses {
		if len(NextMaintenanceStatuses(s)) == 0 {
			t.Errorf("%s is a dead end", s)
		}
	}
}

func TestIsOpenMaintenance(t *testing.T) {
	for s, want := range map[string]bool{
		MaintenanceOpen: true, MaintenanceInProgress: true, MaintenanceOnHold: true,
		MaintenanceCompleted: false, MaintenanceCancelled: false,
	} {
		if IsOpenMaintenance(s) != want {
			t.Errorf("%s: want %v", s, want)
		}
	}
}
