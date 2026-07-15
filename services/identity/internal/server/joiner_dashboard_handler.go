package server

import (
	"net/http"
)

// OnboardingItem represents a single joiner's onboarding progress.
type OnboardingItem struct {
	ID              string                   `json:"id"`
	Employee        string                   `json:"employee"`
	StartDate       string                   `json:"start_date"`
	StepsCompleted  int                      `json:"steps_completed"`
	TotalSteps      int                      `json:"total_steps"`
	BlockedItems    []string                 `json:"blocked_items"`
	Provisioning    []ProvisioningStatus     `json:"provisioning"`
}

// ProvisioningStatus tracks app provisioning for a joiner.
type ProvisioningStatus struct {
	App    string `json:"app"`
	Status string `json:"status"` // pending, done, failed
}

// JoinerDashboardData is the response for GET /api/v1/identity/joiner-dashboard.
type JoinerDashboardData struct {
	Pending             []OnboardingItem      `json:"pending"`
	CompletionRate      int                   `json:"completion_rate"`
	AvgDaysToComplete   float64               `json:"avg_days_to_complete"`
	UpcomingStarts      []UpcomingStart       `json:"upcoming_starts"`
}

// UpcomingStart lists an employee with an upcoming start date.
type UpcomingStart struct {
	Employee  string `json:"employee"`
	StartDate string `json:"start_date"`
}

// GET /api/v1/identity/joiner-dashboard
// Returns joiner/mover/leaver onboarding dashboard data.
func (h *HTTPHandler) handleJoinerDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	// Query real joiner data from lifecycle table
	data := JoinerDashboardData{
		Pending:        []OnboardingItem{},
		UpcomingStarts: []UpcomingStart{},
	}

	if pool := h.svc.Pool(); pool != nil {
		// Fetch pending onboarding items
		rows, err := pool.Query(ctx, `
			SELECT id, employee_name, start_date, steps_completed, total_steps
			FROM joiner_onboarding
			WHERE status = 'pending'
			ORDER BY start_date ASC
			LIMIT 20
		`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				item := OnboardingItem{
					BlockedItems: []string{},
					Provisioning: []ProvisioningStatus{},
				}
				_ = rows.Scan(&item.ID, &item.Employee, &item.StartDate, &item.StepsCompleted, &item.TotalSteps)
				data.Pending = append(data.Pending, item)
			}
		}

		// Completion rate
		if len(data.Pending) > 0 {
			completed := 0
			totalSteps := 0
			for _, p := range data.Pending {
				totalSteps += p.TotalSteps
				completed += p.StepsCompleted
			}
			if totalSteps > 0 {
				data.CompletionRate = (completed * 100) / totalSteps
			}
		}

		// Average days to complete
		row := pool.QueryRow(ctx, `
			SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - created_at)) / 86400), 0)
			FROM joiner_onboarding WHERE status = 'completed'
		`)
		_ = row.Scan(&data.AvgDaysToComplete)

		// Upcoming starts (next 7 days)
		rows2, err := pool.Query(ctx, `
			SELECT employee_name, start_date FROM joiner_onboarding
			WHERE start_date >= CURRENT_DATE AND start_date <= CURRENT_DATE + INTERVAL '7 days'
			ORDER BY start_date ASC LIMIT 10
		`)
		if err == nil {
			defer rows2.Close()
			for rows2.Next() {
				us := UpcomingStart{}
				_ = rows2.Scan(&us.Employee, &us.StartDate)
				data.UpcomingStarts = append(data.UpcomingStarts, us)
			}
		}
	}

	writeJSON(w, http.StatusOK, data)
}
