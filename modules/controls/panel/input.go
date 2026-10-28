package panel

import (
	"slices"

	"github.com/almerlucke/psalm/audio"
)

// InputSelect adds a choice of the input device the audio engine listens to (a headset, the microphone of the computer): the devices of the engine when it can choose
// (audio.InputSelector: Psalm.PatchInput(0) is one with the portaudio engine), nil and nothing added when it can't. The stream is opened again when one is chosen, a moment of
// silence in what is heard of the input
func (pn *Panel) InputSelect(sel any) *Param {
	s, ok := sel.(audio.InputSelector)
	if !ok {
		return nil
	}
	devs := s.InputDevices()
	if len(devs) < 2 {
		return nil
	}
	return pn.Select("Input", devs, func(i int) {
		if i >= 0 && i < len(devs) && devs[i] != s.InputDevice() {
			go func() { _ = s.SetInputDevice(devs[i]) }() // the engine waits for the audio callback to end: not from inside it
		}
	}, Default(float64(max(0, slices.Index(devs, s.InputDevice())))), NoDefault())
}
