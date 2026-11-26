package shell

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/michael-duren/career-strategy/internal/modules"
)

func render(t *testing.T, current string) string {
	t.Helper()
	var b strings.Builder
	if err := Header(current).Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestHeaderListsHomeAndEveryModule(t *testing.T) {
	html := render(t, "leetgrinder")
	got := []string{}
	for _, m := range regexp.MustCompile(`<li data-module-id="([^"]+)" data-routes="([^"]*)"><a class="app-link" href="([^"]+)"[^>]*>([^<]+)</a>`).FindAllStringSubmatch(html, -1) {
		got = append(got, m[1]+" "+m[3]+" "+m[4])
	}
	want := []string{"home / HOME"}
	for _, m := range modules.Registry {
		want = append(want, m.ID+" "+m.Pages[0]+" "+m.Short)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("links %v, want %v", got, want)
	}
	for _, s := range []string{`href="/settings/modules" aria-label="Settings"`, `data-app-clock`, `id="site-search"`, `src="/ui/shell.js"`, `src="/ui/site-search.js"`, `data-routes="/journal /logs /personal-journal"`, `data-app-signout`} {
		if !strings.Contains(html, s) {
			t.Errorf("header missing %s", s)
		}
	}
}

func TestHeaderMarksOnlyTheCurrentLink(t *testing.T) {
	for _, current := range []string{"home", "leetgrinder", "notes"} {
		html := render(t, current)
		marked := regexp.MustCompile(`data-module-id="([^"]+)"[^>]*><a[^>]*aria-current="page"`).FindAllStringSubmatch(html, -1)
		if len(marked) != 1 || marked[0][1] != current {
			t.Errorf("current %s: aria-current on %v", current, marked)
		}
	}
	if strings.Contains(render(t, "settings"), `aria-current`) {
		t.Error("a page outside the menu marks a link current")
	}
}
