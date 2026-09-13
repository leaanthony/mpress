package components

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// QR renders a QR code as inline SVG at build time.
// Usage: :::qr{url="https://example.com" size="200" label="Scan me"}
//
// Attributes:
//   - url (required): The URL or text to encode
//   - size: SVG width/height in pixels (default: 200)
//   - label: Caption text below the QR code
//   - level: Error correction level: L, M, Q, H (default: M)
type QR struct {
	Meta    map[string]string
	Content string
}

func (q *QR) Parse(content string) error {
	q.Content = strings.TrimSpace(content)
	return nil
}

func (q *QR) Render() (string, error) {
	url := q.Meta["url"]
	if url == "" {
		// Fall back to body content as the data to encode
		url = q.Content
	}
	if url == "" {
		return "", fmt.Errorf("qr component requires url attribute or body content")
	}

	size := 200
	if s := q.Meta["size"]; s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			size = v
		}
	}

	level := qrcode.Medium
	if l := strings.ToUpper(q.Meta["level"]); l != "" {
		switch l {
		case "L":
			level = qrcode.Low
		case "M":
			level = qrcode.Medium
		case "Q":
			level = qrcode.High
		case "H":
			level = qrcode.Highest
		}
	}

	label := q.Meta["label"]

	// Generate QR code bitmap
	qr, err := qrcode.New(url, level)
	if err != nil {
		return "", fmt.Errorf("generating QR code: %w", err)
	}
	qr.DisableBorder = false
	bitmap := qr.Bitmap()

	modules := len(bitmap)
	if modules == 0 {
		return "", fmt.Errorf("empty QR code")
	}

	// Render as SVG using rect elements for each dark module
	cellSize := float64(size) / float64(modules)

	var b strings.Builder
	b.WriteString(`<figure class="mpress-qr" role="img" aria-label="QR code`)
	if label != "" {
		b.WriteString(`: `)
		b.WriteString(html.EscapeString(label))
	}
	b.WriteString(`">`)

	b.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" shape-rendering="crispEdges">`,
		size, size, size, size))

	// White background
	b.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" fill="#fff"/>`, size, size))

	// Draw dark modules
	for y, row := range bitmap {
		for x, dark := range row {
			if dark {
				rx := float64(x) * cellSize
				ry := float64(y) * cellSize
				// Use ceil for width/height to avoid gaps between modules
				w := cellSize + 0.5
				h := cellSize + 0.5
				b.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="#000"/>`,
					rx, ry, w, h))
			}
		}
	}

	b.WriteString(`</svg>`)

	// Optional label
	if label != "" {
		b.WriteString(fmt.Sprintf(`<figcaption class="mpress-qr-label">%s</figcaption>`, html.EscapeString(label)))
	}

	b.WriteString(`</figure>`)

	return b.String(), nil
}
