package domain

// Maintenance request statuses.
const (
	MaintenanceOpen       = "open"
	MaintenanceInProgress = "in_progress"
	MaintenanceOnHold     = "on_hold"
	MaintenanceCompleted  = "completed"
	MaintenanceCancelled  = "cancelled"
)

var MaintenanceStatuses = []string{
	MaintenanceOpen, MaintenanceInProgress, MaintenanceOnHold, MaintenanceCompleted, MaintenanceCancelled,
}

// Maintenance priorities, lowest first.
var MaintenancePriorities = []string{"low", "normal", "high", "urgent"}

var maintenanceTransitions = map[string][]string{
	MaintenanceOpen:       {MaintenanceInProgress, MaintenanceOnHold, MaintenanceCompleted, MaintenanceCancelled},
	MaintenanceInProgress: {MaintenanceOpen, MaintenanceOnHold, MaintenanceCompleted, MaintenanceCancelled},
	MaintenanceOnHold:     {MaintenanceOpen, MaintenanceInProgress, MaintenanceCancelled},
	MaintenanceCompleted:  {MaintenanceOpen},
	MaintenanceCancelled:  {MaintenanceOpen},
}

// NextMaintenanceStatuses lists the statuses a request can move to.
func NextMaintenanceStatuses(from string) []string {
	return maintenanceTransitions[from]
}

// CanTransitionMaintenance reports whether a request can move from one status
// to another. Closed requests (completed or cancelled) can only be reopened.
func CanTransitionMaintenance(from, to string) bool {
	for _, s := range maintenanceTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// IsOpenMaintenance reports whether a status still needs attention.
func IsOpenMaintenance(status string) bool {
	return status == MaintenanceOpen || status == MaintenanceInProgress || status == MaintenanceOnHold
}
