# Psalm documentation

Psalm is a modular audio synthesis and sequencing library for Go. You build a patch out of
modules (synths, effects, filters, controls), connect their audio and value outputs, and either
play it live through an audio engine or render it to a sound file.

## Contents

| Page | What's in it |
|---|---|
| [Core concepts](core-concepts.md) | Psalm, config, modules, audio and value connections, messages, control ticks, rendering, audio engines |
| [Controls](controls.md) | Sequencing and control modules: `timer`, `steps`, `templ`, `score` (notes in beats, notation, sequencer), `arrange` (song form, parts, automation, events), `keyboard` (MIDI, see [MIDI](midi.md)), `panel` (faders, knobs, buttons, step grids and meters on a web page), `stepseq` (a step sequencer: rolls, chances, conditions, swing, patterns), `gen`, `dispatch`, `logger` |
| [Scripting](scripting.md) | The patcher's generators and scripts: the clock's pattern language, sequence orders, notes, quantize, arp, song/lane/gate/part, and the JavaScript `script` module (how it runs, its API, every `genny.*` generator with its options and defaults, limits) |
| [Sources](sources.md) | Sound sources: `player`, `blosc` oscillator, `lfo`, `granular` |
| [Presets](presets.md) | the presets of every synth by family (about 545 more than the first ones in the synths page) |
| [Synths](synths.md) | `poly` voice allocation, the `subtractive` synth, the 6 operator `dx` engine with SysEx loading and the 4 operator FM presets, the `grains` granular sampler, the `pulsar` synth, the `wavetable` synth, the physically modelled `piano`, `drums`, the SFZ `sampler`, the tonewheel `organ`, plucked and bowed strings (`waveguide`), mallets and bells (`modal`), the `drumsynth` 808 and 909, sung voices and choirs (`vocal`), phase distortion (`pd`), the `westcoast` complex oscillator and low pass gate, the `supersaw` |
| [Effects](effects.md) | Reverbs (`freeverb`, `fdnreverb`, `shimmer`, `convolver`), `chorus`, `flanger`, `phaser`, `rotary` (Leslie), `tremolo` (and auto-pan), `pingpong`, `tape` (echo), `ringmod` (and frequency shifter), dynamics (`compressor` with sidechain, `gate`, `transient`), tone (`eq`, `saturate`, `crusher`, `widener`), `follower` (and auto-wah), `vocoder`, `granulator` (with freeze), `pitchshift`, `pan`, `gain`, `latency` |
| [MIDI](midi.md) | Playing live from MIDI: the `midi` messages, parser and inputs (raw devices, JACK MIDI, a queue) and the `keyboard` module (velocity curves, pedals, controllers, pitch bend, clock, handlers for every event) |
| [Harmony](harmony.md) | `tools/harmony`: chords from symbols, voicings and voice leading, scales and keys with roman numerals, an arpeggiator and a check for clashing notes |
| [Rendering](rendering.md) | Offline rendering with reverb tails, stems, repeatable renders (`psalm.Seed`), profiling per module and the `meter` module (peak, RMS, LUFS) |
| [Conventions](conventions.md) | Channels, units, messages, value inputs per module, tempo, randomness, silence and latency |
| [Mixer](mixer.md) | The `mixer`: channel strips (volume, pan, mute, solo, sends, meters), effect buses, groups with inserts and a master with a lookahead limiter |
| [Playlist](playlist.md) | Songs one after the other in one patch: each with its own tempo and clock, built ahead while the one before plays, ringing out, gaps or cross fades, queued and skipped from code or a panel; patches within a patch (`NewSub`) |
| [Filters](filters.md) | The `filters` module: SVF, Moog style ladder, biquad EQ and formant (vowel) filters |
| [Blocks](blocks.md) | Low level DSP building blocks used by the modules (`fft`, filters, `pitchshift`, `blosc`, `delay`, `mix`, `sine`, `oversample`), and `expression`: the pitch bend, mod wheel and aftertouch of a voice |
| [Examples](examples.md) | The example programs in `cmd/examples` and what they demonstrate |

## Quick start

```go
package main

import (
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/synths/subtractive"
)

func main() {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 1024})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	// A polyphonic synth with a preset
	synth := subtractive.NewPoly(subtractive.PluckPreset(), 8).AddTo(p)

	// Trigger a note every beat: timer -> template -> synth
	synth.ValueIn(templ.NewDurationTrigger(0.2, 0.3, template.Template{
		"pitch": 60.0,
	}).ValueIn(timer.WithTicks(float64(psalm.PPQ())).AddTo(p)))

	// Synth -> reverb -> output
	p.Out(fdnreverb.New(fdnreverb.DefaultConfig()).AddTo(p).In(synth))

	_ = p.Listen() // play until Ctrl+C
}
```

## Conventions used across modules

- **Stereo**: most effects and filters have one stereo input (index 0) and one stereo output.
  Connecting a mono source to a stereo input duplicates it, connecting a stereo source to a mono
  input mixes it down.
- **Value inputs**: modules accept plain values on numbered inputs (handy for LFOs and
  generators) and named settings through messages. Each module's page lists both.
- **Smoothing**: parameters that are commonly modulated (cutoff, pan, ...) glide over a buffer
  instead of jumping, so modulation doesn't click.
- **Units**: frequencies in Hz, times in milliseconds unless the name says seconds, pitch as MIDI
  note numbers (60 is middle C), levels as linear amplitude.
