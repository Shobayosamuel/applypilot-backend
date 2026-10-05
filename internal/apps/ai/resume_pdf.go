package ai

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/apperr"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/pdfgen"
)

// profileDoc is the subset of the stored profile JSON (see prompts.go's
// toolSaveProfile schema) needed to render a resume PDF.
type profileDoc struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Location  struct {
		City    string `json:"city"`
		State   string `json:"state"`
		Country string `json:"country"`
	} `json:"location"`
	Links struct {
		LinkedIn  string `json:"linkedin"`
		GitHub    string `json:"github"`
		Portfolio string `json:"portfolio"`
		Website   string `json:"website"`
	} `json:"links"`
	WorkExperience []profileWorkExperience `json:"work_experience"`
	Education      []profileEducation      `json:"education"`
	Languages      []string                `json:"languages"`
	Certifications []string                `json:"certifications"`
}

type profileWorkExperience struct {
	Company   string `json:"company"`
	Title     string `json:"title"`
	Location  string `json:"location"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Current   bool   `json:"current"`
}

type profileEducation struct {
	School       string `json:"school"`
	Degree       string `json:"degree"`
	FieldOfStudy string `json:"field_of_study"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
}

// ResumePDF renders a previously generated tailor_resume result as a PDF,
// filling in contact details, dates, and education from the current profile
// (the tailored content itself only covers headline, summary, and bullets).
func (s *Service) ResumePDF(ctx context.Context, userID, generationID string) ([]byte, string, error) {
	g, err := s.repo.Get(ctx, userID, generationID)
	if err != nil {
		return nil, "", apperr.NotFound("generation not found")
	}
	if g.Kind != "tailor_resume" {
		return nil, "", apperr.Invalid("this generation is not a tailored resume")
	}
	var tailored TailoredResumeResource
	if err := json.Unmarshal([]byte(g.Output), &tailored); err != nil {
		return nil, "", apperr.Upstream("could not read the saved tailored resume", err)
	}

	raw, err := s.profile.Raw(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	var prof profileDoc
	if err := json.Unmarshal(raw, &prof); err != nil {
		return nil, "", apperr.Upstream("could not read the candidate profile", err)
	}

	name := strings.TrimSpace(prof.FirstName + " " + prof.LastName)
	pdf, err := pdfgen.Render(pdfgen.Resume{
		Name:           name,
		Headline:       tailored.Headline,
		Email:          prof.Email,
		Phone:          prof.Phone,
		Location:       joinNonEmpty(", ", prof.Location.City, prof.Location.State, prof.Location.Country),
		Links:          nonEmpty(prof.Links.LinkedIn, prof.Links.GitHub, prof.Links.Portfolio, prof.Links.Website),
		Summary:        tailored.Summary,
		Skills:         tailored.Skills,
		Experience:     mergeExperience(prof.WorkExperience, tailored.Experience),
		Education:      renderEducation(prof.Education),
		Certifications: prof.Certifications,
		Languages:      prof.Languages,
	})
	if err != nil {
		return nil, "", err
	}
	filename := strings.TrimSpace(name+" Resume") + ".pdf"
	if name == "" {
		filename = "Resume.pdf"
	}
	return pdf, filename, nil
}

// mergeExperience fills in location and dates for each tailored role by
// matching it to the profile's original entry with the same company.
func mergeExperience(original []profileWorkExperience, tailored []TailoredExperience) []pdfgen.Experience {
	byCompany := make(map[string]int, len(original))
	for i, o := range original {
		byCompany[strings.ToLower(strings.TrimSpace(o.Company))] = i
	}
	out := make([]pdfgen.Experience, 0, len(tailored))
	for _, t := range tailored {
		e := pdfgen.Experience{Company: t.Company, Title: t.Title, Bullets: t.Bullets}
		if i, ok := byCompany[strings.ToLower(strings.TrimSpace(t.Company))]; ok {
			o := original[i]
			e.Location = o.Location
			e.Dates = dateRange(o.StartDate, o.EndDate, o.Current)
		}
		out = append(out, e)
	}
	return out
}

func renderEducation(education []profileEducation) []pdfgen.Education {
	out := make([]pdfgen.Education, 0, len(education))
	for _, e := range education {
		out = append(out, pdfgen.Education{
			School: e.School, Degree: e.Degree, FieldOfStudy: e.FieldOfStudy,
			Dates: dateRange(e.StartDate, e.EndDate, false),
		})
	}
	return out
}

func dateRange(start, end string, current bool) string {
	switch {
	case start == "" && end == "":
		return ""
	case current:
		return start + " – Present"
	case end == "":
		return start
	default:
		return start + " – " + end
	}
}

func joinNonEmpty(sep string, parts ...string) string {
	var out []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, sep)
}

func nonEmpty(parts ...string) []string {
	var out []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
