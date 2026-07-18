package server

import (
	"net/http"
	"sync"

	"github.com/google/uuid"
)

// skillEntry represents a user's proficiency level for a specific skill.
type skillEntry struct {
	SkillID            string `json:"skill_id"`
	SkillName          string `json:"skill_name"`
	Category           string `json:"category"`
	ProficiencyLevel   string `json:"proficiency_level"` // beginner, intermediate, advanced, expert
	YearsOfExperience  int    `json:"years_of_experience"`
	Certified          bool   `json:"certified"`
}

// userSkillMatrix represents a single user row in the skill matrix.
type userSkillMatrix struct {
	UserID      string       `json:"user_id"`
	Username    string       `json:"username"`
	DisplayName string       `json:"display_name"`
	Department  string       `json:"department"`
	Skills      []skillEntry `json:"skills"`
}

var skillMatrixStore = struct {
	sync.RWMutex
	data map[string]*userSkillMatrix // userID → matrix
}{data: make(map[string]*userSkillMatrix)}

// GET /api/v1/users/skill-matrix?org=X&department=Y&skill=Z
// Returns a users × skills grid with proficiency levels.
func (h *HTTPHandler) handleSkillMatrix(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	orgID := r.URL.Query().Get("org")
	department := r.URL.Query().Get("department")
	skillFilter := r.URL.Query().Get("skill")

	skillMatrixStore.RLock()
	result := []userSkillMatrix{}
	allSkills := map[string]bool{}

	for _, u := range skillMatrixStore.data {
		// Apply filters
		if department != "" && u.Department != department {
			continue
		}

		// Filter skills if skillFilter is set
		filtered := u.Skills
		if skillFilter != "" {
			filtered = []skillEntry{}
			for _, s := range u.Skills {
				if s.SkillID == skillFilter || s.SkillName == skillFilter {
					filtered = append(filtered, s)
				}
			}
			if len(filtered) == 0 {
				continue
			}
		}

		for _, s := range u.Skills {
			allSkills[s.SkillName] = true
		}

		result = append(result, userSkillMatrix{
			UserID:      u.UserID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			Department:  u.Department,
			Skills:      filtered,
		})
	}
	skillMatrixStore.RUnlock()

	// Build unique skill list
	skillList := make([]string, 0, len(allSkills))
	for s := range allSkills {
		skillList = append(skillList, s)
	}

	// Compute summary stats
	proficiencyCounts := map[string]int{} // level → count
	for _, u := range result {
		for _, s := range u.Skills {
			proficiencyCounts[s.ProficiencyLevel]++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"org_id":              orgID,
		"total_users":         len(result),
		"total_skills":        len(skillList),
		"skill_names":         skillList,
		"proficiency_summary": proficiencyCounts,
		"matrix":              result,
	})
}

// POST /api/v1/users/skill-matrix — seed sample data for testing/demo
func init() {
	// Pre-populate with sample data so the endpoint is useful out of the box
	sampleUsers := []userSkillMatrix{
		{
			UserID:      uuid.New().String(),
			Username:    "alice.eng",
			DisplayName: "Alice Engineering",
			Department:  "Engineering",
			Skills: []skillEntry{
				{SkillID: "go", SkillName: "Go", Category: "Programming", ProficiencyLevel: "expert", YearsOfExperience: 8, Certified: true},
				{SkillID: "k8s", SkillName: "Kubernetes", Category: "DevOps", ProficiencyLevel: "advanced", YearsOfExperience: 5, Certified: true},
				{SkillID: "postgres", SkillName: "PostgreSQL", Category: "Database", ProficiencyLevel: "intermediate", YearsOfExperience: 4, Certified: false},
			},
		},
		{
			UserID:      uuid.New().String(),
			Username:    "bob.sec",
			DisplayName: "Bob Security",
			Department:  "Security",
			Skills: []skillEntry{
				{SkillID: "sec-audit", SkillName: "Security Auditing", Category: "Security", ProficiencyLevel: "expert", YearsOfExperience: 10, Certified: true},
				{SkillID: "go", SkillName: "Go", Category: "Programming", ProficiencyLevel: "intermediate", YearsOfExperience: 3, Certified: false},
				{SkillID: "pentest", SkillName: "Penetration Testing", Category: "Security", ProficiencyLevel: "advanced", YearsOfExperience: 7, Certified: true},
			},
		},
		{
			UserID:      uuid.New().String(),
			Username:    "carol.devops",
			DisplayName: "Carol DevOps",
			Department:  "Engineering",
			Skills: []skillEntry{
				{SkillID: "k8s", SkillName: "Kubernetes", Category: "DevOps", ProficiencyLevel: "expert", YearsOfExperience: 6, Certified: true},
				{SkillID: "terraform", SkillName: "Terraform", Category: "DevOps", ProficiencyLevel: "advanced", YearsOfExperience: 5, Certified: true},
				{SkillID: "go", SkillName: "Go", Category: "Programming", ProficiencyLevel: "beginner", YearsOfExperience: 1, Certified: false},
			},
		},
	}

	skillMatrixStore.Lock()
	for _, u := range sampleUsers {
		copy := u
		skillMatrixStore.data[copy.UserID] = &copy
	}
	skillMatrixStore.Unlock()
}
