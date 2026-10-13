package panel

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
)

// Theme is a look of the page: values for its CSS variables, in light and in dark, and CSS of its
// own. The variables (written without the leading "--"):
//
//	bg, card, ink, muted, line      the page, the cards of the sections, the text, quiet text, borders
//	track, accent, accent-ink       the travel of faders and knobs, what is on, the text on it
//	meter-low, meter-mid, meter-high
//	shadow                          of the cards (a CSS box-shadow)
//	font                            the font (a CSS font-family)
//	radius                          the corners of the cards (buttons have two thirds of it)
//	fader-height                    faders and meters
//
// A theme with Dark looks like Light in light mode and like Dark in dark mode (the system's, or
// the one chosen on the page); without Dark it looks like Light in both. Variables a theme leaves
// out keep the page's own. CSS comes after the page's styles: it can change anything, the classes
// of the page are those of its controls (.fader, .knob, .meter, section, button ...), and every
// control is in an element with data-kind and data-name
type Theme struct {
	Name  string
	Light map[string]string
	Dark  map[string]string
	CSS   string
}

// Themes are the themes every page can choose from (the page has a choice of them, remembered in
// the browser); a panel's own theme (SetTheme) is added to them
var Themes = map[string]Theme{
	"default": {Name: "default"},
	"midnight": {Name: "midnight", Light: map[string]string{
		"bg": "#0b1020", "card": "#131a2e", "ink": "#e3e8f7", "muted": "#8a93b3", "line": "#26304d",
		"track": "#1e2742", "accent": "#7aa2ff", "accent-ink": "#0b1020", "shadow": "0 2px 12px rgba(0,0,0,.4)",
	}},
	"paper": {Name: "paper", Light: map[string]string{
		"bg": "#f3ede1", "card": "#fbf7ef", "ink": "#2b2620", "muted": "#7d7364", "line": "#ddd2bf",
		"track": "#e8dfcd", "accent": "#b5523b", "accent-ink": "#fbf7ef", "meter-low": "#5e8c4a",
		"font": `Georgia, "Iowan Old Style", serif`, "shadow": "none", "radius": "4px",
	}},
	"amber": {Name: "amber", Light: map[string]string{
		"bg": "#16120c", "card": "#211b12", "ink": "#f2d9a8", "muted": "#a08a62", "line": "#3a2f1f",
		"track": "#2e261a", "accent": "#ffb000", "accent-ink": "#16120c", "meter-low": "#ffb000",
		"meter-mid": "#ff8a00", "meter-high": "#ff4d1a", "shadow": "none",
		"font": `"JetBrains Mono", "SFMono-Regular", Menlo, Consolas, monospace`,
	}},
	"contrast": {Name: "contrast", Light: map[string]string{
		"bg": "#ffffff", "card": "#ffffff", "ink": "#000000", "muted": "#333333", "line": "#000000",
		"track": "#d0d0d0", "accent": "#0040ff", "accent-ink": "#ffffff", "shadow": "none",
	}, Dark: map[string]string{
		"bg": "#000000", "card": "#000000", "ink": "#ffffff", "muted": "#dddddd", "line": "#ffffff",
		"track": "#444444", "accent": "#ffe600", "accent-ink": "#000000", "shadow": "none",
	}},
	"neon": {Name: "neon", Light: map[string]string{
		"bg": "#0d0221", "card": "#1a0b3a", "ink": "#f5e9ff", "muted": "#a58bd1", "line": "#3b1d73",
		"track": "#2a1257", "accent": "#ff2bd6", "accent-ink": "#0d0221", "meter-low": "#21f0e1",
		"meter-mid": "#f9f871", "meter-high": "#ff2b6d", "shadow": "0 0 18px rgba(255,43,214,.15)", "radius": "16px",
	}},
}

// SetTheme gives the panel a theme: the one its page shows until another is chosen there. It is
// added to the themes of the page under its name
func (pn *Panel) SetTheme(t Theme) *Panel {
	if t.Name == "" {
		t.Name = "custom"
	}
	pn.mu.Lock()
	defer pn.mu.Unlock()
	pn.theme = &t
	return pn
}

// themes returns the themes of the page by name, and the one it starts with
func (pn *Panel) themes() (map[string]Theme, string) {
	all := maps.Clone(Themes)
	pn.mu.Lock()
	defer pn.mu.Unlock()
	if pn.theme == nil {
		return all, "default"
	}
	all[pn.theme.Name] = *pn.theme
	return all, pn.theme.Name
}

// themeNames returns the names of the themes, the default first
func themeNames(all map[string]Theme) []string {
	names := slices.Sorted(maps.Keys(all))
	if i := slices.Index(names, "default"); i > 0 {
		names = append([]string{"default"}, slices.Delete(names, i, i+1)...)
	}
	return names
}

// stylesheet returns the CSS of the theme
func (t Theme) stylesheet() string {
	var b strings.Builder
	vars := func(m map[string]string) {
		for _, k := range slices.Sorted(maps.Keys(m)) {
			fmt.Fprintf(&b, "\t--%s: %s;\n", strings.TrimPrefix(k, "--"), m[k])
		}
	}
	// as specific as the page's own dark rules, so the light values hold in dark mode too
	b.WriteString(`:root, :root:not([data-mode="light"]), :root[data-mode="dark"] {` + "\n")
	vars(t.Light)
	b.WriteString("}\n")
	if len(t.Dark) > 0 {
		b.WriteString("@media (prefers-color-scheme: dark) {\n:root:not([data-mode=\"light\"]) {\n")
		vars(t.Dark)
		b.WriteString("}\n}\n:root[data-mode=\"dark\"] {\n")
		vars(t.Dark)
		b.WriteString("}\n")
	}
	b.WriteString(t.CSS)
	return b.String()
}

// serveTheme serves the stylesheet of a theme (?name=, the panel's own without)
func (pn *Panel) serveTheme(w http.ResponseWriter, r *http.Request) {
	all, def := pn.themes()
	t, ok := all[r.URL.Query().Get("name")]
	if !ok {
		t = all[def]
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(t.stylesheet()))
}
