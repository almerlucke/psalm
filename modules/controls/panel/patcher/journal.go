package patcher

import (
	"strings"
	"sync"
	"time"
)

// journalLines is how many lines the log of a patch keeps
const journalLines = 1000

// journalShown is how many of them the page is sent (the newest)
const journalShown = 300

// Journal is the log of a patch: a line from any module (a script's log), each starting with the time and the name of the module that wrote it,
// all in one place, opened from the bar at the top of the page. Safe to use from any goroutine
type Journal struct {
	mu    sync.Mutex
	lines []string
	n     int // lines written so far, also the ones forgotten
}

// NewJournal returns an empty log
func NewJournal() *Journal { return &Journal{} }

// Add writes a line: the time, the name of the module and the text (a line of its own: breaks in it become spaces)
func (j *Journal) Add(source, text string) {
	text = strings.Join(strings.Fields(text), " ")
	line := time.Now().Format("15:04:05.000") + "  " + source + "  " + text
	j.mu.Lock()
	j.n++
	j.lines = append(j.lines, line)
	if len(j.lines) > journalLines {
		j.lines = append([]string(nil), j.lines[len(j.lines)-journalLines:]...)
	}
	j.mu.Unlock()
}

// Text is the newest lines, the newest below
func (j *Journal) Text() string {
	j.mu.Lock()
	defer j.mu.Unlock()
	lines := j.lines
	if len(lines) > journalShown {
		lines = lines[len(lines)-journalShown:]
	}
	return strings.Join(lines, "\n")
}

// Len is how many lines were written so far
func (j *Journal) Len() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.n
}

// Clear forgets the lines
func (j *Journal) Clear() {
	j.mu.Lock()
	j.lines, j.n = nil, 0
	j.mu.Unlock()
}
