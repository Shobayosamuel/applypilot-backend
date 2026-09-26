package ai

import (
	"encoding/json"
	"time"
)

type ParsedProfileResource struct {
	Profile map[string]any `json:"profile"`
}

type MatchResource struct {
	Score         int      `json:"score"`
	Verdict       string   `json:"verdict"`
	Summary       string   `json:"summary"`
	MatchedSkills []string `json:"matched_skills"`
	MissingSkills []string `json:"missing_skills"`
	Keywords      []string `json:"keywords"`
	Suggestions   []string `json:"suggestions"`
}

type CoverLetterResource struct {
	CoverLetter string `json:"cover_letter"`
}

type TailoredExperience struct {
	Company string   `json:"company"`
	Title   string   `json:"title"`
	Bullets []string `json:"bullets"`
}

type TailoredResumeResource struct {
	Headline      string               `json:"headline"`
	Summary       string               `json:"summary"`
	Skills        []string             `json:"skills"`
	Experience    []TailoredExperience `json:"experience"`
	KeywordsAdded []string             `json:"keywords_added"`
	Notes         string               `json:"notes"`
}

type AnswerResource struct {
	Answer string `json:"answer"`
}

type FieldValue struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type MapFieldsResource struct {
	Values []FieldValue `json:"values"`
}

type ExtractedJobResource struct {
	Title       string `json:"title"`
	Company     string `json:"company"`
	Location    string `json:"location"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

type UsageResource struct {
	CallsToday   int `json:"calls_today"`
	DailyLimit   int `json:"daily_limit"`
	InputTokens  int `json:"input_tokens_30d"`
	OutputTokens int `json:"output_tokens_30d"`
	CacheRead    int `json:"cache_read_tokens_30d"`
	Calls30d     int `json:"calls_30d"`
}

type GenerationResource struct {
	ID           string    `json:"id"`
	JobID        *string   `json:"job_id"`
	Kind         string    `json:"kind"`
	InputSummary string    `json:"input_summary"`
	Output       string    `json:"output"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	CacheRead    int       `json:"cache_read_tokens"`
	CreatedAt    time.Time `json:"created_at"`
}

func NewGenerationResources(list []Generation) []GenerationResource {
	out := make([]GenerationResource, 0, len(list))
	for _, g := range list {
		out = append(out, GenerationResource{
			ID: g.ID, JobID: g.JobID, Kind: g.Kind, InputSummary: g.InputSummary, Output: g.Output, Model: g.Model,
			InputTokens: g.InputTokens, OutputTokens: g.OutputTokens, CacheRead: g.CacheRead, CreatedAt: g.CreatedAt,
		})
	}
	return out
}

// outputString renders a result for the generation log.
func outputString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}
