package keyboard

import (
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
)

// a file plays two keyboards, one for each channel, at the frames of its notes
func TestPlayerDrivesKeyboards(t *testing.T) {
	f := &midi.File{Division: 480}
	var tr midi.Track
	tr.AddMeta(0, midi.MetaTempo, midi.TempoData(60)) // a second a quarter
	tr.Add(240, midi.NoteOnMsg(1, 60, 100))           // at 0.5 s
	tr.Add(480, midi.NoteOffMsg(1, 60, 0))
	tr.Add(720, midi.NoteOnMsg(10, 36, 127)) // at 1.5 s, a drum
	tr.Add(730, midi.NoteOffMsg(10, 36, 0))
	f.Tracks = []midi.Track{tr}

	p := psalm.New(nil)
	sr := p.Config().SampleRate
	player := midi.NewPlayer(f)
	ins := midi.Tee(player, 2)
	keys, drums := &recorder{}, &recorder{}
	New(ins[0], Config{Channel: 1}, keys).AddTo(p)
	New(ins[1], Config{Channel: 10, Hits: true}, drums).AddTo(p)
	player.Play()
	for range int(2*sr)/p.Config().BufferSize + 1 {
		run(p)
	}
	// the frame a message plays on (the sample timestamps of psalm count from 1)
	at := func(m psalm.Message) int64 { return m.MustTickVal("tick").SampleTimestamp - 1 }
	if len(keys.msgs) < 2 || keys.msgs[0]["noteOn"] != "midi-1-60" || keys.msgs[1]["noteOff"] != "midi-1-60" {
		t.Fatalf("channel 1 got %v", keys.msgs)
	}
	if got, want := at(keys.msgs[0]), int64(0.5*sr); got != want {
		t.Errorf("the note on at sample %d, want %d", got, want)
	}
	if got, want := at(keys.msgs[1]), int64(1*sr); got != want {
		t.Errorf("the note off at sample %d, want %d", got, want)
	}
	// the drum is a hit: a note with a duration, no note off
	var hits []psalm.Message
	for _, m := range drums.msgs {
		if m["command"] == "trigger" {
			hits = append(hits, m)
		}
	}
	if len(hits) != 1 {
		t.Fatalf("channel 10 triggers %v", hits)
	}
	if _, ok := hits[0]["duration"]; !ok || at(hits[0]) != int64(1.5*sr) {
		t.Errorf("the drum %v", hits[0])
	}
}
