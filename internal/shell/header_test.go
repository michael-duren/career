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
		marked := regexp.MustCompile(`<li data-module-id="([^"]+)"[^>]*><a[^>]*aria-current="page"`).FindAllStringSubmatch(html, -1)
		if len(marked) != 1 || marked[0][1] != current {
			t.Errorf("current %s: aria-current on %v", current, marked)
		}
	}
	if regexp.MustCompile(`<li[^>]*><a[^>]*aria-current`).MatchString(render(t, "settings")) {
		t.Error("a page outside the menu marks a menu link current")
	}
}

var tabPattern = regexp.MustCompile(`<a class="app-tab" href="([^"]+)" data-module-id="([^"]+)" data-routes="[^"]*"( hidden)?( aria-current="page")?>`)

func TestTabBarHasHomeEveryModuleAndMore(t *testing.T) {
	html := render(t, "leetgrinder")
	got := []string{}
	for _, m := range tabPattern.FindAllStringSubmatch(html, -1) {
		got = append(got, m[2]+" "+m[1]+m[3])
	}
	want := []string{"home /"}
	for i, m := range modules.Registry {
		hidden := ""
		if i >= TabBarSize {
			hidden = " hidden"
		}
		want = append(want, m.ID+" "+m.Pages[0]+hidden)
	}
	want = append(want, "more /more")
	if !slices.Equal(got, want) {
		t.Fatalf("tabs %v, want %v", got, want)
	}
	for _, tab := range Tabs() {
		if len(tab.Icon) == 0 {
			t.Errorf("tab %s has no icon in tab-icons.json", tab.ID)
		}
	}
	if !strings.Contains(html, `data-app-search-toggle`) || !strings.Contains(html, `id="site-search-box"`) {
		t.Error("phone search button missing")
	}
}

func TestTabBarMarksTheCurrentTab(t *testing.T) {
	for current, want := range map[string]string{"home": "home", "leetgrinder": "leetgrinder", "timeline": "timeline", "notes": "more", "settings": "more", "": ""} {
		marked := []string{}
		for _, m := range tabPattern.FindAllStringSubmatch(render(t, current), -1) {
			if m[4] != "" {
				marked = append(marked, m[2])
			}
		}
		if (want == "" && len(marked) != 0) || (want != "" && !slices.Equal(marked, []string{want})) {
			t.Errorf("current %q: tab marked %v, want %q", current, marked, want)
		}
	}
}

func TestRegistryHasHeaderLabels(t *testing.T) {
	seen := map[string]bool{}
	for _, m := range modules.Registry {
		if m.Short == "" || m.Short != strings.ToUpper(m.Short) || seen[m.Short] {
			t.Errorf("%s: short label %q must be non-empty, uppercase and unique", m.ID, m.Short)
		}
		seen[m.Short] = true
		if len(m.Pages) == 0 {
			t.Errorf("%s has no main route", m.ID)
		}
	}
}
