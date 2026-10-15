// Code generated from the presets of the FM synth that dx took over; DO NOT EDIT.

package dx

import "github.com/almerlucke/genny/float/envelopes/adsr"

func init() {
	for name, preset := range fmPresets {
		Presets[name] = preset
	}
}

// fmPresets are the presets of the FM synth, by name
var fmPresets = map[string]func() Patch{
	"accordion":       AccordionPreset,
	"bass":            BassPreset,
	"bell":            BellPreset,
	"brass":           BrassPreset,
	"celesta":         CelestaPreset,
	"churchOrgan":     ChurchOrganPreset,
	"clarinet":        ClarinetPreset,
	"clavinet":        ClavinetPreset,
	"ePiano":          EPianoPreset,
	"flute":           FlutePreset,
	"frenchHorn":      FrenchHornPreset,
	"fretlessBass":    FretlessBassPreset,
	"glockenspiel":    GlockenspielPreset,
	"grandPiano":      GrandPianoPreset,
	"harp":            HarpPreset,
	"harpsichord":     HarpsichordPreset,
	"honkyTonk":       HonkyTonkPreset,
	"kalimba":         KalimbaPreset,
	"koto":            KotoPreset,
	"lead":            LeadPreset,
	"mallet":          MalletPreset,
	"metallic":        MetallicPreset,
	"musicBox":        MusicBoxPreset,
	"nylonGuitar":     NylonGuitarPreset,
	"oboe":            OboePreset,
	"organ":           OrganPreset,
	"overdriveGuitar": OverdriveGuitarPreset,
	"pad":             PadPreset,
	"panFlute":        PanFlutePreset,
	"percussiveOrgan": PercussiveOrganPreset,
	"pizzicato":       PizzicatoPreset,
	"pluck":           PluckPreset,
	"sawLead":         SawLeadPreset,
	"saxophone":       SaxophonePreset,
	"sitar":           SitarPreset,
	"slapBass":        SlapBassPreset,
	"squareLead":      SquareLeadPreset,
	"steelDrum":       SteelDrumPreset,
	"steelGuitar":     SteelGuitarPreset,
	"strings":         StringsPreset,
	"sweepPad":        SweepPadPreset,
	"taiko":           TaikoPreset,
	"timpani":         TimpaniPreset,
	"trumpet":         TrumpetPreset,
	"tubularBells":    TubularBellsPreset,
	"uprightBass":     UprightBassPreset,
	"warmPad":         WarmPadPreset,
	"woodblock":       WoodblockPreset,
	"xylophone":       XylophonePreset,
}

// AccordionPreset: two slightly detuned buzzing reeds
func AccordionPreset() Patch {
	return fmPatch("accordion", 4, 0, 0.5599, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0, Env: adsr.NewSetting(1, 30, 0.9, 100, 1, 120)},
		{Ratio: 1, Level: 0.45, Velocity: 0.3, Env: adsr.NewSetting(1, 30, 0.9, 100, 1, 120)},
		{Ratio: 1, Level: 0.8, Velocity: 0, Detune: 8, Env: adsr.NewSetting(1, 30, 0.9, 100, 1, 120)},
		{Ratio: 1, Level: 0.4, Velocity: 0.3, Env: adsr.NewSetting(1, 30, 0.9, 100, 1, 120)},
	})
}

// BassPreset is a punchy FM bass: a stack whose top operator snaps shut quickly
func BassPreset() Patch {
	return fmPatch("bass", 0, 0.3, 0.4003, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0, Env: adsr.NewSetting(1, 1, 0.8, 200, 1, 80)},
		{Ratio: 1, Level: 0.5, Velocity: 0.6, Env: adsr.NewSetting(1, 1, 0.25, 180, 1, 80)},
		{Ratio: 2, Level: 0.25, Velocity: 0.5, Env: adsr.NewSetting(1, 1, 0, 90, 1, 50)},
		{Ratio: 1, Level: 0.15, Velocity: 0.3, Env: adsr.NewSetting(1, 1, 0, 60, 1, 40)},
	})
}

// BellPreset is a clear bell with inharmonic partials from a 1:3.5 ratio
func BellPreset() Patch {
	return fmPatch("bell", 4, 0, 0.6964, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 4000, 1, 1500)},
		{Ratio: 3.5, Level: 0.6, Velocity: 0.6, Env: adsr.NewSetting(1, 1, 0, 2500, 1, 1000)},
		{Ratio: 2, Level: 0.5, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 3000, 1, 1200)},
		{Ratio: 5.19, Level: 0.3, Velocity: 0.5, Env: adsr.NewSetting(1, 1, 0, 1500, 1, 800)},
	})
}

// BrassPreset is a brass section like tone: the modulators swell in slower than the carrier
func BrassPreset() Patch {
	return fmPatch("brass", 1, 0.25, 0.4089, fmExtras{VibratoRate: 5, VibratoDepth: 0.1, VibratoDelay: 400}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 40, 0.85, 200, 1, 200)},
		{Ratio: 1, Level: 0.45, Velocity: 0.7, Env: adsr.NewSetting(1, 90, 0.7, 300, 1, 200)},
		{Ratio: 1, Level: 0.2, Velocity: 0.5, Env: adsr.NewSetting(1, 120, 0.6, 300, 1, 200)},
		{Ratio: 1, Level: 0.2, Velocity: 0.5, Detune: 7, Env: adsr.NewSetting(1, 80, 0.5, 300, 1, 200)},
	})
}

// CelestaPreset: a soft, sparkling bell keyboard
func CelestaPreset() Patch {
	return fmPatch("celesta", 4, 0, 1.013, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 1400, 1, 500)},
		{Ratio: 4, Level: 0.3, Velocity: 0.6, Env: adsr.NewSetting(1, 1, 0, 300, 1, 200)},
		{Ratio: 4, Level: 0.25, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 900, 1, 300)},
		{Ratio: 9.5, Level: 0.15, Velocity: 0.5, Env: adsr.NewSetting(1, 1, 0, 150, 1, 100)},
	})
}

// ChurchOrganPreset: a full, slowly speaking pipe organ
func ChurchOrganPreset() Patch {
	return fmPatch("churchOrgan", 7, 0.05, 0.709, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0, Env: adsr.NewSetting(1, 60, 1, 10, 1, 400)},
		{Ratio: 2, Level: 0.8, Velocity: 0, Env: adsr.NewSetting(1, 70, 1, 10, 1, 400)},
		{Ratio: 4, Level: 0.5, Velocity: 0, Env: adsr.NewSetting(1, 80, 1, 10, 1, 400)},
		{Ratio: 0.5, Level: 0.8, Velocity: 0, Env: adsr.NewSetting(1, 60, 1, 10, 1, 400)},
	})
}

// ClarinetPreset: a hollow woodwind, a 1:2 modulator gives mostly odd harmonics
func ClarinetPreset() Patch {
	return fmPatch("clarinet", 0, 0, 0.2897, fmExtras{VibratoRate: 4.8, VibratoDepth: 0.05, VibratoDelay: 500}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.1, Env: adsr.NewSetting(1, 40, 0.9, 100, 1, 150)},
		{Ratio: 2, Level: 0.35, Velocity: 0.5, Env: adsr.NewSetting(1, 40, 0.8, 200, 1, 150)},
		{Ratio: 1, Level: 0, Velocity: 0, Env: adsr.NewSetting(1, 10, 1, 10, 1, 100)},
		{Ratio: 1, Level: 0, Velocity: 0, Env: adsr.NewSetting(1, 10, 1, 10, 1, 100)},
	})
}

// ClavinetPreset: a funky, buzzy plucked key
func ClavinetPreset() Patch {
	return fmPatch("clavinet", 0, 0.4, 0.6446, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 700, 1, 60)},
		{Ratio: 1, Level: 0.6, Velocity: 0.8, Env: adsr.NewSetting(1, 1, 0.2, 400, 1, 60)},
		{Ratio: 3, Level: 0.3, Velocity: 0.4, Env: adsr.NewSetting(1, 1, 0, 200, 1, 40)},
		{Ratio: 1, Level: 0.2, Velocity: 0.3, Env: adsr.NewSetting(1, 1, 0, 150, 1, 40)},
	})
}

// EPianoPreset is a tine electric piano: a 1:1 pair for the body and a high ratio pair for the
// bright, quickly decaying attack. Velocity makes it brighter
func EPianoPreset() Patch {
	return fmPatch("ePiano", 4, 0, 1.133, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 2, 0, 2500, 1, 400)},
		{Ratio: 1, Level: 0.35, Velocity: 0.8, Env: adsr.NewSetting(1, 2, 0.1, 1200, 1, 300)},
		{Ratio: 1, Level: 0.25, Velocity: 0.3, Detune: 3, Env: adsr.NewSetting(1, 1, 0, 600, 1, 200)},
		{Ratio: 14, Level: 0.12, Velocity: 0.9, Env: adsr.NewSetting(1, 1, 0, 80, 1, 50)},
	})
}

// FlutePreset: a nearly pure tone with breath: strong feedback turns operator 4 into noise,
// which roughens operator 3
func FlutePreset() Patch {
	return fmPatch("flute", 4, 0.9, 0.6663, fmExtras{VibratoRate: 5, VibratoDepth: 0.1, VibratoDelay: 400}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.1, Env: adsr.NewSetting(1, 60, 0.9, 200, 1, 180)},
		{Ratio: 1, Level: 0.08, Velocity: 0.5, Env: adsr.NewSetting(1, 80, 0.6, 300, 1, 180)},
		{Ratio: 2, Level: 0.15, Velocity: 0.1, Env: adsr.NewSetting(1, 60, 0.9, 200, 1, 180)},
		{Ratio: 1, Level: 0.4, Velocity: 0.3, Env: adsr.NewSetting(1, 30, 0.15, 200, 1, 150)},
	})
}

// FrenchHornPreset: a mellow, round brass tone
func FrenchHornPreset() Patch {
	return fmPatch("frenchHorn", 0, 0.1, 0.4236, fmExtras{VibratoRate: 4.8, VibratoDepth: 0.06, VibratoDelay: 500}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 70, 0.9, 200, 1, 250)},
		{Ratio: 1, Level: 0.3, Velocity: 0.6, Env: adsr.NewSetting(1, 120, 0.7, 400, 1, 250)},
		{Ratio: 1, Level: 0.1, Velocity: 0.3, Env: adsr.NewSetting(1, 150, 0.6, 400, 1, 250)},
		{Ratio: 1, Level: 0.05, Velocity: 0.2, Env: adsr.NewSetting(1, 150, 0.6, 400, 1, 250)},
	})
}

// FretlessBassPreset: a smooth, singing bass that glides between notes
func FretlessBassPreset() Patch {
	return fmPatch("fretlessBass", 0, 0, 0.4053, fmExtras{Glide: 60}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.1, Env: adsr.NewSetting(1, 10, 0.8, 400, 1, 150)},
		{Ratio: 1, Level: 0.25, Velocity: 0.5, Env: adsr.NewSetting(1, 20, 0.4, 500, 1, 150)},
		{Ratio: 1, Level: 0, Velocity: 0, Env: adsr.NewSetting(1, 10, 1, 10, 1, 100)},
		{Ratio: 1, Level: 0, Velocity: 0, Env: adsr.NewSetting(1, 10, 1, 10, 1, 100)},
	})
}

// GlockenspielPreset: bright metal bars with inharmonic partials
func GlockenspielPreset() Patch {
	return fmPatch("glockenspiel", 4, 0, 0.9275, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 2500, 1, 800)},
		{Ratio: 3.5, Level: 0.4, Velocity: 0.5, Env: adsr.NewSetting(1, 1, 0, 800, 1, 300)},
		{Ratio: 2.76, Level: 0.3, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 1500, 1, 500)},
		{Ratio: 5.4, Level: 0.25, Velocity: 0.5, Env: adsr.NewSetting(1, 1, 0, 400, 1, 200)},
	})
}

// GrandPianoPreset: a body pair plus an octave partial, with a short hammer transient on top
func GrandPianoPreset() Patch {
	return fmPatch("grandPiano", 4, 0, 0.9264, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.3, Env: adsr.NewSetting(1, 1, 0, 3500, 1, 500)},
		{Ratio: 1, Level: 0.45, Velocity: 0.8, Env: adsr.NewSetting(1, 1, 0, 900, 1, 300)},
		{Ratio: 2, Level: 0.3, Velocity: 0.3, Detune: 2, Env: adsr.NewSetting(1, 1, 0, 1800, 1, 400)},
		{Ratio: 7, Level: 0.15, Velocity: 0.9, Env: adsr.NewSetting(1, 1, 0, 40, 1, 30)},
	})
}

// HarpPreset: long ringing plucked strings
func HarpPreset() Patch {
	return fmPatch("harp", 4, 0, 0.9237, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 2500, 1, 500)},
		{Ratio: 2, Level: 0.2, Velocity: 0.6, Env: adsr.NewSetting(1, 1, 0, 200, 1, 150)},
		{Ratio: 2, Level: 0.3, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 1200, 1, 400)},
		{Ratio: 3, Level: 0.15, Velocity: 0.5, Env: adsr.NewSetting(1, 1, 0, 150, 1, 100)},
	})
}

// HarpsichordPreset: a bright, plucked tone with a quick release
func HarpsichordPreset() Patch {
	return fmPatch("harpsichord", 2, 0.2, 0.4732, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.1, Env: adsr.NewSetting(1, 1, 0, 1500, 1, 150)},
		{Ratio: 3, Level: 0.4, Velocity: 0.3, Env: adsr.NewSetting(1, 1, 0, 900, 1, 150)},
		{Ratio: 1, Level: 0.3, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 600, 1, 100)},
		{Ratio: 5, Level: 0.25, Velocity: 0.3, Env: adsr.NewSetting(1, 1, 0, 300, 1, 80)},
	})
}

// HonkyTonkPreset: the grand piano with a second, detuned string
func HonkyTonkPreset() Patch {
	return fmPatch("honkyTonk", 4, 0, 0.8153, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.3, Env: adsr.NewSetting(1, 1, 0, 3500, 1, 500)},
		{Ratio: 1, Level: 0.45, Velocity: 0.8, Env: adsr.NewSetting(1, 1, 0, 900, 1, 300)},
		{Ratio: 1, Level: 0.8, Velocity: 0.3, Detune: 14, Env: adsr.NewSetting(1, 1, 0, 1800, 1, 400)},
		{Ratio: 7, Level: 0.15, Velocity: 0.9, Env: adsr.NewSetting(1, 1, 0, 40, 1, 30)},
	})
}

// KalimbaPreset: a thumb piano, a soft tine with a little metallic click
func KalimbaPreset() Patch {
	return fmPatch("kalimba", 4, 0, 1.144, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 700, 1, 200)},
		{Ratio: 6, Level: 0.15, Velocity: 0.6, Env: adsr.NewSetting(1, 1, 0, 60, 1, 40)},
		{Ratio: 3, Level: 0.2, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 300, 1, 100)},
		{Ratio: 1, Level: 0.1, Velocity: 0.3, Env: adsr.NewSetting(1, 1, 0, 100, 1, 60)},
	})
}

// KotoPreset: a plucked zither with a sharp attack
func KotoPreset() Patch {
	return fmPatch("koto", 2, 0.3, 0.5225, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 1200, 1, 200)},
		{Ratio: 3, Level: 0.35, Velocity: 0.6, Env: adsr.NewSetting(1, 1, 0, 180, 1, 100)},
		{Ratio: 1, Level: 0.25, Velocity: 0.4, Env: adsr.NewSetting(1, 1, 0, 150, 1, 80)},
		{Ratio: 1, Level: 0.25, Velocity: 0.4, Env: adsr.NewSetting(1, 1, 0, 100, 1, 60)},
	})
}

// LeadPreset is a bright lead: feedback on the top of a stack gives a saw like edge
func LeadPreset() Patch {
	return fmPatch("lead", 0, 0.6, 0.3409, fmExtras{Glide: 30, VibratoRate: 5.5, VibratoDepth: 0.2, VibratoDelay: 300}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.1, Env: adsr.NewSetting(1, 5, 0.85, 200, 1, 250)},
		{Ratio: 1, Level: 0.35, Velocity: 0.5, Env: adsr.NewSetting(1, 10, 0.6, 400, 1, 250)},
		{Ratio: 1, Level: 0.2, Velocity: 0.3, Env: adsr.NewSetting(1, 10, 0.7, 400, 1, 250)},
		{Ratio: 1, Level: 0.15, Velocity: 0.2, Env: adsr.NewSetting(1, 10, 0.8, 400, 1, 250)},
	})
}

// MalletPreset is a marimba like mallet: a short, woody attack from a 1:4 modulator
func MalletPreset() Patch {
	return fmPatch("mallet", 4, 0, 1.266, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 600, 1, 200)},
		{Ratio: 4, Level: 0.4, Velocity: 0.7, Env: adsr.NewSetting(1, 1, 0, 60, 1, 40)},
		{Ratio: 4, Level: 0.25, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 300, 1, 100)},
		{Ratio: 10, Level: 0.2, Velocity: 0.6, Env: adsr.NewSetting(1, 1, 0, 30, 1, 20)},
	})
}

// MetallicPreset is a gong / metal plate sound from inharmonic ratios and deep modulation
func MetallicPreset() Patch {
	return fmPatch("metallic", 1, 0.15, 0.3087, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 2, 0, 5000, 1, 2000)},
		{Ratio: 1.41, Level: 0.9, Velocity: 0.5, Env: adsr.NewSetting(1, 2, 0, 3500, 1, 1500)},
		{Ratio: 2.77, Level: 0.6, Velocity: 0.5, Env: adsr.NewSetting(1, 2, 0, 2500, 1, 1200)},
		{Ratio: 3.93, Level: 0.5, Velocity: 0.5, Env: adsr.NewSetting(1, 2, 0, 2000, 1, 1000)},
	})
}

// MusicBoxPreset: tiny plucked comb teeth
func MusicBoxPreset() Patch {
	return fmPatch("musicBox", 5, 0, 1.448, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 1800, 1, 400)},
		{Ratio: 2, Level: 0.4, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 900, 1, 300)},
		{Ratio: 4, Level: 0.2, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 500, 1, 200)},
		{Ratio: 7, Level: 0.2, Velocity: 0.6, Env: adsr.NewSetting(1, 1, 0, 120, 1, 80)},
	})
}

// NylonGuitarPreset: a soft, round plucked string
func NylonGuitarPreset() Patch {
	return fmPatch("nylonGuitar", 2, 0, 0.5016, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 1500, 1, 200)},
		{Ratio: 1, Level: 0.3, Velocity: 0.6, Env: adsr.NewSetting(1, 1, 0, 300, 1, 150)},
		{Ratio: 3, Level: 0.2, Velocity: 0.3, Env: adsr.NewSetting(1, 1, 0, 200, 1, 100)},
		{Ratio: 2, Level: 0.15, Velocity: 0.4, Env: adsr.NewSetting(1, 1, 0, 250, 1, 100)},
	})
}

// OboePreset: a nasal double reed
func OboePreset() Patch {
	return fmPatch("oboe", 4, 0, 0.5914, fmExtras{VibratoRate: 5, VibratoDepth: 0.1, VibratoDelay: 300}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.1, Env: adsr.NewSetting(1, 30, 0.9, 100, 1, 120)},
		{Ratio: 1, Level: 0.55, Velocity: 0.5, Env: adsr.NewSetting(1, 30, 0.8, 200, 1, 120)},
		{Ratio: 2, Level: 0.3, Velocity: 0.1, Env: adsr.NewSetting(1, 30, 0.9, 100, 1, 120)},
		{Ratio: 3, Level: 0.25, Velocity: 0.4, Env: adsr.NewSetting(1, 30, 0.8, 200, 1, 120)},
	})
}

// OrganPreset is an additive organ: four carriers at sub, unison, octave and octave plus fifth
func OrganPreset() Patch {
	return fmPatch("organ", 7, 0.1, 0.8223, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0, Env: adsr.NewSetting(1, 5, 1, 10, 1, 60)},
		{Ratio: 0.5, Level: 0.7, Velocity: 0, Env: adsr.NewSetting(1, 5, 1, 10, 1, 60)},
		{Ratio: 2, Level: 0.6, Velocity: 0, Env: adsr.NewSetting(1, 5, 1, 10, 1, 60)},
		{Ratio: 3, Level: 0.4, Velocity: 0, Env: adsr.NewSetting(1, 5, 1, 10, 1, 60)},
	})
}

// OverdriveGuitarPreset: a sustained, gritty lead guitar from heavy feedback on a stack
func OverdriveGuitarPreset() Patch {
	return fmPatch("overdriveGuitar", 0, 0.8, 0.3051, fmExtras{VibratoRate: 5.5, VibratoDepth: 0.15, VibratoDelay: 500}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.1, Env: adsr.NewSetting(1, 3, 0.7, 800, 1, 200)},
		{Ratio: 1, Level: 0.7, Velocity: 0.6, Env: adsr.NewSetting(1, 3, 0.6, 600, 1, 200)},
		{Ratio: 2, Level: 0.4, Velocity: 0.4, Env: adsr.NewSetting(1, 3, 0.5, 500, 1, 200)},
		{Ratio: 1, Level: 0.4, Velocity: 0.3, Env: adsr.NewSetting(1, 3, 0.7, 500, 1, 200)},
	})
}

// PadPreset is a warm, slowly evolving pad from detuned parallel carriers
func PadPreset() Patch {
	return fmPatch("pad", 5, 0, 0.8441, fmExtras{VibratoRate: 4.5, VibratoDepth: 0.06, VibratoDelay: 600}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0, Env: adsr.NewSetting(1, 800, 0.9, 1000, 1, 1800)},
		{Ratio: 1, Level: 0.8, Velocity: 0, Detune: 6, Env: adsr.NewSetting(1, 900, 0.9, 1000, 1, 1800)},
		{Ratio: 2, Level: 0.4, Velocity: 0, Detune: -5, Env: adsr.NewSetting(1, 1100, 0.8, 1000, 1, 1800)},
		{Ratio: 1, Level: 0.15, Velocity: 0, Env: adsr.NewSetting(1, 2000, 0.6, 2000, 1, 2000)},
	})
}

// PanFlutePreset: a breathier flute with a chiff at the start of every note
func PanFlutePreset() Patch {
	return fmPatch("panFlute", 4, 0.9, 0.6692, fmExtras{VibratoRate: 5, VibratoDepth: 0.06, VibratoDelay: 400}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.1, Env: adsr.NewSetting(1, 60, 0.9, 200, 1, 180)},
		{Ratio: 1, Level: 0.04, Velocity: 0.5, Env: adsr.NewSetting(1, 80, 0.6, 300, 1, 180)},
		{Ratio: 2, Level: 0.15, Velocity: 0.1, Env: adsr.NewSetting(1, 60, 0.9, 200, 1, 180)},
		{Ratio: 1, Level: 0.7, Velocity: 0.4, Env: adsr.NewSetting(1, 5, 0.1, 120, 1, 150)},
	})
}

// PercussiveOrganPreset: an additive organ with a decaying third harmonic click on every note
func PercussiveOrganPreset() Patch {
	return fmPatch("percussiveOrgan", 7, 0, 0.9195, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0, Env: adsr.NewSetting(1, 3, 1, 10, 1, 60)},
		{Ratio: 0.5, Level: 0.6, Velocity: 0, Env: adsr.NewSetting(1, 3, 1, 10, 1, 60)},
		{Ratio: 2, Level: 0.5, Velocity: 0, Env: adsr.NewSetting(1, 3, 1, 10, 1, 60)},
		{Ratio: 3, Level: 0.7, Velocity: 0, Env: adsr.NewSetting(1, 1, 0, 250, 1, 60)},
	})
}

// PizzicatoPreset: plucked strings
func PizzicatoPreset() Patch {
	return fmPatch("pizzicato", 4, 0, 1.35, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 400, 1, 100)},
		{Ratio: 1, Level: 0.3, Velocity: 0.7, Env: adsr.NewSetting(1, 1, 0, 80, 1, 50)},
		{Ratio: 2, Level: 0.4, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 250, 1, 80)},
		{Ratio: 1, Level: 0.2, Velocity: 0.5, Env: adsr.NewSetting(1, 1, 0, 60, 1, 40)},
	})
}

// PluckPreset is a bright plucked string: a modulator that closes quickly, like a harp or koto
func PluckPreset() Patch {
	return fmPatch("pluck", 2, 0.2, 0.5593, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 900, 1, 300)},
		{Ratio: 3, Level: 0.3, Velocity: 0.7, Env: adsr.NewSetting(1, 1, 0, 150, 1, 100)},
		{Ratio: 1, Level: 0.2, Velocity: 0.5, Env: adsr.NewSetting(1, 1, 0, 200, 1, 100)},
		{Ratio: 1, Level: 0.3, Velocity: 0.6, Env: adsr.NewSetting(1, 1, 0, 120, 1, 80)},
	})
}

// SawLeadPreset: operator 4 with strong feedback is a saw like oscillator, operator 1 adds body
func SawLeadPreset() Patch {
	return fmPatch("sawLead", 7, 0.75, 0.8143, fmExtras{Glide: 25, VibratoRate: 5.5, VibratoDepth: 0.15, VibratoDelay: 300}, [4]fmOp{
		{Ratio: 1, Level: 0.3, Velocity: 0, Env: adsr.NewSetting(1, 5, 0.9, 100, 1, 150)},
		{Ratio: 1, Level: 0, Velocity: 0, Env: adsr.NewSetting(1, 10, 1, 10, 1, 100)},
		{Ratio: 1, Level: 0, Velocity: 0, Env: adsr.NewSetting(1, 10, 1, 10, 1, 100)},
		{Ratio: 1, Level: 1, Velocity: 0, Env: adsr.NewSetting(1, 5, 0.9, 100, 1, 150)},
	})
}

// SaxophonePreset: a reedy, breathy sax with vibrato
func SaxophonePreset() Patch {
	return fmPatch("saxophone", 1, 0.5, 0.4544, fmExtras{VibratoRate: 5, VibratoDepth: 0.15, VibratoDelay: 300}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 40, 0.85, 200, 1, 150)},
		{Ratio: 1, Level: 0.5, Velocity: 0.7, Env: adsr.NewSetting(1, 50, 0.7, 300, 1, 150)},
		{Ratio: 3, Level: 0.12, Velocity: 0.4, Env: adsr.NewSetting(1, 60, 0.6, 300, 1, 150)},
		{Ratio: 1, Level: 0.1, Velocity: 0.3, Env: adsr.NewSetting(1, 40, 0.6, 300, 1, 150)},
	})
}

// SitarPreset: a plucked string with the characteristic buzzing bridge (feedback and a high partial)
func SitarPreset() Patch {
	return fmPatch("sitar", 1, 0.6, 0.4232, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 2500, 1, 500)},
		{Ratio: 1, Level: 0.5, Velocity: 0.6, Env: adsr.NewSetting(1, 1, 0, 1500, 1, 400)},
		{Ratio: 9.97, Level: 0.2, Velocity: 0.3, Env: adsr.NewSetting(1, 1, 0, 800, 1, 300)},
		{Ratio: 1, Level: 0.3, Velocity: 0.3, Env: adsr.NewSetting(1, 1, 0, 1200, 1, 400)},
	})
}

// SlapBassPreset: a snappy, bright bass with a percussive pop
func SlapBassPreset() Patch {
	return fmPatch("slapBass", 0, 0.5, 0.6177, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0.5, 300, 1, 100)},
		{Ratio: 1, Level: 0.7, Velocity: 0.8, Env: adsr.NewSetting(1, 1, 0.15, 120, 1, 80)},
		{Ratio: 3, Level: 0.4, Velocity: 0.5, Env: adsr.NewSetting(1, 1, 0, 40, 1, 30)},
		{Ratio: 1, Level: 0.3, Velocity: 0.4, Env: adsr.NewSetting(1, 1, 0, 30, 1, 30)},
	})
}

// SquareLeadPreset: a hollow, square like lead (1:2 modulator) with glide
func SquareLeadPreset() Patch {
	return fmPatch("squareLead", 0, 0, 0.3571, fmExtras{Glide: 20, VibratoRate: 5.5, VibratoDepth: 0.15, VibratoDelay: 300}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.1, Env: adsr.NewSetting(1, 5, 0.9, 100, 1, 150)},
		{Ratio: 2, Level: 0.45, Velocity: 0.4, Env: adsr.NewSetting(1, 5, 0.9, 100, 1, 150)},
		{Ratio: 1, Level: 0, Velocity: 0, Env: adsr.NewSetting(1, 10, 1, 10, 1, 100)},
		{Ratio: 1, Level: 0, Velocity: 0, Env: adsr.NewSetting(1, 10, 1, 10, 1, 100)},
	})
}

// SteelDrumPreset: a bright, ringing steel pan
func SteelDrumPreset() Patch {
	return fmPatch("steelDrum", 4, 0, 0.997, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 900, 1, 250)},
		{Ratio: 2, Level: 0.35, Velocity: 0.6, Env: adsr.NewSetting(1, 1, 0, 250, 1, 150)},
		{Ratio: 3, Level: 0.3, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 600, 1, 200)},
		{Ratio: 1, Level: 0.2, Velocity: 0.5, Env: adsr.NewSetting(1, 1, 0, 150, 1, 100)},
	})
}

// SteelGuitarPreset: a brighter, ringing steel string
func SteelGuitarPreset() Patch {
	return fmPatch("steelGuitar", 2, 0.25, 0.5209, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 2000, 1, 250)},
		{Ratio: 1, Level: 0.5, Velocity: 0.7, Env: adsr.NewSetting(1, 1, 0.05, 500, 1, 200)},
		{Ratio: 4, Level: 0.25, Velocity: 0.3, Env: adsr.NewSetting(1, 1, 0, 250, 1, 100)},
		{Ratio: 1, Level: 0.3, Velocity: 0.4, Env: adsr.NewSetting(1, 1, 0, 300, 1, 100)},
	})
}

// StringsPreset: a string section from three detuned saw like carriers sharing one modulator
func StringsPreset() Patch {
	return fmPatch("strings", 5, 0, 0.8892, fmExtras{VibratoRate: 5.2, VibratoDepth: 0.08, VibratoDelay: 400}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0, Env: adsr.NewSetting(1, 250, 0.9, 500, 1, 600)},
		{Ratio: 1, Level: 0.9, Velocity: 0, Detune: 7, Env: adsr.NewSetting(1, 250, 0.9, 500, 1, 600)},
		{Ratio: 1, Level: 0.8, Velocity: 0, Detune: -7, Env: adsr.NewSetting(1, 250, 0.9, 500, 1, 600)},
		{Ratio: 1, Level: 0.25, Velocity: 0.3, Env: adsr.NewSetting(1, 300, 0.8, 500, 1, 600)},
	})
}

// SweepPadPreset: the modulator rises slowly, so every chord brightens over a few seconds
func SweepPadPreset() Patch {
	return fmPatch("sweepPad", 0, 0, 0.3823, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0, Env: adsr.NewSetting(1, 1500, 0.9, 500, 1, 2500)},
		{Ratio: 1, Level: 0.6, Velocity: 0, Detune: 4, Env: adsr.NewSetting(1, 4000, 1, 100, 1, 2500)},
		{Ratio: 2, Level: 0.15, Velocity: 0, Env: adsr.NewSetting(1, 3000, 1, 100, 1, 2500)},
		{Ratio: 1, Level: 0, Velocity: 0, Env: adsr.NewSetting(1, 10, 1, 10, 1, 100)},
	})
}

// TaikoPreset: a big, low drum: a noisy burst from operator 4 (full feedback) for the skin hit
func TaikoPreset() Patch {
	return fmPatch("taiko", 4, 0.95, 1.145, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.3, Env: adsr.NewSetting(1, 2, 0, 900, 1, 300)},
		{Ratio: 1.4, Level: 0.5, Velocity: 0.6, Env: adsr.NewSetting(1, 1, 0, 80, 1, 50)},
		{Ratio: 2.1, Level: 0.2, Velocity: 0.3, Env: adsr.NewSetting(1, 2, 0, 400, 1, 150)},
		{Ratio: 1, Level: 0.3, Velocity: 0.5, Env: adsr.NewSetting(1, 1, 0, 50, 1, 30)},
	})
}

// TimpaniPreset: a tuned kettle drum with inharmonic overtones
func TimpaniPreset() Patch {
	return fmPatch("timpani", 4, 0, 0.9861, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.3, Env: adsr.NewSetting(1, 2, 0, 1500, 1, 500)},
		{Ratio: 1.5, Level: 0.4, Velocity: 0.6, Env: adsr.NewSetting(1, 2, 0, 200, 1, 100)},
		{Ratio: 1.68, Level: 0.4, Velocity: 0.3, Env: adsr.NewSetting(1, 2, 0, 800, 1, 300)},
		{Ratio: 2.3, Level: 0.3, Velocity: 0.5, Env: adsr.NewSetting(1, 1, 0, 100, 1, 80)},
	})
}

// TrumpetPreset: a bright brass tone that opens up as it speaks, with delayed vibrato
func TrumpetPreset() Patch {
	return fmPatch("trumpet", 1, 0.35, 0.3844, fmExtras{VibratoRate: 5.5, VibratoDepth: 0.12, VibratoDelay: 350}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 30, 0.85, 200, 1, 150)},
		{Ratio: 1, Level: 0.55, Velocity: 0.7, Env: adsr.NewSetting(1, 60, 0.75, 300, 1, 150)},
		{Ratio: 2, Level: 0.15, Velocity: 0.4, Env: adsr.NewSetting(1, 80, 0.6, 300, 1, 150)},
		{Ratio: 1, Level: 0.2, Velocity: 0.4, Env: adsr.NewSetting(1, 50, 0.6, 300, 1, 150)},
	})
}

// TubularBellsPreset: long, deep chimes
func TubularBellsPreset() Patch {
	return fmPatch("tubularBells", 1, 0, 0.3108, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 6000, 1, 2500)},
		{Ratio: 3.5, Level: 0.7, Velocity: 0.5, Env: adsr.NewSetting(1, 1, 0, 3000, 1, 1500)},
		{Ratio: 5.6, Level: 0.3, Velocity: 0.3, Env: adsr.NewSetting(1, 1, 0, 1500, 1, 800)},
		{Ratio: 2.4, Level: 0.3, Velocity: 0.3, Env: adsr.NewSetting(1, 1, 0, 2000, 1, 1000)},
	})
}

// UprightBassPreset: a plucked acoustic bass, round with a soft finger attack
func UprightBassPreset() Patch {
	return fmPatch("uprightBass", 2, 0, 0.5352, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 5, 0, 1200, 1, 150)},
		{Ratio: 1, Level: 0.3, Velocity: 0.5, Env: adsr.NewSetting(1, 5, 0, 250, 1, 100)},
		{Ratio: 2, Level: 0.15, Velocity: 0.3, Env: adsr.NewSetting(1, 1, 0, 150, 1, 80)},
		{Ratio: 1, Level: 0.1, Velocity: 0.3, Env: adsr.NewSetting(1, 1, 0, 60, 1, 40)},
	})
}

// WarmPadPreset: a soft, slowly blooming pad
func WarmPadPreset() Patch {
	return fmPatch("warmPad", 5, 0, 0.4761, fmExtras{VibratoRate: 5.2, VibratoDepth: 0.05, VibratoDelay: 400}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0, Env: adsr.NewSetting(1, 1200, 0.9, 1000, 1, 2000)},
		{Ratio: 1, Level: 0.9, Velocity: 0, Detune: 7, Env: adsr.NewSetting(1, 1200, 0.9, 1000, 1, 2000)},
		{Ratio: 1, Level: 0.8, Velocity: 0, Detune: -7, Env: adsr.NewSetting(1, 1200, 0.9, 1000, 1, 2000)},
		{Ratio: 2, Level: 0.15, Velocity: 0, Env: adsr.NewSetting(1, 2000, 0.7, 1500, 1, 2000)},
	})
}

// WoodblockPreset: a short, hollow knock
func WoodblockPreset() Patch {
	return fmPatch("woodblock", 4, 0, 1.347, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 80, 1, 40)},
		{Ratio: 2.53, Level: 0.4, Velocity: 0.5, Env: adsr.NewSetting(1, 1, 0, 30, 1, 20)},
		{Ratio: 3.7, Level: 0.4, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 60, 1, 40)},
		{Ratio: 1, Level: 0.2, Velocity: 0.5, Env: adsr.NewSetting(1, 1, 0, 20, 1, 20)},
	})
}

// XylophonePreset: short, dry wooden bars
func XylophonePreset() Patch {
	return fmPatch("xylophone", 4, 0, 1.512, fmExtras{}, [4]fmOp{
		{Ratio: 1, Level: 1, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 350, 1, 100)},
		{Ratio: 3, Level: 0.35, Velocity: 0.7, Env: adsr.NewSetting(1, 1, 0, 40, 1, 30)},
		{Ratio: 3.93, Level: 0.25, Velocity: 0.2, Env: adsr.NewSetting(1, 1, 0, 150, 1, 60)},
		{Ratio: 10, Level: 0.2, Velocity: 0.6, Env: adsr.NewSetting(1, 1, 0, 20, 1, 20)},
	})
}
