package main

import (
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/tools/render"
)

// the whole song is written, every part fits its chords, and its first seconds render
func TestExodusPlays(t *testing.T) {
	t.Chdir("../../..") // the sounds are found from the root of the repository
	psalm.PushConfig(psalm.Config{BPM: bpm, PPQ: 240, SampleRate: 44100, BufferSize: 1024})
	defer psalm.PopConfig()
	psalm.Seed(7)
	p := psalm.New(nil)
	defer p.Close()
	o := newOrchestra(p)
	if n := writeParts(p, o); n != 0 {
		t.Fatalf("%d notes clash with the chords", n)
	}
	arrangement := o.arrangement()
	arrangement.AddTo(p)
	p.Out(o.mix.Master())
	if d := arrangement.Duration(); d < 300 || d > 480 {
		t.Errorf("the song lasts %.0f s", d)
	}
	stats, err := render.Stems(p, t.TempDir()+"/exodus.wav", render.Options{Seconds: 6, Tail: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(stats)
}
