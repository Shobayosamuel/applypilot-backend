// Package pdfgen renders a resume to PDF bytes. It knows nothing about AI,
// job tailoring, or profiles — callers hand it plain content to lay out.
package pdfgen

import (
	"bytes"
	"strings"

	"github.com/go-pdf/fpdf"
)

type Experience struct {
	Company, Title, Location, Dates string
	Bullets                         []string
}

type Education struct {
	School, Degree, FieldOfStudy, Dates string
}

type Resume struct {
	Name, Headline, Email, Phone, Location string
	Links                                  []string // rendered as-is, e.g. "linkedin.com/in/jane"
	Summary                                string
	Skills                                 []string
	Experience                             []Experience
	Education                              []Education
	Certifications                         []string
	Languages                              []string
}

const (
	margin    = 18.0
	pageWidth = 210.0 // A4 mm
	textWidth = pageWidth - 2*margin
)

// doc carries the pdf plus the cp1252 translator every piece of text must go
// through: fpdf's built-in fonts take single-byte cp1252, not UTF-8, so
// writing raw UTF-8 (accented names, en dashes, the "·" separator) produces
// mojibake without it.
type doc struct {
	pdf *fpdf.Fpdf
	tr  func(string) string
}

func (d *doc) cell(w, h float64, text string, border string, ln int, align string) {
	d.pdf.CellFormat(w, h, d.tr(text), border, ln, align, false, 0, "")
}

func (d *doc) multiCell(w, h float64, text string) {
	d.pdf.MultiCell(w, h, d.tr(text), "", "L", false)
}

// Render renders r as a single-column A4 PDF.
func Render(r Resume) ([]byte, error) {
	p := fpdf.New("P", "mm", "A4", "")
	p.SetMargins(margin, margin, margin)
	p.SetAutoPageBreak(true, margin)
	p.AddPage()
	d := &doc{pdf: p, tr: p.UnicodeTranslatorFromDescriptor("")}

	p.SetFont("Helvetica", "B", 18)
	d.cell(textWidth, 8, r.Name, "", 1, "L")

	if r.Headline != "" {
		p.SetFont("Helvetica", "", 11)
		p.SetTextColor(70, 70, 70)
		d.cell(textWidth, 6, r.Headline, "", 1, "L")
		p.SetTextColor(0, 0, 0)
	}

	contact := joinNonEmpty(" | ", r.Email, r.Phone, r.Location)
	if contact != "" {
		p.SetFont("Helvetica", "", 9)
		p.SetTextColor(90, 90, 90)
		d.cell(textWidth, 5, contact, "", 1, "L")
	}
	if len(r.Links) > 0 {
		d.cell(textWidth, 5, strings.Join(r.Links, "  |  "), "", 1, "L")
	}
	p.SetTextColor(0, 0, 0)
	p.Ln(2)

	if r.Summary != "" {
		d.section("Summary")
		d.body(r.Summary)
	}

	if len(r.Skills) > 0 {
		d.section("Skills")
		d.body(strings.Join(r.Skills, " · "))
	}

	if len(r.Experience) > 0 {
		d.section("Experience")
		for _, e := range r.Experience {
			p.SetFont("Helvetica", "B", 10.5)
			d.cell(textWidth, 5.5, joinNonEmpty(" — ", e.Title, e.Company), "", 1, "L")
			meta := joinNonEmpty("  ·  ", e.Location, e.Dates)
			if meta != "" {
				p.SetFont("Helvetica", "I", 9)
				p.SetTextColor(90, 90, 90)
				d.cell(textWidth, 5, meta, "", 1, "L")
				p.SetTextColor(0, 0, 0)
			}
			p.SetFont("Helvetica", "", 10)
			for _, b := range e.Bullets {
				if b = strings.TrimSpace(b); b != "" {
					d.bullet(b)
				}
			}
			p.Ln(1.5)
		}
	}

	if len(r.Education) > 0 {
		d.section("Education")
		for _, e := range r.Education {
			p.SetFont("Helvetica", "B", 10.5)
			d.cell(textWidth, 5.5, joinNonEmpty(", ", e.School, joinNonEmpty(" ", e.Degree, e.FieldOfStudy)), "", 1, "L")
			if e.Dates != "" {
				p.SetFont("Helvetica", "I", 9)
				p.SetTextColor(90, 90, 90)
				d.cell(textWidth, 5, e.Dates, "", 1, "L")
				p.SetTextColor(0, 0, 0)
			}
		}
	}

	if len(r.Certifications) > 0 {
		d.section("Certifications")
		d.body(strings.Join(r.Certifications, " · "))
	}

	if len(r.Languages) > 0 {
		d.section("Languages")
		d.body(strings.Join(r.Languages, " · "))
	}

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (d *doc) section(title string) {
	d.pdf.Ln(1)
	d.pdf.SetFont("Helvetica", "B", 11)
	d.pdf.SetTextColor(30, 30, 30)
	d.cell(textWidth, 6, strings.ToUpper(title), "B", 1, "L")
	d.pdf.SetTextColor(0, 0, 0)
	d.pdf.Ln(1)
}

func (d *doc) body(text string) {
	d.pdf.SetFont("Helvetica", "", 10)
	d.multiCell(textWidth, 5, text)
	d.pdf.Ln(1)
}

// bullet indents one line item; fpdf resets X to the page's left margin after
// a MultiCell, so no explicit reset is needed afterward.
func (d *doc) bullet(text string) {
	d.pdf.SetX(margin + 4)
	d.cell(4, 5, "-", "", 0, "L")
	x, _ := d.pdf.GetXY()
	d.multiCell(textWidth-(x-margin), 5, text)
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
