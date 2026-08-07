package poly

import (
	"testing"

	"github.com/almerlucke/psalm"
)

type MockVoice struct {
	*psalm.Module
	active bool
	id     string
}

func NewMockVoice() *MockVoice {
	v := &MockVoice{
		Module: psalm.NewModule([]int{psalm.Mono}, []int{psalm.Mono}),
		active: false,
	}
	v.SetSelf(v)
	return v
}

func (v *MockVoice) NoteOn(amplitude float64, message psalm.Message, config *psalm.Config) {
	v.active = true
}
func (v *MockVoice) NoteOff() { v.active = false }
func (v *MockVoice) Note(duration float64, amplitude float64, message psalm.Message, config *psalm.Config) {
	v.active = true
}
func (v *MockVoice) Clear()                        {}
func (v *MockVoice) IsActive() bool                { return v.active }
func (v *MockVoice) ID() string                    { return v.id }
func (v *MockVoice) SetID(id string) psalm.Modular { v.id = id; return v }

func TestPolyPanic(t *testing.T) {
	voices := []Voice{NewMockVoice()}
	p := New(1, voices)

	// Trigger a note so the voice becomes active
	p.ReceiveMessage(psalm.Message{
		"command":   "trigger",
		"noteOn":    "note1",
		"amplitude": 0.5,
		"message":   psalm.Message{},
	})

	// Now trigger another note to cause stealing (since there's only 1 voice)
	// But use malformed data types that might cause panics in activateStolenVoiceInfo
	p.ReceiveMessage(psalm.Message{
		"command":   "trigger",
		"noteOn":    123,             // Should be string
		"amplitude": "0.5",           // Should be float64
		"message":   "not a message", // Should be Message
	})

	// Synthesize should trigger activateStolenVoiceInfo
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Recovered from panic: %v", r)
		}
	}()

	p.Synthesize()
}
