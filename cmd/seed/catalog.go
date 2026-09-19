package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"strings"
	"sync"
)

// Content types the catalog stores under. Listings infer a type from the key's
// extension, but the upload declares it and the download serves it, so the
// seed sends the real one.
const (
	typePNG  = "image/png"
	typeJPEG = "image/jpeg"
	typeSVG  = "image/svg+xml"
	typePDF  = "application/pdf"
	typeCSV  = "text/csv"
	typeJSON = "application/json"
	typeCSS  = "text/css"
	typeMD   = "text/markdown"
	typeText = "text/plain"
)

// asset is one catalog entry: the key it uploads under, its declared content
// type, and the bytes. Folders are key prefixes, so the catalog's keys shape
// the tree the studio's browser shows.
type asset struct {
	Key         string
	ContentType string
	Data        []byte
}

// catalog is the mock asset tree. Every file is genuine for its type — the
// images decode, the PDFs open, the data parses — so downloads from the studio
// look right, and sizes spread from bytes to hundreds of kilobytes so the size
// column and sorting have something to show. Root files sit beside four
// folders, one of them three levels deep, to exercise folder navigation.
//
// The tree is deterministic and the images cost real CPU to paint, so it is
// built once per process.
func catalog() []asset {
	catalogOnce.Do(func() { catalogEntries = buildCatalog() })
	return catalogEntries
}

var (
	catalogOnce    sync.Once
	catalogEntries []asset
)

func buildCatalog() []asset {
	return []asset{
		// Root files.
		{Key: "README.md", ContentType: typeMD, Data: readme()},
		{Key: "robots.txt", ContentType: typeText, Data: []byte("User-agent: *\nAllow: /\n\nSitemap: https://docs.example.com/sitemap.xml\n")},
		{Key: "favicon.svg", ContentType: typeSVG, Data: svgMark()},

		// Brand and page imagery.
		{Key: "images/logo.png", ContentType: typePNG, Data: pngImage(256, 256, brandLight)},
		{Key: "images/logo-dark.png", ContentType: typePNG, Data: pngImage(256, 256, brandDark)},
		{Key: "images/og-card.png", ContentType: typePNG, Data: pngImage(1200, 630, brandLight)},
		{Key: "images/hero.jpg", ContentType: typeJPEG, Data: jpegImage(1280, 720, brandLight)},
		{Key: "images/hero-dark.jpg", ContentType: typeJPEG, Data: jpegImage(1280, 720, brandDark)},
		{Key: "images/icons/menu.svg", ContentType: typeSVG, Data: svgIcon("M3 6h18M3 12h18M3 18h18")},
		{Key: "images/icons/search.svg", ContentType: typeSVG, Data: svgIcon("M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14zM20 20l-4-4")},
		{Key: "images/icons/close.svg", ContentType: typeSVG, Data: svgIcon("M6 6l12 12M18 6L6 18")},
		{Key: "images/icons/external.svg", ContentType: typeSVG, Data: svgIcon("M14 4h6v6M20 4l-9 9M19 14v6H4V5h6")},
		{Key: "images/screenshots/dashboard.png", ContentType: typePNG, Data: pngImage(800, 500, screenLight)},
		{Key: "images/screenshots/settings.png", ContentType: typePNG, Data: pngImage(800, 500, screenLight)},
		{Key: "images/screenshots/mobile/home.png", ContentType: typePNG, Data: pngImage(390, 844, screenDark)},
		{Key: "images/screenshots/mobile/search.png", ContentType: typePNG, Data: pngImage(390, 844, screenDark)},

		// Downloadable documents.
		{Key: "docs/getting-started.pdf", ContentType: typePDF, Data: pdfDocument("Getting Started", 3)},
		{Key: "docs/brand-guidelines.pdf", ContentType: typePDF, Data: pdfDocument("Brand Guidelines", 12)},
		{Key: "docs/press-kit.pdf", ContentType: typePDF, Data: pdfDocument("Press Kit", 6)},

		// Data files.
		{Key: "data/pricing.csv", ContentType: typeCSV, Data: pricingCSV(2500)},
		{Key: "data/regions.json", ContentType: typeJSON, Data: regionsJSON()},

		// Stylesheets.
		{Key: "styles/theme.css", ContentType: typeCSS, Data: themeCSS()},
		{Key: "styles/print.css", ContentType: typeCSS, Data: []byte("@media print {\n  nav, footer, .sidebar { display: none; }\n  main { max-width: none; }\n}\n")},
	}
}

// palette drives the generated images: a background gradient between two
// colors and an accent for the mark drawn over it.
type palette struct {
	from, to, accent color.RGBA
}

var (
	brandLight  = palette{from: rgb(240, 244, 255), to: rgb(199, 210, 254), accent: rgb(79, 70, 229)}
	brandDark   = palette{from: rgb(17, 24, 39), to: rgb(49, 46, 129), accent: rgb(165, 180, 252)}
	screenLight = palette{from: rgb(250, 250, 250), to: rgb(229, 231, 235), accent: rgb(107, 114, 128)}
	screenDark  = palette{from: rgb(24, 24, 27), to: rgb(39, 39, 42), accent: rgb(161, 161, 170)}
)

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{R: r, G: g, B: b, A: 255} }

// paint renders the image: a diagonal gradient with a centered circle of the
// accent color and a faint grid, so every generated file is visibly a picture
// rather than a flat block, and compresses to a realistic size.
func paint(w, h int, p palette) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	cx, cy := float64(w)/2, float64(h)/2
	radius := math.Min(cx, cy) * 0.6
	for y := range h {
		for x := range w {
			t := (float64(x)/float64(w) + float64(y)/float64(h)) / 2
			c := mix(p.from, p.to, t)
			if x%40 == 0 || y%40 == 0 {
				c = mix(c, p.accent, 0.08)
			}
			dx, dy := float64(x)-cx, float64(y)-cy
			if d := math.Sqrt(dx*dx + dy*dy); d < radius {
				// Soft edge over the last two pixels of the circle.
				c = mix(c, p.accent, math.Min(1, (radius-d)/2))
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// mix linearly interpolates two colors; t is clamped to [0, 1].
func mix(a, b color.RGBA, t float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	return color.RGBA{
		R: channel(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: channel(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: channel(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 255,
	}
}

// channel rounds and clamps a float to one 8-bit color channel.
func channel(v float64) uint8 { return uint8(math.Round(math.Max(0, math.Min(255, v)))) }

func pngImage(w, h int, p palette) []byte {
	var buf bytes.Buffer
	// Encoding into a buffer cannot fail for a valid image.
	_ = png.Encode(&buf, paint(w, h, p))
	return buf.Bytes()
}

func jpegImage(w, h int, p palette) []byte {
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, paint(w, h, p), &jpeg.Options{Quality: 85})
	return buf.Bytes()
}

// svgIcon is a 24px stroked icon from one path.
func svgIcon(d string) []byte {
	return []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="` + d + `"/></svg>` + "\n")
}

// svgMark is the brand mark: a rounded square with a circle.
func svgMark() []byte {
	return []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width="64" height="64"><rect width="64" height="64" rx="14" fill="#4f46e5"/><circle cx="32" cy="32" r="16" fill="#eef2ff"/></svg>` + "\n")
}

// pdfDocument builds a valid PDF-1.4 with the given number of text pages. The
// cross-reference table carries real byte offsets, so strict readers open it
// without repair; the page count scales the size.
func pdfDocument(title string, pages int) []byte {
	// Objects: 1 catalog, 2 pages, 3 font, then a page + content pair per page.
	var objects []string
	objects = append(objects,
		"<< /Type /Catalog /Pages 2 0 R >>",
		"", // Pages: filled once the page object numbers are known.
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	)
	kids := make([]string, 0, pages)
	for i := 1; i <= pages; i++ {
		pageNum := len(objects) + 1
		contentNum := pageNum + 1
		kids = append(kids, fmt.Sprintf("%d 0 R", pageNum))
		objects = append(objects, fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>",
			contentNum))
		var content strings.Builder
		fmt.Fprintf(&content, "BT /F1 24 Tf 72 720 Td (%s) Tj ET\n", title)
		for line := range 40 {
			fmt.Fprintf(&content, "BT /F1 11 Tf 72 %d Td (Page %d, line %d: placeholder body text for the seed document.) Tj ET\n",
				680-line*15, i, line+1)
		}
		objects = append(objects, fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", content.Len(), content.String()))
	}
	objects[1] = fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), pages)

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, 0, len(objects))
	for i, obj := range objects {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, off := range offsets {
		fmt.Fprintf(&out, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return out.Bytes()
}

// pricingCSV is a deterministic price list with the given number of rows.
func pricingCSV(rows int) []byte {
	regions := []string{"us-east", "us-west", "eu-central", "ap-southeast"}
	tiers := []string{"free", "starter", "team", "enterprise"}
	var out strings.Builder
	out.WriteString("sku,region,tier,seats,monthly_usd,annual_usd\n")
	for i := range rows {
		seats := 1 + (i*7)%250
		monthly := float64(seats) * (4 + float64(i%5)*2.5)
		fmt.Fprintf(&out, "SKU-%05d,%s,%s,%d,%.2f,%.2f\n",
			i+1, regions[i%len(regions)], tiers[(i/3)%len(tiers)], seats, monthly, monthly*10)
	}
	return []byte(out.String())
}

func regionsJSON() []byte {
	return []byte(`{
  "regions": [
    { "id": "us-east", "name": "US East", "location": "Virginia", "default": true },
    { "id": "us-west", "name": "US West", "location": "Oregon", "default": false },
    { "id": "eu-central", "name": "EU Central", "location": "Frankfurt", "default": false },
    { "id": "ap-southeast", "name": "Asia Pacific", "location": "Singapore", "default": false }
  ]
}
`)
}

func themeCSS() []byte {
	var out strings.Builder
	out.WriteString(":root {\n  --brand: #4f46e5;\n  --brand-soft: #eef2ff;\n  --ink: #111827;\n  --paper: #ffffff;\n}\n\n")
	out.WriteString("@media (prefers-color-scheme: dark) {\n  :root {\n    --brand: #a5b4fc;\n    --brand-soft: #312e81;\n    --ink: #f9fafb;\n    --paper: #111827;\n  }\n}\n\n")
	for _, step := range []int{1, 2, 3, 4, 6, 8, 12, 16, 24, 32} {
		fmt.Fprintf(&out, ".p-%d { padding: %dpx; }\n.m-%d { margin: %dpx; }\n.gap-%d { gap: %dpx; }\n", step, step*4, step, step*4, step, step*4)
	}
	return []byte(out.String())
}

func readme() []byte {
	return []byte(`# Site assets

Static files referenced by this app's pages.

- ` + "`images/`" + ` — brand marks, hero art, icons, and screenshots.
- ` + "`docs/`" + ` — downloadable PDFs linked from the docs.
- ` + "`data/`" + ` — tables the pricing and regions pages render.
- ` + "`styles/`" + ` — stylesheets loaded by the published site.

Everything here was written by ` + "`cmd/seed`" + ` and can be replaced freely.
`)
}
