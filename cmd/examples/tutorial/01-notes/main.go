// Step 1: notes and a sequencer.
//
// Everything in an arrangement starts as notes. A score.Note has pitches (MIDI numbers: 60 is
// middle C, one number per semitone), a length in beats and optionally a velocity and an amplitude.
// A sequencer plays a list of notes on a synth, in time with the tempo of the patch.
//
// This step plays two things at once on an electric piano: chords written note by note in Go, and
// a melody written in score notation, a compact text form:
//
//	C5:1      the note C5 (72) lasting 1 beat
//	E5        no length: as long as the note before
//	G5:2@.8   2 beats, velocity 0.8 (how hard it is played: louder and brighter)
//	r:1       a rest of 1 beat
//	[C4 E4]:2 a chord
//	C5:1_3    lasting 1 beat, but sounding for 3 (it rings on under the next notes)
//	|         a bar line: ignored, only for reading
//
// Run it:
//
//	go run ./cmd/examples/tutorial/01-notes
//	go run ./cmd/examples/tutorial/01-notes -render notes.wav
//
// This step writes out everything a program needs around the music. The next steps put that part
// in the tutorial package (tutorial.New and tutorial.Play), so they show only what they teach.
package main

import (
	"flag"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/tools/render"
)

func main() {
	renderTo := flag.String("render", "", "render to this WAV file instead of playing")
	flag.Parse()

	// the patch: the tempo (beats per minute), the resolution of a beat in ticks (PPQ), the sample
	// rate and the buffer size; no audio engine when rendering to a file
	psalm.PushConfig(psalm.Config{BPM: 96, PPQ: 240, SampleRate: 44100, BufferSize: 512})
	var p *psalm.Psalm
	if *renderTo != "" {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	// a synth with 8 voices (8 notes can sound at once). Every preset of every synth is calibrated
	// to the same loudness at amplitude 1, so the amplitudes below set the balance
	keys := fm.NewPoly(fm.EPianoPreset(), 8).AddTo(p)

	// --- notes written in Go: a chord per bar of 4 beats ---

	chords := []score.Note{
		score.Chord([]float64{48, 60, 64, 67}, 4), // C major: C3 C4 E4 G4
		score.Chord([]float64{45, 60, 64, 69}, 4), // A minor
		score.Chord([]float64{41, 60, 65, 69}, 4), // F major
		score.Chord([]float64{43, 59, 62, 67}, 4), // G major
	}

	// --- the same kind of notes from score notation: the melody over the chords ---

	melody := score.MustParse(`
		E5:1 G5 C6:2@.8 | E5:1 A5 C6:1.5 B5:.5 |
		A5:1 F5 C5:2    | D5:1 G5 B4:1_3 r:1`)

	// a sequencer plays the notes on the synth. Amp scales the amplitude of every note, Legato is
	// the part of its length a note sounds (0.9: a short gap between notes), Velocity is the
	// velocity of notes without one, Loop starts again after the last note
	chordPart := score.NewSequencer(chords, score.Config{Amp: 0.18, Legato: 0.95, Velocity: 0.5, Loop: true}).AddTo(p)
	melodyPart := score.NewSequencer(melody, score.Config{Amp: 0.35, Legato: 0.9, Velocity: 0.65, Loop: true}).AddTo(p)

	// connect the sequencers to the synth: they send it their notes as messages
	keys.ValueIn(chordPart)
	keys.ValueIn(melodyPart)

	// the synth to the speakers
	p.Out(keys)

	if *renderTo != "" {
		// 16 beats at 96 BPM, twice, and 3 seconds for the last notes to ring out
		_, err := render.ToFile(p, *renderTo, render.Options{Seconds: 2 * 16 * 60 / 96.0, Tail: 3})
		if err != nil {
			panic(err)
		}
		return
	}
	_ = p.Listen() // play until Ctrl-C
}
