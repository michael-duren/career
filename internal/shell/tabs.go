package shell

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/michael-duren/career-strategy/internal/modules"
)

// tabIconsJSON holds the phone tab bar's 22px line icons as SVG path data per
// tab id. src/components/AppHeader.astro imports the same file, so both
// headers draw identical icons.
//
//go:embed tab-icons.json
var tabIconsJSON []byte

var tabIcons = func() map[string][]string {
	icons := map[string][]string{}
	if err := json.Unmarshal(tabIconsJSON, &icons); err != nil {
		panic(fmt.Sprintf("shell: parse tab-icons.json: %v", err))
	}
	return icons
}()

// TabBarSize is how many module tabs sit between Home and More.
const TabBarSize = 3

// Tab is one phone tab bar slot: Home, a module, or More.
type Tab struct {
	Link
	Icon []string
	// Hidden is the server's first guess (the first TabBarSize modules in
	// registry order show); shell.js applies the user's order and switches.
	Hidden bool
}

// Tabs returns Home, every module and More. Modules past the first
// TabBarSize start hidden.
func Tabs() []Tab {
	tabs := []Tab{}
	for i, link := range Links() {
		tabs = append(tabs, Tab{Link: link, Icon: tabIcons[link.ID], Hidden: i > TabBarSize})
	}
	return append(tabs, Tab{Link: Link{ID: "more", Short: "MORE", Href: "/more", Routes: "/more /settings"}, Icon: tabIcons["more"]})
}

// currentTab is the tab marked current before shell.js runs: Home or a
// module in the first slots, else More (it lists every other module and
// Settings).
func currentTab(current string) string {
	for _, tab := range Tabs() {
		if tab.ID == current && !tab.Hidden {
			return current
		}
	}
	if current == "settings" || current == "more" {
		return "more"
	}
	if _, ok := modules.Lookup(current); ok {
		return "more"
	}
	return ""
}
