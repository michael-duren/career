package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif" // favicon formats the logo service may return
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Company logos (docs/redesign/pages/companies.md): the Companies page shows
// each company's favicon in single-color ink. The browser can only paint an
// image through a CSS mask when it is same-origin (or CORS-enabled), so the
// server fetches the icon from a favicon service, turns it into an alpha mask
// and serves it from /api/companies/logo. Only the hostname leaves the server;
// the upstream URL is fixed, so a saved company URL cannot point the fetch at
// an internal address.

const (
	defaultLogoUpstream = "https://www.google.com/s2/favicons?sz=128&domain="
	maxLogoBytes        = 512 << 10
	maxLogoPixels       = 1024 * 1024
	logoTTL             = 7 * 24 * time.Hour
	// A domain the service has no icon for is retried after a while; a
	// network error sooner.
	logoMissingTTL = 6 * time.Hour
	logoErrorTTL   = 5 * time.Minute
	maxLogoEntries = 1000
)

var errNoLogo = errors.New("no logo for this domain")

// logoHost is a DNS name with at least two labels whose last label is not
// all digits (so IP literals are refused).
var logoHost = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9-]*[a-z][a-z0-9-]*$`)

// normalizeLogoHost lowercases host, drops a leading "www." and validates it.
func normalizeLogoHost(host string) (string, bool) {
	host = strings.TrimPrefix(strings.ToLower(strings.TrimSuffix(host, ".")), "www.")
	if len(host) > 253 || !logoHost.MatchString(host) {
		return "", false
	}
	return host, true
}

type logoEntry struct {
	png     []byte // nil when there is no usable logo
	expires time.Time
}

// logoCache keeps converted logos in memory. Browsers cache the responses
// too (Cache-Control), so the server fetches a domain at most once per TTL.
type logoCache struct {
	mu      sync.Mutex
	entries map[string]logoEntry
	group   singleflight.Group
}

func (c *logoCache) get(host string, now time.Time) (logoEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[host]
	if !ok || now.After(e.expires) {
		return logoEntry{}, false
	}
	return e, true
}

func (c *logoCache) put(host string, e logoEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil || len(c.entries) >= maxLogoEntries {
		// Simple bound: start over rather than track recency. A personal
		// workspace tracks well under this many companies.
		c.entries = map[string]logoEntry{}
	}
	c.entries[host] = e
}

// companyLogo serves GET /api/companies/logo?domain=<host>: the domain's icon
// as a PNG whose alpha is the logo shape (pixels are black; the page paints
// them in --ink with a mask), or 404 when there is no usable icon.
func (s *Server) companyLogo(w http.ResponseWriter, r *http.Request) {
	host, ok := normalizeLogoHost(r.URL.Query().Get("domain"))
	if !ok {
		respond(w, 400, map[string]string{"error": "domain must be a host name like example.com"})
		return
	}
	data, err := s.logo(r.Context(), host)
	if err != nil {
		// Let the browser remember a missing logo for a while too, so a list
		// re-render doesn't ask again for every monogram.
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "private, max-age=3600")
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"error":"No logo for this domain"}` + "\n"))
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=604800")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'")
	_, _ = w.Write(data)
}

// logo returns the cached mask for host, fetching it once when missing.
func (s *Server) logo(ctx context.Context, host string) ([]byte, error) {
	now := s.clock()
	if e, ok := s.logos.get(host, now); ok {
		if e.png == nil {
			return nil, errNoLogo
		}
		return e.png, nil
	}
	v, err, _ := s.logos.group.Do(host, func() (any, error) {
		// Detached from the request so one cancelled page load doesn't fail
		// (and briefly cache a failure for) everyone waiting on this fetch.
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		data, err := s.fetchLogo(fetchCtx, host)
		ttl := logoTTL
		switch {
		case errors.Is(err, errNoLogo):
			ttl = logoMissingTTL
		case err != nil:
			ttl = logoErrorTTL
		}
		s.logos.put(host, logoEntry{png: data, expires: s.clock().Add(ttl)})
		return data, err
	})
	if err != nil {
		return nil, err
	}
	return v.([]byte), nil
}

func (s *Server) fetchLogo(ctx context.Context, host string) ([]byte, error) {
	upstream := s.logoUpstream
	if upstream == "" {
		upstream = defaultLogoUpstream
	}
	req, err := http.NewRequestWithContext(ctx, "GET", upstream+url.QueryEscape(host), nil)
	if err != nil {
		return nil, err
	}
	client := s.logoClient
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	// The service answers 404 (with a generic globe) for unknown domains.
	if res.StatusCode != http.StatusOK {
		if res.StatusCode >= 500 {
			return nil, fmt.Errorf("logo service: HTTP %d", res.StatusCode)
		}
		return nil, errNoLogo
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, maxLogoBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxLogoBytes {
		return nil, errNoLogo
	}
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(raw)); err != nil || cfg.Width*cfg.Height > maxLogoPixels {
		return nil, errNoLogo
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, errNoLogo
	}
	mask, ok := inkMask(img)
	if !ok {
		return nil, errNoLogo
	}
	var out bytes.Buffer
	if err := png.Encode(&out, mask); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// inkMask turns a colored icon into a black image whose alpha is the logo
// shape, so it reads as one ink color on any theme:
//   - an icon on an opaque background (white square, colored tile filling the
//     icon) keeps the pixels that differ from the corner color;
//   - an icon on a transparent background keeps its silhouette, unless it is a
//     single-color badge with a mark knocked out of it (a white glyph on a
//     rounded tile), which keeps the mark instead of a solid blob.
//
// It reports false when the result is almost empty or almost solid, which
// would read as a blank or filled square; the page then shows the monogram.
func inkMask(src image.Image) (*image.NRGBA, bool) {
	b := src.Bounds()
	if b.Dx() < 8 || b.Dy() < 8 {
		return nil, false
	}
	px := func(x, y int) color.NRGBA { return color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA) }
	corners := []color.NRGBA{px(b.Min.X, b.Min.Y), px(b.Max.X-1, b.Min.Y), px(b.Min.X, b.Max.Y-1), px(b.Max.X-1, b.Max.Y-1)}
	var cr, cg, cb, ca int
	for _, c := range corners {
		cr, cg, cb, ca = cr+int(c.R), cg+int(c.G), cb+int(c.B), ca+int(c.A)
	}
	bg := color.NRGBA{uint8(cr / 4), uint8(cg / 4), uint8(cb / 4), 255}
	useBackground := ca/4 >= 128
	if !useBackground {
		if badge, ok := badgeColor(src); ok {
			bg, useBackground = badge, true
		}
	}

	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	var covered, total int
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := px(x, y)
			alpha := int(c.A)
			if useBackground {
				alpha = alpha * ramp(colorDistance(c, bg)) / 255
			}
			out.SetNRGBA(x-b.Min.X, y-b.Min.Y, color.NRGBA{A: uint8(alpha)})
			total++
			if alpha >= 128 {
				covered++
			}
		}
	}
	share := float64(covered) / float64(total)
	return out, share >= 0.02 && share <= 0.9
}

// badgeColor finds the color of a single-color tile with a mark inside it: the
// rim of the opaque shape (opaque pixels next to transparency or the image
// edge) is almost all one color, and some pixels inside clearly differ. A
// silhouette in one color has no differing pixels; a multi-color logo has a
// mixed rim. Both keep their silhouette.
func badgeColor(src image.Image) (color.NRGBA, bool) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	pixels := make([]color.NRGBA, w*h)
	for y := range h {
		for x := range w {
			pixels[y*w+x] = color.NRGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
		}
	}
	opaque := func(x, y int) bool { return x >= 0 && y >= 0 && x < w && y < h && pixels[y*w+x].A >= 128 }
	type bin struct{ n, r, g, b int }
	bins := map[int]*bin{}
	var rim []color.NRGBA
	for y := range h {
		for x := range w {
			if !opaque(x, y) || (opaque(x-1, y) && opaque(x+1, y) && opaque(x, y-1) && opaque(x, y+1)) {
				continue
			}
			c := pixels[y*w+x]
			rim = append(rim, c)
			k := int(c.R>>4)<<8 | int(c.G>>4)<<4 | int(c.B>>4)
			if bins[k] == nil {
				bins[k] = &bin{}
			}
			e := bins[k]
			e.n, e.r, e.g, e.b = e.n+1, e.r+int(c.R), e.g+int(c.G), e.b+int(c.B)
		}
	}
	if len(rim) == 0 {
		return color.NRGBA{}, false
	}
	var top *bin
	for _, e := range bins {
		if top == nil || e.n > top.n {
			top = e
		}
	}
	tile := color.NRGBA{uint8(top.r / top.n), uint8(top.g / top.n), uint8(top.b / top.n), 255}
	var near int
	for _, c := range rim {
		if colorDistance(c, tile) <= 48 {
			near++
		}
	}
	var solid, far int
	for _, c := range pixels {
		if c.A < 128 {
			continue
		}
		solid++
		if colorDistance(c, tile) > 96 {
			far++
		}
	}
	return tile, float64(near)/float64(len(rim)) >= 0.8 && float64(far)/float64(solid) >= 0.04
}

// colorDistance is the largest per-channel difference, 0-255.
func colorDistance(a, b color.NRGBA) int {
	d := 0
	for _, v := range []int{int(a.R) - int(b.R), int(a.G) - int(b.G), int(a.B) - int(b.B)} {
		if v < 0 {
			v = -v
		}
		d = max(d, v)
	}
	return d
}

// ramp maps a color distance to alpha: anti-aliased edges between 24 and 96
// fade, so outlines stay smooth.
func ramp(d int) int {
	switch {
	case d <= 24:
		return 0
	case d >= 96:
		return 255
	default:
		return (d - 24) * 255 / 72
	}
}
