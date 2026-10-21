// Package catalog is every synth and effect of psalm as a kind of module of the patcher
// (modules/controls/panel/patcher), each with its presets and its settings, and a keyboard to play
// them from.
//
//	kinds := catalog.Kinds(catalog.Config{Input: keys, IR: hall})
//	pt := patcher.New(p, ctl, "Patch", kinds)
package catalog

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/shapechain"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/keyboard"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/dxeditor"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/script"
	"github.com/almerlucke/psalm/modules/effects/chorus"
	"github.com/almerlucke/psalm/modules/effects/compressor"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/effects/crusher"
	"github.com/almerlucke/psalm/modules/effects/eq"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/flanger"
	"github.com/almerlucke/psalm/modules/effects/follower"
	"github.com/almerlucke/psalm/modules/effects/gate"
	"github.com/almerlucke/psalm/modules/effects/granulator"
	"github.com/almerlucke/psalm/modules/effects/phaser"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/effects/pitchshift"
	"github.com/almerlucke/psalm/modules/effects/ringmod"
	"github.com/almerlucke/psalm/modules/effects/rotary"
	"github.com/almerlucke/psalm/modules/effects/saturate"
	"github.com/almerlucke/psalm/modules/effects/shimmer"
	"github.com/almerlucke/psalm/modules/effects/tape"
	"github.com/almerlucke/psalm/modules/effects/transient"
	"github.com/almerlucke/psalm/modules/effects/tremolo"
	"github.com/almerlucke/psalm/modules/effects/vocoder"
	"github.com/almerlucke/psalm/modules/effects/widener"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/synths/drumsynth"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/grains"
	"github.com/almerlucke/psalm/synths/modal"
	"github.com/almerlucke/psalm/synths/organ"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/sampler"
	"github.com/almerlucke/psalm/synths/shaper"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/supersaw"
	"github.com/almerlucke/psalm/synths/vocal"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/sndfile"
)

// Config is what the kinds need from outside
type Config struct {
	// Input is what keyboards play from: a MIDI device, the page's keys (nil: keyboards play nothing)
	Input midi.Input
	// IR is an impulse response for a hall (the convolver), nil leaves it out
	IR [][]float64
	// Voices is how many notes a synth plays at once (0 is 12)
	Voices int
	// Drums is the folder of the drum samples of the drum machine (see drummachine.go); "" is
	// assets/sounds/drums. Without the samples the drum machine is left out
	Drums string
	// SysEx is the folder of SysEx files of the DX7 editor (list, load as the bank, save into); ""
	// leaves it off
	SysEx string
}

var (
	ins  = []patcher.Port{{Name: "in", Type: patcher.Audio}}
	outs = []patcher.Port{{Name: "out", Type: patcher.Audio}}
	note = []patcher.Port{{Name: "notes", Type: patcher.Notes}}
)

// synth is a kind of synth: its presets (sorted), how one is made at a preset and described
func synth(name string, presets []string, make func(preset string, voices int) psalm.Modular, describe func(preset string) params.Set, voices int) patcher.Kind {
	return patcher.Kind{Name: name, Group: "Synths", Presets: presets, Ins: note, Outs: outs, Make: func(preset string) (*patcher.Made, error) {
		return &patcher.Made{Module: make(preset, voices), Describe: describe,
			Rebuild: func(preset string) psalm.Modular { return make(preset, voices) }}, nil
	}}
}

// shaperKind is the shaper synth: the usual presets and settings, and the four chains it morphs
// through as text in its side panel
func shaperKind(voices int) patcher.Kind {
	texts := func(preset string) map[string]string {
		cfg := shaper.Presets[preset]()
		out := map[string]string{}
		for i, key := range shaper.ChainKeys {
			out[key] = cfg.Chains[i]
		}
		return out
	}
	return patcher.Kind{Name: "shaper", Group: "Synths", Presets: sorted(shaper.Presets), Ins: note, Outs: outs,
		Make: func(preset string) (*patcher.Made, error) {
			mk := func(name string, v int) psalm.Modular { return shaper.NewPoly(shaper.Presets[name](), v) }
			made := &patcher.Made{Module: mk(preset, voices),
				Describe:    func(n string) params.Set { return shaper.Describe(shaper.Presets[n]()) },
				Rebuild:     func(n string) psalm.Modular { return mk(n, voices) },
				PresetTexts: texts,
				TextMessage: func(key, value string) (psalm.Message, error) {
					if !slices.Contains(shaper.ChainKeys[:], key) {
						return nil, errors.New("no such chain")
					}
					if strings.TrimSpace(value) == "" {
						if key == shaper.ChainKeys[0] {
							return nil, errors.New("chain A can't be empty")
						}
						return psalm.Message{key: ""}, nil
					}
					c, err := shapechain.Parse(value)
					if err != nil {
						return nil, err
					}
					return psalm.Message{key: c}, nil
				}}
			t := texts(preset)
			for i, key := range shaper.ChainKeys {
				made.Texts = append(made.Texts, patcher.Text{Key: key, Label: "chain " + string(rune('A'+i)), Value: t[key], Lines: 3, Help: shaper.ChainHelp})
			}
			return made, nil
		}}
}

func sorted[T any](m map[string]T) []string { return slices.Sorted(maps.Keys(m)) }

// effect is a kind of effect: how one is made, and described as it is
func effect(name string, make func() (psalm.Modular, func() params.Set)) patcher.Kind {
	return patcher.Kind{Name: name, Group: "Effects", Ins: ins, Outs: outs, Make: func(string) (*patcher.Made, error) {
		m, set := make()
		return &patcher.Made{Module: m, Describe: func(string) params.Set { return set() }}, nil
	}}
}

// Kinds returns the kinds: the patcher's own (LFO, random, mixer; clock, sequence, notes, quantize),
// scripts (of JavaScript, all of them in one runtime: one Kinds for a patch), a keyboard, every
// synth (with its presets) and every effect
func Kinds(cfg Config) []patcher.Kind {
	v := cfg.Voices
	if v <= 0 {
		v = 12
	}
	table := gen.Brightness(32)
	h := &hub{input: cfg.Input}
	kinds := []patcher.Kind{keyboardKind(h)}
	kinds = append(kinds, patcher.Sources()...)
	kinds = append(kinds, patcher.Generators()...)
	kinds = append(kinds, script.Kind(script.NewRuntime()))
	kinds = append(kinds,
		synth("subtractive", sorted(subtractive.Presets), func(n string, v int) psalm.Modular { return subtractive.NewPoly(subtractive.Presets[n](), v) },
			func(n string) params.Set { return subtractive.Describe(subtractive.Presets[n]()) }, v),
		dxKind(v, cfg.SysEx),
		synth("wavetable", sorted(wavetable.Presets), func(n string, v int) psalm.Modular { return wavetable.NewPoly(wavetable.Presets[n](table), v) },
			func(n string) params.Set { return wavetable.Describe(wavetable.Presets[n](table)) }, v),
		synth("supersaw", sorted(supersaw.Presets), func(n string, v int) psalm.Modular { return supersaw.NewPoly(supersaw.Presets[n](), v) },
			func(n string) params.Set { return supersaw.Describe(supersaw.Presets[n]()) }, v),
		shaperKind(v),
		samplerKind(v),
		grainsKind(v),
		synth("pulsar", sorted(pulsar.Presets), func(n string, v int) psalm.Modular { return pulsar.NewPoly(pulsar.Presets[n](), v) },
			func(n string) params.Set { return pulsar.Describe(pulsar.Presets[n]()) }, v),
		synth("organ", sorted(organ.Presets), func(n string, _ int) psalm.Modular { return organ.New(organ.Presets[n]()) },
			func(n string) params.Set { return organ.Describe(organ.Presets[n]()) }, v),
		synth("piano", sorted(piano.Presets), func(n string, v int) psalm.Modular { return piano.NewPoly(piano.Presets[n](), 2*v) },
			func(n string) params.Set { return piano.Describe(piano.Presets[n]()) }, v),
		synth("waveguide", sorted(waveguide.Presets), func(n string, v int) psalm.Modular { return waveguide.NewPoly(waveguide.Presets[n](), v) },
			func(n string) params.Set { return waveguide.Describe(waveguide.Presets[n]()) }, v),
		synth("modal", sorted(modal.Presets), func(n string, v int) psalm.Modular { return modal.NewPoly(modal.Presets[n](), v) },
			func(n string) params.Set { return modal.Describe(modal.Presets[n]()) }, v),
		synth("vocal", sorted(vocal.Presets), func(n string, v int) psalm.Modular { return vocal.NewPoly(vocal.Presets[n](), max(4, v*2/3)) },
			func(n string) params.Set { return vocal.Describe(vocal.Presets[n]()) }, v),
		synth("drumsynth", sorted(drumsynth.Kits), func(n string, v int) psalm.Modular {
			return drumsynth.NewPoly(drumsynth.Config{Kit: drumsynth.Kits[n]()}, max(16, v))
		}, func(n string) params.Set { return drumsynth.Describe(drumsynth.Config{Kit: drumsynth.Kits[n]()}) }, v),
	)
	kinds = append(kinds,
		effect("chorus", func() (psalm.Modular, func() params.Set) {
			m := chorus.New(chorus.DefaultConfig())
			return m, func() params.Set { return chorus.Describe(m.Settings()) }
		}),
		effect("flanger", func() (psalm.Modular, func() params.Set) {
			m := flanger.New(flanger.DefaultConfig())
			return m, func() params.Set { return flanger.Describe(m.Settings()) }
		}),
		effect("phaser", func() (psalm.Modular, func() params.Set) {
			m := phaser.New(phaser.DefaultConfig())
			return m, func() params.Set { return phaser.Describe(m.Settings()) }
		}),
		effect("tremolo", func() (psalm.Modular, func() params.Set) {
			m := tremolo.New(tremolo.DefaultConfig())
			return m, func() params.Set { return tremolo.Describe(m.Settings()) }
		}),
		effect("rotary", func() (psalm.Modular, func() params.Set) {
			m := rotary.New(rotary.DefaultConfig())
			return m, func() params.Set { return rotary.Describe(m.Settings()) }
		}),
		effect("ringmod", func() (psalm.Modular, func() params.Set) {
			m := ringmod.New(ringmod.DefaultConfig())
			return m, func() params.Set { return ringmod.Describe(m.Settings()) }
		}),
		effect("wah", func() (psalm.Modular, func() params.Set) {
			m := follower.NewWah(follower.DefaultWah())
			return m, func() params.Set { return follower.DescribeWah(m.Settings()) }
		}),
		effect("saturate", func() (psalm.Modular, func() params.Set) {
			m := saturate.New(saturate.DefaultConfig())
			return m, func() params.Set { return saturate.Describe(m.Settings()) }
		}),
		effect("crusher", func() (psalm.Modular, func() params.Set) {
			m := crusher.New(crusher.DefaultConfig())
			return m, func() params.Set { return crusher.Describe(m.Settings()) }
		}),
		effect("tape", func() (psalm.Modular, func() params.Set) {
			m := tape.New(tape.DefaultConfig())
			return m, func() params.Set { return tape.Describe(m.Settings()) }
		}),
		effect("pingpong", func() (psalm.Modular, func() params.Set) {
			m := pingpong.New(pingpong.DefaultConfig())
			return m, func() params.Set { return pingpong.Describe(m.Settings()) }
		}),
		effect("pitchshift", func() (psalm.Modular, func() params.Set) {
			m := pitchshift.New(7, 60, 0.5)
			return m, func() params.Set { return pitchshift.Describe(m) }
		}),
		effect("granulator", func() (psalm.Modular, func() params.Set) {
			m := granulator.New(granulator.DefaultConfig())
			return m, func() params.Set { return granulator.Describe(m.Settings()) }
		}),
		effect("reverb", func() (psalm.Modular, func() params.Set) {
			m := fdnreverb.New(fdnreverb.DefaultConfig())
			return m, func() params.Set { return fdnreverb.Describe(m.Settings()) }
		}),
		effect("shimmer", func() (psalm.Modular, func() params.Set) {
			m := shimmer.New(shimmer.DefaultConfig())
			return m, func() params.Set { return shimmer.Describe(m.Settings()) }
		}),
		effect("compressor", func() (psalm.Modular, func() params.Set) {
			m := compressor.New(compressor.DefaultConfig())
			return m, func() params.Set { return compressor.Describe(m.Settings()) }
		}),
		effect("gate", func() (psalm.Modular, func() params.Set) {
			m := gate.New(gate.DefaultConfig())
			return m, func() params.Set { return gate.Describe(m.Settings()) }
		}),
		effect("transient", func() (psalm.Modular, func() params.Set) {
			m := transient.New(transient.DefaultConfig())
			return m, func() params.Set { return transient.Describe(m.Settings()) }
		}),
		effect("eq", func() (psalm.Modular, func() params.Set) {
			m := eq.New(eq.DefaultConfig())
			return m, func() params.Set { return eq.Describe(m.Settings()) }
		}),
		effect("widener", func() (psalm.Modular, func() params.Set) {
			m := widener.New(widener.DefaultConfig())
			return m, func() params.Set { return widener.Describe(m.Settings()) }
		}),
	)
	kinds = append(kinds, vocoderKind())
	if cfg.IR != nil {
		kinds = append(kinds, effect("hall", func() (psalm.Modular, func() params.Set) {
			m := convolver.New(cfg.IR, convolver.Config{PartitionSize: 512, Mix: 0.35, Gain: 0.7})
			return m, func() params.Set { return convolver.Describe(m.Settings()) }
		}))
	}
	if k, ok := drumMachineKind(cfg.Drums); ok {
		kinds = append(kinds, k)
	}
	return kinds
}

// vocoderKind is the vocoder: a voice (the modulator) on its first input and a synth (the carrier) on its second. What comes
// out is the synth saying what the voice says; it sounds a frame late, which the mixer makes up for on the other strips
func vocoderKind() patcher.Kind {
	return patcher.Kind{Name: "vocoder", Group: "Effects", Outs: outs,
		Ins: []patcher.Port{{Name: "voice", Type: patcher.Audio, Index: 0}, {Name: "carrier", Type: patcher.Audio, Index: 1}},
		Make: func(string) (*patcher.Made, error) {
			m := vocoder.New(vocoder.DefaultConfig())
			return &patcher.Made{Module: m, Describe: func(string) params.Set { return vocoder.Describe(m.Settings()) }}, nil
		}}
}

// dxKind is the DX7, with its own editor
func dxKind(voices int, sysex string) patcher.Kind {
	names := sorted(dx.Presets)
	patch := func(n string) dx.Patch {
		p := dx.Presets[n]()
		p.Name = n
		return p
	}
	return patcher.Kind{Name: "dx", Group: "Synths", Presets: names, Ins: note, Outs: outs, Make: func(preset string) (*patcher.Made, error) {
		synth := dx.NewPoly(patch(preset), voices, 2)
		return &patcher.Made{Module: synth,
			Describe: func(n string) params.Set { return dx.Describe(patch(n)) },
			Edit: func(page *panel.Panel, name, preset string) patcher.Settings {
				var bank []dx.Patch
				for _, n := range names {
					bank = append(bank, patch(n))
				}
				opts := []dxeditor.Option{dxeditor.Bank(bank)}
				if sysex != "" {
					opts = append(opts, dxeditor.SysExFolder(sysex))
				}
				return dxeditor.Add(page, name, synth, patch(preset), opts...)
			}}, nil
	}}
}

// keyboardKind is a keyboard: notes from the input (a MIDI keyboard, the page's keys) to the synths
// its cables go to, and the mod wheel and the pressure as modulation
func keyboardKind(h *hub) patcher.Kind {
	return patcher.Kind{Name: "keyboard", Group: "Sources",
		Outs: []patcher.Port{{Name: "notes", Type: patcher.Notes}, {Name: "wheel", Type: patcher.Mod, Index: 0}, {Name: "pressure", Type: patcher.Mod, Index: 1}},
		Make: func(string) (*patcher.Made, error) {
			kb := keyboard.New(h.reader(), keyboard.Config{Dynamics: 1, Controls: map[int]string{midi.CCModWheel: "modWheel", midi.CCSustain: "sustain"},
				HoldNotes: true, PitchBend: "pitchBend", Pressure: "pressure", PolyPressure: "pressure"})
			var wheel, pressure float64 // on the audio thread
			kb.OnControl(midi.CCModWheel, func(v float64, _ psalm.CtrlTick) { wheel = v })
			kb.OnEvent(func(e midi.Event, _ psalm.CtrlTick) {
				if e.Message.Kind() == midi.ChannelPressure {
					pressure = float64(e.Message.Pressure()) / 127
				}
			})
			return &patcher.Made{Module: kb,
				Notes: func(targets []psalm.Receiver) {
					kb.AllNotesOff(psalm.CtrlTick{})
					kb.SetTargets(targets...)
				},
				Mod: func(out int) func() float64 {
					if out == 1 {
						return func() float64 { return pressure }
					}
					return func() float64 { return wheel }
				}}, nil
		}}
}

// hub shares an input between keyboards that come and go: each reads every event of every buffer
type hub struct {
	mu     sync.Mutex
	input  midi.Input
	events []midi.Event
	reads  int64
}

type hubReader struct {
	h     *hub
	reads int64
}

func (h *hub) reader() midi.Input {
	h.mu.Lock()
	defer h.mu.Unlock()
	return &hubReader{h: h, reads: h.reads}
}

func (r *hubReader) Read(events []midi.Event, frames int, sampleRate float64) []midi.Event {
	h := r.h
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.input == nil {
		return events
	}
	if r.reads >= h.reads {
		clear(h.events)
		h.events = h.input.Read(h.events[:0], frames, sampleRate)
		h.reads++
	}
	r.reads = h.reads
	return append(events, h.events...)
}

// samplerKind is the sampler: its sounds are a text in its side panel, a line each (a first key, a file, how it is cut into pieces: hits,
// change, even or whole, or an SFZ instrument; see sampler.ParseBanks), so a loop or a chant becomes a kit on the keys that a script
// or a keyboard plays. A text that can't be read (a missing file, a mode it doesn't know) leaves the sounds as they were
func samplerKind(voices int) patcher.Kind {
	const help = "- a line a sound: `36 file.wav hits 9 0.08` (the first key, the file, how to cut it)\n" +
		"- `hits rise gap`: a piece at every hit; `change sensitivity gap`: where the sound changes (a sung line); `even n`; `whole hiKey root`\n" +
		"- `0 instrument.sfz sfz` an SFZ instrument; add `vol=-6 pan=0.3 name=word`; # starts a comment"
	var (
		mu    sync.Mutex
		cache = map[string]sampler.Sound{}
	)
	load := func(path string) (sampler.Sound, error) {
		mu.Lock()
		defer mu.Unlock()
		if snd, ok := cache[path]; ok {
			return snd, nil
		}
		snd, err := sampler.LoadFile(path)
		if err == nil {
			cache[path] = snd
		}
		return snd, err
	}
	return patcher.Kind{Name: "sampler", Group: "Synths", Ins: note, Outs: outs,
		Make: func(string) (*patcher.Made, error) {
			var status string // what the sounds made, or what is wrong with the text
			return &patcher.Made{Module: sampler.NewPoly(sampler.DefaultConfig(nil), voices),
				Texts: []patcher.Text{{Key: "sounds", Label: "sounds", Lines: 8, Help: help}},
				TextMessage: func(key, value string) (psalm.Message, error) {
					if key != "sounds" {
						return nil, errors.New("no such text")
					}
					if strings.TrimSpace(value) == "" {
						return nil, errors.New("no sounds")
					}
					inst, report, err := sampler.ParseBanks(value, load)
					mu.Lock()
					defer mu.Unlock()
					if err != nil {
						status = "error: " + err.Error()
						return nil, err
					}
					status = strings.Join(report, "\n")
					return psalm.Message{"instrument": inst}, nil
				},
				Status: func() string { mu.Lock(); defer mu.Unlock(); return status }}, nil
		}}
}

// grainsKind is the granular sampler: the usual presets and settings, and its sounds as a text in its side panel, a line each: a name, a
// file and the MIDI pitch it is at (`choir assets/sounds/chants/Johannes.Hymnus.wav 62`). The first is what it plays; a script picks
// another with set({sound: "name"}) (the pitch of that sound goes with it)
func grainsKind(voices int) patcher.Kind {
	const help = "- a line a sound: `choir file.wav 62` (a name, the file, the MIDI pitch it is at; 60 without)\n" +
		"- the first is played; a script picks another with `set({sound: \"name\"})`; # starts a comment"
	return patcher.Kind{Name: "grains", Group: "Synths", Presets: sorted(grains.Presets), Ins: note, Outs: outs,
		Make: func(preset string) (*patcher.Made, error) {
			mk := func(name string) psalm.Modular { return grains.NewPoly(grains.Presets[name](nil, 60), voices) }
			var cache = map[string]sndfile.SoundFiler[float32]{}
			return &patcher.Made{Module: mk(preset), Rebuild: mk,
				Describe: func(n string) params.Set { return grains.Describe(grains.Presets[n](nil, 60)) },
				Texts:    []patcher.Text{{Key: "sounds", Label: "sounds", Lines: 5, Help: help}},
				TextMessage: func(key, value string) (psalm.Message, error) {
					if key != "sounds" {
						return nil, errors.New("no such text")
					}
					bank, roots, first := sndfile.SoundBank[float32]{}, map[string]float64{}, ""
					for n, line := range strings.Split(value, "\n") {
						line = strings.TrimSpace(line)
						if line == "" || strings.HasPrefix(line, "#") {
							continue
						}
						f := strings.Fields(line)
						if len(f) < 2 || len(f) > 3 {
							return nil, fmt.Errorf("line %d: a name, a file and the pitch: choir file.wav 62", n+1)
						}
						root := 60.0
						if len(f) == 3 {
							var err error
							if root, err = strconv.ParseFloat(f[2], 64); err != nil {
								return nil, fmt.Errorf("line %d: %q is not a pitch", n+1, f[2])
							}
						}
						sf, ok := cache[f[1]]
						if !ok {
							var err error
							if sf, err = sndfile.NewSoundFile[float32](f[1]); err != nil {
								return nil, fmt.Errorf("line %d: %v", n+1, err)
							}
							cache[f[1]] = sf
						}
						bank[f[0]], roots[f[0]] = sf, root
						if first == "" {
							first = f[0]
						}
					}
					if first == "" {
						return nil, errors.New("no sounds")
					}
					return psalm.Message{"bank": bank, "roots": roots, "first": first}, nil
				}}, nil
		}}
}
