package server

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/michael-duren/career-strategy/internal/modules"
)

// icon draws a size×size image: bg everywhere, fg in the centre square.
func icon(size int, bg, fg color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			c := bg
			if x >= size/4 && x < size*3/4 && y >= size/4 && y < size*3/4 {
				c = fg
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

var (
	white  = color.NRGBA{255, 255, 255, 255}
	black  = color.NRGBA{0, 0, 0, 255}
	purple = color.NRGBA{120, 60, 200, 255}
	clear  = color.NRGBA{}
)

// coverage is the share of mask pixels at least half opaque, and whether the
// centre (the mark) and the corner (the background) are inked.
func coverage(m *image.NRGBA) (share float64, centre, corner bool) {
	b := m.Bounds()
	var on int
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if m.NRGBAAt(x, y).A >= 128 {
				on++
			}
		}
	}
	return float64(on) / float64(b.Dx()*b.Dy()), m.NRGBAAt(b.Dx()/2, b.Dy()/2).A >= 128, m.NRGBAAt(0, 0).A >= 128
}

func TestInkMask(t *testing.T) {
	// A white glyph knocked out of a purple tile with transparent corners.
	badge := icon(64, clear, white)
	for y := 4; y < 60; y++ {
		for x := 4; x < 60; x++ {
			if badge.NRGBAAt(x, y) == clear {
				badge.SetNRGBA(x, y, purple)
			}
		}
	}
	// A two-color mark on transparent (left half purple, right half white): the rim is
	// mixed, so it is a logo in two colors, not a tile, and keeps its whole silhouette.
	twoTone := icon(64, clear, purple)
	for y := 16; y < 48; y++ {
		for x := 32; x < 48; x++ {
			twoTone.SetNRGBA(x, y, white)
		}
	}
	for _, tc := range []struct {
		name   string
		img    image.Image
		ok     bool
		centre bool
	}{
		{"mark on white", icon(64, white, purple), true, true},
		{"white mark on a full-bleed tile", icon(64, purple, white), true, true},
		{"silhouette on transparent", icon(64, clear, purple), true, true},
		{"badge keeps the knocked-out mark", badge, true, true},
		{"two-color mark keeps its silhouette", twoTone, true, true},
		{"solid square", icon(64, black, black), false, false},
		{"blank", icon(64, clear, clear), false, false},
		{"too small", icon(4, white, black), false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, ok := inkMask(tc.img)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !ok {
				return
			}
			share, centre, corner := coverage(m)
			if tc.name == "badge keeps the knocked-out mark" && m.NRGBAAt(8, 8).A != 0 {
				t.Fatal("badge tile was inked; only the mark should be")
			}
			if tc.name == "two-color mark keeps its silhouette" && share < 0.24 {
				t.Fatalf("two-color mark lost its white half: coverage %.2f", share)
			}
			if centre != tc.centre || corner {
				t.Fatalf("centre inked %v (want %v), corner inked %v (want false)", centre, tc.centre, corner)
			}
			if c := m.NRGBAAt(m.Bounds().Dx()/2, m.Bounds().Dy()/2); c.R|c.G|c.B != 0 {
				t.Fatalf("mask pixels must be black, got %v", c)
			}
		})
	}
}

func TestNormalizeLogoHost(t *testing.T) {
	for in, want := range map[string]string{
		"grafana.com":                     "grafana.com",
		"WWW.Datadoghq.com.":              "datadoghq.com",
		"jobs.example.co.uk":              "jobs.example.co.uk",
		"xn--bcher-kva.ch":                "xn--bcher-kva.ch",
		"localhost":                       "",
		"127.0.0.1":                       "",
		"[::1]":                           "",
		"evil.com/x":                      "",
		"evil.com:8080":                   "",
		"-bad.com":                        "",
		"":                                "",
		"a..com":                          "",
		"user@evil.com":                   "",
		"169.254.169.254":                 "",
		"metadata.internal":               "metadata.internal",
		strings.Repeat("a.", 130) + "com": "",
	} {
		got, ok := normalizeLogoHost(in)
		if ok != (want != "") || got != want {
			t.Errorf("normalizeLogoHost(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
}

// logoServer is a fake favicon service: good.example has an icon, gone.example
// answers 404 like the real service, broken.example answers 500.
func logoServer(t *testing.T, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	good := encodePNG(t, icon(64, white, purple))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch r.URL.Query().Get("domain") {
		case "good.example":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(good)
		case "broken.example":
			w.WriteHeader(500)
		default:
			w.WriteHeader(404)
			_, _ = w.Write(good) // the real service sends a globe with its 404
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCompanyLogo(t *testing.T) {
	var hits atomic.Int32
	upstream := logoServer(t, &hits)
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	s := &Server{logoUpstream: upstream.URL + "/?domain=", now: func() time.Time { return now }}
	get := func(domain string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.companyLogo(w, httptest.NewRequest("GET", "/api/companies/logo?domain="+domain, nil))
		return w
	}

	w := get("www.good.example")
	if w.Code != 200 || w.Header().Get("Content-Type") != "image/png" || !strings.Contains(w.Header().Get("Cache-Control"), "max-age") {
		t.Fatal(w.Code, w.Header(), w.Body.String())
	}
	if w.Header().Get("Content-Security-Policy") != "default-src 'none'" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("logo response must be locked down", w.Header())
	}
	img, err := png.Decode(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	if c := color.NRGBAModel.Convert(img.At(32, 32)).(color.NRGBA); c.A != 255 || c.R|c.G|c.B != 0 {
		t.Fatalf("centre pixel %v, want opaque black", c)
	}
	if c := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA); c.A != 0 {
		t.Fatalf("corner pixel %v, want transparent", c)
	}

	// Cached: www. and case are normalised, so this is the same host.
	if w := get("Good.Example"); w.Code != 200 || hits.Load() != 1 {
		t.Fatalf("second request: %d, upstream hits %d", w.Code, hits.Load())
	}

	// No icon: 404, cached for hours, not refetched on every page load.
	if w := get("gone.example"); w.Code != 404 || w.Header().Get("Cache-Control") != "private, max-age=3600" {
		t.Fatal("missing logo", w.Code, w.Header())
	}
	get("gone.example")
	if hits.Load() != 2 {
		t.Fatalf("missing logo refetched: %d hits", hits.Load())
	}
	now = now.Add(logoMissingTTL + time.Minute)
	get("gone.example")
	if hits.Load() != 3 {
		t.Fatalf("missing logo not retried after its TTL: %d hits", hits.Load())
	}

	// A service error is retried after minutes, not hours.
	get("broken.example")
	now = now.Add(logoErrorTTL + time.Second)
	if w := get("broken.example"); w.Code != 404 || hits.Load() != 5 {
		t.Fatalf("service error: %d, %d hits", w.Code, hits.Load())
	}

	for _, bad := range []string{"", "localhost", "10.0.0.1", "good.example%2Fpath"} {
		if w := get(bad); w.Code != 400 {
			t.Errorf("domain %q: %d, want 400", bad, w.Code)
		}
	}
}

func TestCompanyLogoConcurrentFetchesOnce(t *testing.T) {
	var hits atomic.Int32
	upstream := logoServer(t, &hits)
	s := &Server{logoUpstream: upstream.URL + "/?domain="}
	done := make(chan []byte)
	for range 8 {
		go func() {
			data, _ := s.logo(context.Background(), "good.example")
			done <- data
		}()
	}
	for range 8 {
		if data := <-done; len(data) == 0 {
			t.Fatal("a concurrent caller got no logo")
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("upstream hit %d times, want 1", hits.Load())
	}
}

func TestCompanyLogoRoute(t *testing.T) {
	db := testDB(t)
	h := newOAuthHarness(t, db)
	if w := h.do("GET", "/api/companies/logo?domain=localhost", "", nil, false); w.Code != 401 {
		t.Fatal("anonymous", w.Code)
	}
	if w := h.do("GET", "/api/companies/logo?domain=localhost", "", nil, true); w.Code != 400 {
		t.Fatal("signed in, bad domain", w.Code, w.Body.String())
	}
	if _, err := db.UpdateAppSettings(context.Background(), modules.Patch{Modules: map[string]bool{"companies": false}}); err != nil {
		t.Fatal(err)
	}
	if w := h.do("GET", "/api/companies/logo?domain=localhost", "", nil, true); w.Code != 404 {
		t.Fatal("companies off", w.Code, w.Body.String())
	}
}
