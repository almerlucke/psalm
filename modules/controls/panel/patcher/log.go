package patcher

import (
	"fmt"
	"strings"
	"sync"

	"github.com/almerlucke/psalm"
)

// logLines is how many lines a log module keeps
const logLines = 12

// Log is a module that shows what is sent to it: values and triggers, the last lines of them on
// the module and in its side panel. What comes in goes out the same way, so it can sit in a cable
type Log struct {
	*psalm.Module
	mu    sync.Mutex
	label string
	lines []string
	n     int
}

// NewLog returns a log
func NewLog() *Log {
	l := &Log{Module: psalm.NewModule(nil, nil)}
	l.SetSelf(l)
	return l
}

// SetLabel sets what every line starts with ("" nothing)
func (l *Log) SetLabel(label string) {
	l.mu.Lock()
	l.label = strings.TrimSpace(label)
	l.mu.Unlock()
}

// Receive logs a value (input 0) or a trigger (input 1), and sends it on from the same number output
func (l *Log) Receive(v any, index int) {
	var text string
	switch x := v.(type) {
	case Event:
		text = fmt.Sprintf("trigger step %d", x.Step)
	case []float64:
		parts := make([]string, len(x))
		for i, f := range x {
			parts[i] = fmt.Sprintf("%.4g", f)
		}
		text = "[" + strings.Join(parts, " ") + "]"
	case float64, float32, int, int64, int32, uint, uint8, bool:
		text = fmt.Sprintf("%.4g", psalm.FltVal[float64](v))
	default:
		if psalm.IsBang(v) {
			text = "bang"
		} else {
			text = fmt.Sprint(v)
		}
	}
	l.mu.Lock()
	l.n++
	line := fmt.Sprintf("%d  %s", l.n, text)
	if l.label != "" {
		line = l.label + "  " + line
	}
	l.lines = append(l.lines, line)
	if len(l.lines) > logLines {
		l.lines = l.lines[len(l.lines)-logLines:]
	}
	l.mu.Unlock()
	l.Send(v, index)
}

// Status is the last lines (Made.Log), the newest below
func (l *Log) Status() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.lines, "\n")
}

// Clear forgets the lines
func (l *Log) Clear() {
	l.mu.Lock()
	l.lines, l.n = nil, 0
	l.mu.Unlock()
}

// LogKind is the log module: a value and a trigger in, the same out, and what came in shown
func LogKind() Kind {
	return Kind{Name: "log", Group: "Tools",
		Ins:  []Port{{Name: "value", Type: Value}, {Name: "trigger", Type: Trigger, Index: 1}},
		Outs: []Port{{Name: "value", Type: Value}, {Name: "trigger", Type: Trigger, Index: 1}},
		Make: func(string) (*Made, error) {
			l := NewLog()
			return &Made{Module: l, Log: l.Status,
				Texts: []Text{{Key: "label", Label: "label (starts every line)", Value: "", Help: LogHelp}},
				SetText: func(_, v string) error {
					l.SetLabel(v)
					return nil
				}}, nil
		}}
}

// LogHelp is what to know about the log
const LogHelp = "shows what comes in: a value (a number, a chord in brackets) on the value input, a moment on the trigger input\n" +
	"the last 12 lines in the side panel, the newest below; what comes in goes out the same way\n" +
	"a label starts every line, to tell logs apart"
