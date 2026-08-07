package drums

import (
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/sndfile"
)

func NewDrums(soundBank sndfile.SoundBank, numVoices int) *poly.Poly {
	var initSound sndfile.SoundFiler

	for _, v := range soundBank {
		initSound = v
		break
	}

	voices := make([]poly.Voice, numVoices)
	for i := range numVoices {
		p := player.New(player.Config{
			SndBank:   soundBank,
			SndFile:   initSound,
			EndOffset: 0.99,
		})
		voices[i] = p
	}

	return poly.New(initSound.NumChannels(), voices)
}
