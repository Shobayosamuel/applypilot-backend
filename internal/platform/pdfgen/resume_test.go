package pdfgen

import (
	"bytes"
	"testing"
)

func TestRenderProducesAPDF(t *testing.T) {
	data, err := Render(Resume{
		Name: "Jane Doe", Headline: "Backend Engineer", Email: "jane@example.com", Phone: "+234 700 000 0000",
		Location: "Lagos, Nigeria",
		Links: []Link{
			{Label: "linkedin.com/in/janedoe", URL: "https://linkedin.com/in/janedoe"},
			{Label: "github.com/janedoe", URL: "https://github.com/janedoe"},
		},
		Summary: "Backend engineer with 5 years building payment systems.",
		Skills:  []string{"Go", "PostgreSQL", "Redis"},
		Experience: []Experience{
			{Company: "Acme", Title: "Senior Engineer", Location: "Remote", Dates: "2022 – Present",
				Bullets: []string{"Built the payments ledger.", "Cut p99 latency by 40%."}},
		},
		Education:      []Education{{School: "University of Lagos", Degree: "BSc", FieldOfStudy: "Computer Science", Dates: "2014 – 2018"}},
		Certifications: []string{"AWS Certified Solutions Architect"},
		Languages:      []string{"English (native)"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("output is not a PDF: starts with %q", data[:min(20, len(data))])
	}
	if len(data) < 500 {
		t.Fatalf("suspiciously small PDF: %d bytes", len(data))
	}
}

func TestRenderHandlesEmptyResume(t *testing.T) {
	data, err := Render(Resume{Name: "No Content"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("output is not a PDF")
	}
}
