package drums

import (
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/sndfile"
)

// chokeFade is how many seconds a choked sound takes to fade out (short, without a click)
const chokeFade = 0.008

func NewDrums(soundBank sndfile.SoundBank[float32], numVoices int, numChannels int) *poly.Poly[*player.Player] {
	voices := make([]*player.Player, numVoices)
	for i := range numVoices {
		voices[i] = player.New(player.Config{
			SndBank:     soundBank,
			NumChannels: numChannels,
			EndOffset:   1.0,
			// a note off chokes the sound: an open hihat cut short by the closed one
			FadeOnNoteOff: chokeFade,
		})
	}

	return poly.New[*player.Player](numChannels, voices)
}
