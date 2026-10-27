package dx

// oversampled are the presets that fold back a lot at the pitches of a melody (a scan of the alias of every preset at C5 and C6, a sound at 44.1 kHz against one at 176.4
// kHz brought down: levels/aliasscan_test.go), and how they are played: 1 twice and 2 four times a sample (Patch.Oversample). The ones in the first group had their alias
// at -4 to -20 dB, the second at -30 to -40 dB; at four times the first lose 15 to 35 dB of it. The noises (windGust, hatFM, woodblock) are not here: there is no
// telling an alias from another noise
var oversampled = map[string]int{
	"sawLead": 2, "overdriveGuitar": 2, "sitar": 2, "metallic": 2, "acidBass": 2, "lead": 2, "tubularBells": 2, "pulseClav": 2, "logSawBrass": 2, "hollowReed": 2,
	"saxophone": 2, "woodClarinet": 2, "pwmLead": 2, "syncLead": 2, "reedOrgan": 2,

	"sweepStrings": 1, "riserSweep": 1, "filterPluck": 1, "wheelPad": 1, "velocityFM": 1, "halfSineOrgan": 1, "rubberBass": 1,
}

// louder is what the level of a preset is raised by (calibrated again, at -12 LUFS, at its oversampling): the alias of a hard patch was part of its level
var louder = map[string]float64{"sawLead": 2.2, "lead": 1.3, "sitar": 1.3, "overdriveGuitar": 1.8}

func init() {
	for name, over := range oversampled {
		make := Presets[name]
		if make == nil {
			panic("dx: oversampled preset " + name + " does not exist")
		}
		Presets[name] = func() Patch {
			p := make()
			p.Oversample = over
			if g, ok := louder[name]; ok {
				p.Level *= g
			}
			return p
		}
	}
}
