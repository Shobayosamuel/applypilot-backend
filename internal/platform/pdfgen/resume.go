// Package pdfgen renders a resume to PDF bytes. It knows nothing about AI,
// job tailoring, or profiles — callers hand it plain content to lay out.
package pdfgen

import (
	"bytes"
	"strings"

	"github.com/go-pdf/fpdf"
)

type Link struct {
	Label string // display text, e.g. "linkedin.com/in/jane"
	URL   string // e.g. "https://linkedin.com/in/jane"; "" renders as plain text
}

type Experience struct {
	Company, Title, Location, Dates string
	Bullets                         []string
}

type Education struct {
	School, Degree, FieldOfStudy, Dates string
}

type Resume struct {
	Name, Headline, Email, Phone, Location string
	Links                                  []Link
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

// accent is used for section headings and hyperlink text.
var accent = [3]int{37, 99, 235} // tailwind blue-600

// doc carries the pdf plus the cp1252 translator every piece of text must go
// through: fpdf's built-in fonts take single-byte cp1252, not UTF-8, so
// writing raw UTF-8 (accented names, en dashes, the "·" separator, "•"
// bullets) produces mojibake without it.
type doc struct {
	pdf *fpdf.Fpdf
	tr  func(string) string
}

func (d *doc) cell(w, h float64, text string, border string, ln int, align string) {
	d.pdf.CellFormat(w, h, d.tr(text), border, ln, align, false, 0, "")
}

// link draws text and overlays a clickable region on it, in the accent color.
func (d *doc) link(w, h float64, text, url string) {
	x, y := d.pdf.GetXY()
	if url != "" {
		d.pdf.SetTextColor(accent[0], accent[1], accent[2])
	}
	d.cell(w, h, text, "", 0, "L")
	if url != "" {
		d.pdf.LinkString(x, y, w, h, url)
		d.pdf.SetTextColor(0, 0, 0)
	}
}

func (d *doc) multiCell(w, h float64, text, align string) {
	d.pdf.MultiCell(w, h, d.tr(text), "", align, false)
}

// Render renders r as a single-column A4 PDF.
func Render(r Resume) ([]byte, error) {
	p := fpdf.New("P", "mm", "A4", "")
	p.SetMargins(margin, margin, margin)
	p.SetAutoPageBreak(true, margin)
	p.AddPage()
	d := &doc{pdf: p, tr: p.UnicodeTranslatorFromDescriptor("")}

	p.SetFont("Helvetica", "B", 19)
	d.cell(textWidth, 8.5, r.Name, "", 1, "L")

	if r.Headline != "" {
		p.SetFont("Helvetica", "", 11)
		p.SetTextColor(70, 70, 70)
		d.cell(textWidth, 6, r.Headline, "", 1, "L")
		p.SetTextColor(0, 0, 0)
	}

	d.contactLine(r)
	p.Ln(1.5)

	if r.Summary != "" {
		d.section("Summary")
		p.SetFont("Helvetica", "", 10)
		d.multiCell(textWidth, 5, r.Summary, "J")
		p.Ln(1)
	}

	if len(r.Skills) > 0 {
		d.section("Skills")
		p.SetFont("Helvetica", "", 10)
		d.multiCell(textWidth, 5, strings.Join(r.Skills, "  ·  "), "L")
		p.Ln(1)
	}

	if len(r.Experience) > 0 {
		d.section("Experience")
		for _, e := range r.Experience {
			d.entryHeading(e.Title, e.Dates)
			meta := joinNonEmpty("  ·  ", e.Company, e.Location)
			if meta != "" {
				p.SetFont("Helvetica", "", 9.5)
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
			d.entryHeading(joinNonEmpty(", ", e.School, joinNonEmpty(" ", e.Degree, e.FieldOfStudy)), e.Dates)
		}
	}

	if len(r.Certifications) > 0 {
		d.section("Certifications")
		p.SetFont("Helvetica", "", 10)
		d.multiCell(textWidth, 5, strings.Join(r.Certifications, "  ·  "), "L")
		p.Ln(1)
	}

	if len(r.Languages) > 0 {
		d.section("Languages")
		p.SetFont("Helvetica", "", 10)
		d.multiCell(textWidth, 5, strings.Join(r.Languages, "  ·  "), "L")
		p.Ln(1)
	}

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// contactLine prints email, phone, location and links on one row, each as
// its own clickable segment where a URL applies, separated by "·".
func (d *doc) contactLine(r Resume) {
	type segment struct {
		text, url string
	}
	segs := []segment{}
	if r.Email != "" {
		segs = append(segs, segment{r.Email, "mailto:" + r.Email})
	}
	if r.Phone != "" {
		segs = append(segs, segment{r.Phone, "tel:" + phoneDigits(r.Phone)})
	}
	if r.Location != "" {
		segs = append(segs, segment{r.Location, ""})
	}
	for _, l := range r.Links {
		if l.Label != "" {
			segs = append(segs, segment{l.Label, l.URL})
		}
	}
	if len(segs) == 0 {
		return
	}
	d.pdf.SetFont("Helvetica", "", 9.5)
	const sep = "   ·   "
	sepW := d.pdf.GetStringWidth(d.tr(sep))
	x := margin
	started := false
	for _, s := range segs {
		w := d.pdf.GetStringWidth(d.tr(s.text)) + 1
		needed := w
		if started {
			needed += sepW
		}
		if started && x+needed > margin+textWidth {
			d.pdf.Ln(5)
			x = margin
			started = false
		}
		if started {
			d.pdf.SetTextColor(170, 170, 170)
			d.cell(sepW, 5, sep, "", 0, "L")
			x += sepW
		}
		if s.url == "" {
			d.pdf.SetTextColor(90, 90, 90)
			d.cell(w, 5, s.text, "", 0, "L")
		} else {
			d.link(w, 5, s.text, s.url)
		}
		x += w
		started = true
	}
	d.pdf.SetTextColor(0, 0, 0)
	d.pdf.Ln(6)
}

// entryHeading prints a bold label on the left and its date range flush right.
func (d *doc) entryHeading(label, dates string) {
	d.pdf.SetFont("Helvetica", "B", 10.5)
	if dates == "" {
		d.cell(textWidth, 5.5, label, "", 1, "L")
		return
	}
	d.pdf.SetFont("Helvetica", "", 9.5)
	dateW := d.pdf.GetStringWidth(d.tr(dates)) + 2
	d.pdf.SetFont("Helvetica", "B", 10.5)
	d.cell(textWidth-dateW, 5.5, label, "", 0, "L")
	d.pdf.SetFont("Helvetica", "", 9.5)
	d.pdf.SetTextColor(90, 90, 90)
	d.cell(dateW, 5.5, dates, "", 1, "R")
	d.pdf.SetTextColor(0, 0, 0)
}

func (d *doc) section(title string) {
	d.pdf.Ln(1)
	d.pdf.SetFont("Helvetica", "B", 11)
	d.pdf.SetTextColor(accent[0], accent[1], accent[2])
	d.pdf.SetDrawColor(accent[0], accent[1], accent[2])
	d.cell(textWidth, 6, strings.ToUpper(title), "B", 1, "L")
	d.pdf.SetDrawColor(0, 0, 0)
	d.pdf.SetTextColor(0, 0, 0)
	d.pdf.Ln(1)
}

// bullet indents one line item with a round bullet; fpdf resets X to the
// page's left margin after a MultiCell, so no explicit reset is needed after.
func (d *doc) bullet(text string) {
	d.pdf.SetX(margin + 4)
	d.cell(4, 5, "•", "", 0, "L")
	x, _ := d.pdf.GetXY()
	d.multiCell(textWidth-(x-margin), 5, text, "L")
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

// phoneDigits strips everything but digits and a leading "+", for a tel: URI.
func phoneDigits(phone string) string {
	var b strings.Builder
	for i, r := range phone {
		if r == '+' && i == 0 {
			b.WriteRune(r)
		} else if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
