package patcher

import (
	"regexp"
	"strings"
	"testing"
)

// the log of a patch: a line from a module starts with the time and its name, the newest are kept, and it can be cleared
func TestJournal(t *testing.T) {
	j := NewJournal()
	j.Add("script m3", "hello\n  there")
	if !regexp.MustCompile(`^\d\d:\d\d:\d\d\.\d{3}  script m3  hello there$`).MatchString(j.Text()) {
		t.Errorf("line %q", j.Text())
	}
	for range journalLines + 50 {
		j.Add("a", "x")
	}
	if n := len(strings.Split(j.Text(), "\n")); n != journalShown {
		t.Errorf("%d lines shown", n)
	}
	if j.Len() != journalLines+51 {
		t.Errorf("%d lines written", j.Len())
	}
	j.Clear()
	if j.Text() != "" || j.Len() != 0 {
		t.Error("not cleared")
	}
}
