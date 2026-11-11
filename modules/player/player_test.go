package player

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/sndfile"
)

const crash = "../../assets/sounds/drums/hihat/Cymatics - Humble Crash 1.wav"

// play renders buffers and returns the peak of each
func play(p *Player, buffers int) []float64 {
	var peaks []float64
	for range buffers {
		p.Synthesize()
		peak := 0.0
		for _, ch := range p.Output(0).Channels {
			for _, x := range ch {
				peak = max(peak, math.Abs(float64(x)))
			}
		}
		peaks = append(peaks, peak)
	}
	return peaks
}

// A note off fades a one shot out and stops it when the player chokes
func TestFadeOnNoteOff(t *testing.T) {
	psalm.New(nil)
	sf := sndfile.MustSoundFile[float32](crash)
	for _, fade := range []float64{0, 0.01} {
		p := New(Config{SndFile: sf, EndOffset: 1, FadeOnNoteOff: fade})
		p.NoteOn(1, psalm.Message{}, psalm.CtrlTick{}, p.Config())
		if play(p, 4)[3] < 0.01 {
			t.Fatal("the crash should be sounding")
		}
		p.NoteOff()
		fadeBuffers := int(math.Ceil(fade*p.Config().SampleRate/float64(p.Config().BufferSize))) + 1
		peaks := play(p, fadeBuffers+2)
		after := peaks[len(peaks)-1]
		if fade > 0 && (after != 0 || !p.done) {
			t.Fatalf("a choked sound should be silent after its fade, peak %v", after)
		}
		if fade == 0 && after < 0.01 {
			t.Fatal("without a fade a one shot should play on after a note off")
		}
	}
}

// a message can give the sound file itself, not one of the bank by name
func TestSoundFileMessage(t *testing.T) {
	psalm.New(nil)
	p := New(Config{EndOffset: 1})
	p.NoteOn(1, psalm.Message{}, psalm.CtrlTick{}, p.Config())
	if play(p, 2)[1] != 0 {
		t.Fatal("no sound without a file")
	}
	p = New(Config{EndOffset: 1})
	p.NoteOn(1, psalm.Message{"soundFile": sndfile.SoundFiler[float32](sndfile.MustSoundFile[float32](crash))}, psalm.CtrlTick{}, p.Config())
	if play(p, 4)[3] < 0.01 {
		t.Fatal("the file of the message should be sounding")
	}
}
