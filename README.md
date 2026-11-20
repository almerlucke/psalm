# psalm

A modular audio synthesis and sequencing library for Go. You build a patch out of modules (synths,
effects, filters, controls), connect their audio and value outputs, and either play it live through
an audio engine or render it to a sound file, offline and repeatable.

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

	// a polyphonic synth with a preset
	synth := subtractive.NewPoly(subtractive.PluckPreset(), 8).AddTo(p)

	// a note every beat: timer -> template -> synth
	synth.ValueIn(templ.NewDurationTrigger(0.2, 0.3, template.Template{
		"pitch": 60.0,
	}).ValueIn(timer.WithTicks(float64(psalm.PPQ())).AddTo(p)))

	// synth -> reverb -> output
	p.Out(fdnreverb.New(fdnreverb.DefaultConfig()).AddTo(p).In(synth))

	_ = p.Listen() // play until Ctrl+C
}
```

Pass `nil` instead of the audio engine to build a patch without an audio device, and render it to a
file (`p.ToSoundFile`, or `render.ToFile` for a whole arrangement, with reverb tails and stems).

## What is in it

- **Synths**, with many presets each: a subtractive synth (ladder and state variable filters, glide,
  paraphonic voices), a 6 operator FM engine that loads DX7 SysEx, a physically modelled piano,
  waveguide strings and pipes, modal bells and bars, a tonewheel organ, wavetable, supersaw, pulsar
  and granular synths, 808 and 909 drum synths, sung voices and choirs, an SFZ sampler and a
  streamer for long files.
- **Effects and mixing**: reverbs (feedback delay network, freeverb, convolution, shimmer), chorus,
  flanger, phaser, rotary, tape echo, compressor with side chain, gate, EQ, vocoder, granulator,
  pitch shifter, and a mixer with channel strips, sends, buses, groups and a limiter.
- **Sequencing and composition**: a score notation (`Db3:.5_4`), `arrange` for the form of a song
  (sections, parts, automation), a step sequencer, chord, scale and voice leading tools
  (`tools/harmony`), generators and a JavaScript `script` module.
- **Live control**: MIDI in (raw devices or JACK), a keyboard module with velocity curves, pedals,
  pitch bend and aftertouch, and control panels (faders, knobs, step grids and meters) served as a
  web page, and a visual patcher.
- **Hosting** CLAP and VST3 plugins (a prototype).
- **Rendering**: offline, faster than real time, repeatable (`psalm.Seed`), with a profile per
  module and a meter for peak, RMS and LUFS.

The songs in `cmd/songs` are complete pieces made with it; `cmd/examples` has a small program for
almost every module. Both play live and render to a file:

```
go run ./cmd/songs/uno                      play live
go run ./cmd/songs/uno -render uno.wav      render to a file
```

## Documentation

Start with [docs/README.md](docs/README.md), then [core concepts](docs/core-concepts.md). There
is a page for every family of modules, the synths and their presets, [rendering](docs/rendering.md)
and [the examples](docs/examples.md).

## Requirements

- Go 1.27 or newer.
- [PortAudio](http://www.portaudio.com/) for live audio (`audio/portaudio`, through cgo); on Debian
  and Ubuntu `apt install portaudio19-dev`, on macOS `brew install portaudio`. The songs and
  examples import it, so you need it to build them, also to render to a file. JACK
  (`audio/jack`) is an alternative engine.
- The modules `github.com/almerlucke/genny` and `github.com/almerlucke/sndfile`, which `go`
  fetches for you. `sndfile` reads and writes WAV, AIFF, MP3, FLAC and Ogg in pure Go.

## The sounds are not part of this repository

The examples and songs load recordings from an `assets/` folder: sample libraries, impulse
responses, wavetables, MIDI files and the spoken clips. They are third party material and are not
published here, so a program that needs them stops without them. Everything that is
synthesized (most synths, effects and the tests that do not read files) works without them. To
use your own sounds, put them where the example expects them (see the `assets/...` paths in the
code and in [docs/examples.md](docs/examples.md)), for instance:

```
mkdir assets      # then add your own files
```

## Tests

```
go test ./...
```

Without the assets most tests skip what they cannot load, but a few fail (the drum machine tests
in `modules/controls/panel/patcher/catalog` need its samples). A few tests drive a headless
Chrome.

## License

[MIT](LICENSE). The CLAP and VST3 headers under `modules/clap/include` and
`modules/vst3/include` keep their own licenses (see the `LICENSE` files beside them).
