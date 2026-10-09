# Core concepts

## Psalm and the patch

`psalm.New(engine)` creates the top level patch. It has a stereo input (the audio engine's input)
and a stereo output. Every module you use must be added to it with `AddTo(p)`, so it receives
control ticks and is run.

```go
p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
defer p.Close()

synth := subtractive.NewPoly(subtractive.LeadPreset(), 4).AddTo(p)
p.Out(synth)          // route the synth to the patch output

_ = p.Listen()        // play live until SIGINT / SIGTERM
```

Passing `nil` as engine creates a patch without audio device, which is all you need for offline
rendering.

### Running live or rendering

| Method | What it does |
|---|---|
| `p.Listen()` | Starts the audio engine and blocks until Ctrl+C, then stops it |
| `p.AudioStart()` / `p.AudioStop()` | Start and stop the engine yourself |
| `p.ToSoundFile(path, format, seconds, sampleRate, normalize)` | Renders the patch offline to a file (`writer.WAV` etc. from `github.com/almerlucke/sndfile/writer`), resampling when `sampleRate` differs from the patch, optionally normalizing |
| `p.Close()` | Closes the audio engine |

## Config and environment

`psalm.PushConfig` sets the configuration for the patches you create next (a stack:
`psalm.PopConfig` restores the previous one), so push it before `psalm.New`.

| Field | Default | Meaning |
|---|---|---|
| `SampleRate` | 44100 | Samples per second |
| `BufferSize` | 1024 | Samples processed per cycle |
| `BPM` | 120 | Tempo in beats per minute (a `float64`, fractions like 87.5 are fine) |
| `PPQ` | 240 | Control ticks per quarter note (the resolution of sequencing) |

Every `psalm.New` creates an **environment** (`psalm.Env`) for its patch: a copy of the config,
the **live tempo** and the **random streams** of that patch. It becomes current, and every module
created after it belongs to it, so create a patch's modules after its `psalm.New`. A module reads
its environment, not global state, while it plays:

| Module method | |
|---|---|
| `m.Config()` | The config when the module was created (sample rate, buffer size, PPQ) |
| `m.BPM()` | The live tempo of its patch |
| `m.Env()` | Its environment: `SetBPM(bpm)` changes the tempo of that patch only, `TickDuration()` |
| `m.Rand()` | Its own random stream, from the patch's seed |

So patches don't share anything while they play: one patch can change its tempo without
touching another, and several patches can render in parallel (build them one after the other,
then render each on its own goroutine).

`psalm.Seed(n)` seeds the patches created after it, so a patch built and rendered after the same
seed sounds the same, sample for sample; seeding the next patch never changes one already built.
Without a seed every run is different. See [rendering](rendering.md).

The package functions `psalm.SampleRate()`, `psalm.BufferSize()`, `psalm.BPM()`, `psalm.PPQ()`,
`psalm.TickDuration()` and `psalm.ControlRate()` read the current environment: use them while
building a patch, and the module methods inside modules.

## Modules

Everything is a `psalm.Modular`. Modules have numbered audio input and output sockets (each with
a number of channels: `psalm.Mono` or `psalm.Stereo`) and can send and receive values.

Useful methods on every module:

| Method | Meaning |
|---|---|
| `AddTo(p)` | Adds the module to a patch (required) |
| `In(...)` | Connects audio outputs of other modules to this module's inputs |
| `ValueIn(...)` | Connects value outputs of other modules to this module's value inputs |
| `Disconnect(out, to, in)` / `ValueDisconnect(out, to, in)` | Takes an audio or value connection away |
| `DisconnectAll()` | Takes every connection of the module away, in and out, audio and values |
| `SetID(id)` / `ID()` | Name the module (also used by `poly` for note off) |
| `SetMuted(bool)` | Muted modules still run but output silence |
| `Receive(value, index)` | Receive a plain value on a value input |
| `ReceiveMessage(msg)` | Receive a message (`psalm.Message`, a `map[string]any`) |

Control modules (timers, templates, ...) are created with `psalm.NewControl()` and have no
audio sockets, only value connections.

## Connections

`In` and `ValueIn` take a list of modules, each optionally followed by an output index and an
input index:

```go
// module, [outIndex, [inIndex]], module, [outIndex, [inIndex]], ...
reverb.In(synth)                    // synth output 0 -> reverb input 0
vocoder.In(speech, 0, 0, pad, 0, 1) // speech -> input 0, pad -> input 1
mixer.In(a, 0, 0, b, 0, 0)          // both into input 0: audio connections to one input are summed
filter.ValueIn(lfo, 0, 1)           // lfo value output 0 -> filter value input 1
```

When the indices are left out, the output index is 0 and the input index counts up per module
(the first module goes to input 0, the second to input 1, ...). So to mix several modules into
one input, always give `0, 0` explicitly.

`p.Out(...)` connects modules to the patch output the same way.

### Changing a patch while it plays

Connections can be taken away again, and modules taken out of a patch:

```go
synth.Disconnect(0, reverb, 0)      // synth output 0 no longer into reverb input 0
lfo.ValueDisconnect(0, filter, 1)   // the value connection
module.DisconnectAll()              // everything in and out, audio and values
p.Remove(module)                    // out of the patch: disconnected, no more control ticks
```

`Remove` works on a patch within the patch (a `psalm.Patch`) too: it keeps its own modules and
connections, so it can be added again later (`sub.AddTo(p)` and connect it), or to another patch.
A module that is removed stops at once: a sound it was making is cut off.

While a patch plays, the audio thread walks its modules and connections every buffer, so change
them there: `p.Do(f)` runs `f` between two buffers (at once while the patch is stopped or renders
offline). Everything one `f` does happens in the same pause, so the patch never plays half a
change:

```go
p.Do(func() {
	p.Remove(oldLead)
	newLead.AddTo(p)
	reverb.In(newLead)
})
```

`Do` can be called from any goroutine (a web page, a MIDI handler, a timer); keep the changes
short, the audio waits for them. `p.Restart()` sets the patch back to its start (see
[controls](controls.md#panel)).

A patch can also play within another one with a clock and a tempo of its own: `p.NewSub()`, see
[playlist](playlist.md#patches-within-a-patch).

## Values and messages

Value connections carry any Go value at control time (not audio):

- **Plain values** (floats, ints, ...) on a numbered value input, for example an LFO driving a
  cutoff. Use `psalm.FltVal`, `psalm.IntVal`, `psalm.BoolVal`, `psalm.StrVal` to convert.
- **Messages**: a `psalm.Message` (`map[string]any`) with named settings. Most modules accept
  messages to change any setting by name, see the module pages.
- **Bang**: `psalm.Bang` (or a message with a `"bang"` key) triggers things, like a timer
  telling a template to generate.

`psalm.Message` has typed getters (`FltVal`, `IntVal`, `StrVal`, `BoolVal`, `TickVal`, `Msg`),
`Must...` variants that panic and `If...` variants that call a function only when the key exists.

## Control ticks

Psalm advances control time in ticks (`PPQ` ticks per quarter note). Every module gets
`CtrlTick(tick)` for every tick, with:

| Field | Meaning |
|---|---|
| `SampleTimestamp` | Absolute sample time |
| `TickTimestamp` | Absolute tick count |
| `SampleOffset` | Position of the tick inside the current buffer, used for sample accurate note starts |

Timers use ticks to schedule events, and pass the tick along in their messages so synths can
start notes at the exact sample.

## Audio engines

| Engine | Package | Notes |
|---|---|---|
| PortAudio | `audio/portaudio` | `portaudio.New(sampleRate, bufferSize)`, default input/output device |
| JACK | `audio/jack` | `jack.New(clientName)`, sample rate and buffer size come from the JACK server |

Both implement `audio.Engine` (`GetSampleRate`, `GetBufferSize`, `SetCallback`, `Init`, `Start`,
`Stop`, `Close`). The patch pushes a config with the engine's sample rate and buffer size.
