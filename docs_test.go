package psalm_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// every relative link of the docs goes to a file that exists (and to a heading that exists in it)
func TestDocLinks(t *testing.T) {
	link := regexp.MustCompile(`\]\(([^)#\s:]+\.md)(#[^)]*)?\)`)
	heading := regexp.MustCompile(`(?m)^#+ +(.+)$`)
	slug := func(h string) string {
		h = strings.ToLower(strings.ReplaceAll(h, "`", ""))
		h = regexp.MustCompile(`[^a-z0-9 _-]`).ReplaceAllString(h, "")
		return strings.ReplaceAll(strings.TrimSpace(h), " ", "-")
	}
	files, _ := filepath.Glob("docs/*.md")
	more, _ := filepath.Glob("docs/modules/*.md")
	for _, f := range append(files, more...) {
		text, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range link.FindAllStringSubmatch(string(text), -1) {
			target := filepath.Join(filepath.Dir(f), m[1])
			dest, err := os.ReadFile(target)
			if err != nil {
				t.Errorf("%s: link to %s: %v", f, m[1], err)
				continue
			}
			if m[2] == "" {
				continue
			}
			found := false
			for _, h := range heading.FindAllStringSubmatch(string(dest), -1) {
				if slug(h[1]) == m[2][1:] {
					found = true
				}
			}
			if !found {
				t.Errorf("%s: link to %s%s: no such heading", f, m[1], m[2])
			}
		}
	}
}
