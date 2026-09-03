package drums

import (
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/sndfile"
)

func NewDrums(soundBank sndfile.SoundBank[float32], numVoices int, numChannels int) *poly.Poly {
	voices := make([]poly.Voice, numVoices)
	for i := range numVoices {
		voices[i] = player.New(player.Config{
			SndBank:     soundBank,
			NumChannels: numChannels,
			EndOffset:   1.0,
		})
	}

	return poly.New(numChannels, voices)
}
