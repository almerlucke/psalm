package drums

import (
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/sndfile"
)

func NewDrums(soundBank sndfile.SoundBank[float32], numVoices int) *poly.Poly {
	var initSound sndfile.SoundFiler[float32]

	for _, v := range soundBank {
		initSound = v
		break
	}

	voices := make([]poly.Voice, numVoices)
	for i := range numVoices {
		p := player.New(player.Config{
			SndBank:   soundBank,
			SndFile:   initSound,
			EndOffset: 1.0,
		})
		voices[i] = p
	}

	return poly.New(initSound.NumChannels(), voices)
}
