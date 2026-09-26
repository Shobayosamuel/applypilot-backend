package job

import (
	"encoding/json"
	"time"
)

type Resource struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Company      string          `json:"company"`
	Location     string          `json:"location"`
	URL          string          `json:"url"`
	Description  string          `json:"description"`
	Salary       string          `json:"salary"`
	Source       string          `json:"source"`
	Status       string          `json:"status"`
	Notes        string          `json:"notes"`
	MatchScore   *int            `json:"match_score"`
	MatchDetails json.RawMessage `json:"match_details"`
	AppliedAt    *time.Time      `json:"applied_at"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func NewResource(j *Job) Resource {
	return Resource{
		ID: j.ID, Title: j.Title, Company: j.Company, Location: j.Location, URL: j.URL, Description: j.Description,
		Salary: j.Salary, Source: j.Source, Status: j.Status, Notes: j.Notes, MatchScore: j.MatchScore,
		MatchDetails: j.MatchDetails, AppliedAt: j.AppliedAt, CreatedAt: j.CreatedAt, UpdatedAt: j.UpdatedAt,
	}
}

func NewResources(list []Job) []Resource {
	out := make([]Resource, 0, len(list))
	for i := range list {
		out = append(out, NewResource(&list[i]))
	}
	return out
}

type StatsResource struct {
	ByStatus     map[string]int `json:"by_status"`
	Total        int            `json:"total"`
	AppliedWeek  int            `json:"applied_this_week"`
	AppliedTotal int            `json:"applied_total"`
	AICallsToday int            `json:"ai_calls_today"`
}
