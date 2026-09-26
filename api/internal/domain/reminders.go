package domain

import (
	"fmt"
	"sort"
	"strings"
)

// Reminder kinds.
const (
	ReminderLeaseEnding   = "lease_ending"
	ReminderDepositReturn = "deposit_return"
	ReminderRentIncrease  = "rent_increase"
	ReminderMoveIn        = "move_in"
	ReminderMaintenance   = "maintenance"
)

// Reminder severities, most pressing first.
const (
	SeverityUrgent  = "urgent"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

var severityRank = map[string]int{SeverityUrgent: 0, SeverityWarning: 1, SeverityInfo: 2}

// Reminder is something the landlord should do soon.
type Reminder struct {
	Kind     string `json:"kind"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	// Date is when it is due or happens.
	Date          Date   `json:"date"`
	LeaseID       string `json:"lease_id,omitempty"`
	MaintenanceID string `json:"maintenance_id,omitempty"`
}

// LeaseFacts is what the reminders need to know about a lease.
type LeaseFacts struct {
	ID           string
	TenantName   string
	Place        string
	Start, End   Date
	TerminatedOn *Date
	Renewed      bool
	DepositCents int64
	DepositHeld  int64
	Terms        RentTerms
}

// MaintenanceFacts is what the reminders need to know about a request.
type MaintenanceFacts struct {
	ID, Title, Priority, Status string
	ReportedOn                  Date
}

// Reminder windows.
const (
	LeaseEndingDays  = 90
	RentIncreaseDays = 30
	MoveInDays       = 14
)

// DefaultDepositReturnDays is how long after move-out a deposit should be
// returned when the landlord has not set their own deadline. Many states
// allow 14 to 45 days.
const DefaultDepositReturnDays = 30

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// article is "An" before a vowel sound and "A" otherwise.
func article(word string) string {
	if word != "" && strings.ContainsRune("aeiou", rune(word[0])) {
		return "An"
	}
	return "A"
}

func inDays(n int) string {
	switch n {
	case 0:
		return "today"
	case 1:
		return "tomorrow"
	}
	return "in " + plural(n, "day", "days")
}

// Reminders works out what needs doing on today, most pressing first.
func Reminders(today Date, leases []LeaseFacts, requests []MaintenanceFacts, depositReturnDays int) []Reminder {
	out := []Reminder{}
	for _, l := range leases {
		end := EffectiveEnd(l.End, l.TerminatedOn)
		status := LeaseStatus(l.Start, l.End, l.TerminatedOn, today)
		who := l.TenantName + " at " + l.Place

		switch status {
		case LeaseActive:
			left := today.DaysUntil(end)
			if !l.Renewed && l.TerminatedOn == nil && left <= LeaseEndingDays {
				sev := SeverityInfo
				switch {
				case left <= 30:
					sev = SeverityUrgent
				case left <= 60:
					sev = SeverityWarning
				}
				out = append(out, Reminder{
					Kind: ReminderLeaseEnding, Severity: sev, Date: end, LeaseID: l.ID,
					Title:  "Lease ends " + inDays(left),
					Detail: who + ". Renew it, or plan the move-out and the next tenant.",
				})
			}
			for _, c := range l.Terms.Changes {
				days := today.DaysUntil(c.EffectiveDate)
				if days > 0 && days <= RentIncreaseDays {
					out = append(out, Reminder{
						Kind: ReminderRentIncrease, Severity: SeverityInfo, Date: c.EffectiveDate, LeaseID: l.ID,
						Title:  "New rent starts " + inDays(days),
						Detail: fmt.Sprintf("%s: rent changes from %s.", who, c.EffectiveDate),
					})
				}
			}
		case LeaseUpcoming:
			days := today.DaysUntil(l.Start)
			if days <= MoveInDays {
				detail := who + "."
				if l.DepositCents > 0 && l.DepositHeld == 0 {
					detail += " The deposit has not been recorded yet."
				}
				out = append(out, Reminder{
					Kind: ReminderMoveIn, Severity: SeverityInfo, Date: l.Start, LeaseID: l.ID,
					Title: "Move-in " + inDays(days), Detail: detail,
				})
			}
		case LeaseEnded, LeaseTerminated:
			if l.Renewed || l.DepositHeld <= 0 {
				continue
			}
			due := end.AddDays(depositReturnDays)
			left := today.DaysUntil(due)
			sev := SeverityWarning
			title := "Return the deposit " + inDays(left)
			if left <= 7 {
				sev = SeverityUrgent
			}
			if left < 0 {
				title = "Deposit return is " + plural(-left, "day", "days") + " late"
			}
			out = append(out, Reminder{
				Kind: ReminderDepositReturn, Severity: sev, Date: due, LeaseID: l.ID, Title: title,
				Detail: fmt.Sprintf("%s moved out on %s. Return the deposit or record what was withheld.", who, end),
			})
		}
	}

	for _, m := range requests {
		if !IsOpenMaintenance(m.Status) {
			continue
		}
		age := m.ReportedOn.DaysUntil(today)
		if (m.Priority == "urgent" && age >= 2) || (m.Priority == "high" && age >= 7) {
			out = append(out, Reminder{
				Kind: ReminderMaintenance, Severity: SeverityWarning, Date: m.ReportedOn, MaintenanceID: m.ID,
				Title:  m.Title,
				Detail: fmt.Sprintf("%s %s request still open after %s.", article(m.Priority), m.Priority, plural(age, "day", "days")),
			})
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if severityRank[out[i].Severity] != severityRank[out[j].Severity] {
			return severityRank[out[i].Severity] < severityRank[out[j].Severity]
		}
		return out[i].Date.Before(out[j].Date)
	})
	return out
}
