package ai

import (
	"encoding/json"
	"fmt"

	c "github.com/shobayosamuel/applypilot-backend/internal/platform/claude"
)

const baseSystem = `You are ApplyPilot, an assistant that helps a job seeker apply to jobs faster.

Rules you always follow:
- Use ONLY facts from the candidate profile. Never invent employers, titles, dates, degrees, certifications, metrics, or skills.
- If the profile lacks information needed to answer, leave the value empty rather than guessing.
- Write in a natural, confident, specific voice. No clichés ("I am writing to express my interest", "passionate", "synergy"), no em-dash overuse, no filler.
- Match the language of the job posting (usually English).`

// systemWithProfile puts the profile in its own cached block so repeated calls
// for the same user hit the prompt cache.
func systemWithProfile(profile json.RawMessage) []c.Block {
	return []c.Block{
		{Type: "text", Text: baseSystem},
		{Type: "text", Text: "<candidate_profile>\n" + string(profile) + "\n</candidate_profile>", CacheControl: &c.CacheControl{Type: "ephemeral"}},
	}
}

func jobBlock(j JobInput) string {
	desc := j.Description
	if len(desc) > 30000 {
		desc = desc[:30000]
	}
	return fmt.Sprintf("<job>\nTitle: %s\nCompany: %s\nLocation: %s\nURL: %s\n\nDescription:\n%s\n</job>", j.Title, j.Company, j.Location, j.URL, desc)
}

func userMsg(text string) []c.Message {
	return []c.Message{{Role: "user", Content: []c.Block{{Type: "text", Text: text}}}}
}

const (
	promptParseResume = "Extract this resume into the structured profile. Copy facts exactly; leave unknown fields empty. Normalise dates to YYYY-MM where the month is known."

	promptMatch = "\n\nAssess how well the candidate fits this job, like an experienced technical recruiter would. Weigh hard requirements (years, must-have skills, location/authorisation) above nice-to-haves. Be honest; do not inflate the score."

	promptCoverLetter = `

Write a cover letter for this job.
- Tone: %s
- 250-350 words, 3-4 short paragraphs, plain text (no markdown, no placeholders like [Company]).
- Open with a specific hook tied to the company or role, not a generic opener.
- Connect 2-3 concrete experiences from the profile to the job's most important requirements.
- End with a brief, confident close and the candidate's name.
Output only the letter.`

	promptTailor = `

Tailor the candidate's resume content for this job.
- Rewrite the headline and a 2-3 sentence summary aimed at this role.
- Order skills by relevance to the job (only skills the profile supports).
- For each relevant role, rewrite 3-5 bullets: action verb first, emphasise work that maps to the job's requirements, use the posting's terminology where it truthfully applies, keep any metrics exactly as given.
- Never add experience, tools, or numbers that are not in the profile.`

	promptAnswer = `

Answer this application question on the candidate's behalf, in first person.
<question>%s</question>

- Stay under %d characters.
- Be specific: reference real experience from the profile that fits this company and role.
- Plain text only, no preamble, no quotation marks around the answer.`

	promptMapFields = `
</form_fields>

Fill these job application form fields for the candidate.
- For select/radio fields, the value MUST be copied exactly from that field's options, or "" if none fit.
- For checkboxes return "true" or "false".
- For yes/no questions about work authorisation, sponsorship, relocation etc. use the profile's preferences; if the profile doesn't say, return "".
- For EEO/demographic questions use the profile's eeo answers; if absent, choose a "decline to answer" option when one exists.
- Short free-text questions: a concise truthful answer. Long essay questions: return "" (they are handled separately).
- Never invent facts. Return "" when unsure.`

	promptExtractJob = "\n\nExtract the job posting. Copy the full description faithfully (responsibilities, requirements, benefits), trimming site navigation and boilerplate."
)

// ---------- tool schemas (structured output) ----------

var toolSaveProfile = c.Tool{Name: "save_profile", Description: "Save the extracted candidate profile.", InputSchema: c.Obj(c.Schema{
	"first_name": c.Str(), "last_name": c.Str(), "email": c.Str(), "phone": c.Str(),
	"headline": c.Str(), "summary": c.Str(),
	"location": c.Obj(c.Schema{"address": c.Str(), "city": c.Str(), "state": c.Str(), "country": c.Str(), "postal_code": c.Str()}),
	"links":    c.Obj(c.Schema{"linkedin": c.Str(), "github": c.Str(), "portfolio": c.Str(), "website": c.Str(), "twitter": c.Str()}),
	"work_experience": c.Arr(c.Obj(c.Schema{
		"company": c.Str(), "title": c.Str(), "location": c.Str(),
		"start_date": c.StrD("YYYY-MM or YYYY"), "end_date": c.StrD("YYYY-MM, YYYY, or empty if current"),
		"current": c.Bool(), "description": c.StrD("Bullet points separated by newlines, copied faithfully"),
	})),
	"education": c.Arr(c.Obj(c.Schema{
		"school": c.Str(), "degree": c.Str(), "field_of_study": c.Str(), "start_date": c.Str(), "end_date": c.Str(), "gpa": c.Str(),
	})),
	"skills": c.Arr(c.Str()), "languages": c.Arr(c.Str()), "certifications": c.Arr(c.Str()),
}, "first_name", "last_name", "work_experience", "education", "skills")}

var toolReportMatch = c.Tool{Name: "report_match", Description: "Report the fit assessment.", InputSchema: c.Obj(c.Schema{
	"score":          c.Schema{"type": "integer", "minimum": 0, "maximum": 100, "description": "Overall fit 0-100"},
	"verdict":        c.Schema{"type": "string", "enum": []string{"strong", "good", "fair", "weak"}},
	"summary":        c.StrD("2-3 sentences explaining the score"),
	"matched_skills": c.Arr(c.Str()),
	"missing_skills": c.ArrD(c.Str(), "Requirements the profile does not show"),
	"keywords":       c.ArrD(c.Str(), "Important ATS keywords from the posting"),
	"suggestions":    c.ArrD(c.Str(), "Concrete, truthful ways to strengthen the application"),
}, "score", "verdict", "summary", "matched_skills", "missing_skills", "keywords", "suggestions")}

var toolTailor = c.Tool{Name: "save_tailored_resume", Description: "Save the tailored resume content.", InputSchema: c.Obj(c.Schema{
	"headline": c.Str(), "summary": c.Str(), "skills": c.Arr(c.Str()),
	"experience":     c.Arr(c.Obj(c.Schema{"company": c.Str(), "title": c.Str(), "bullets": c.Arr(c.Str())})),
	"keywords_added": c.ArrD(c.Str(), "Posting keywords now reflected in the content"),
	"notes":          c.StrD("Short advice on gaps the candidate should address honestly"),
}, "headline", "summary", "skills", "experience", "keywords_added", "notes")}

var toolFillFields = c.Tool{Name: "fill_fields", Description: "Return a value for each field id.", InputSchema: c.Obj(c.Schema{
	"values": c.Arr(c.Obj(c.Schema{"id": c.Str(), "value": c.Str()})),
}, "values")}

var toolSaveJob = c.Tool{Name: "save_job", Description: "Save the extracted job.", InputSchema: c.Obj(c.Schema{
	"title": c.Str(), "company": c.Str(), "location": c.Str(), "description": c.Str(),
}, "title", "company", "location", "description")}
