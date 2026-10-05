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

## Config

The current configuration is a stack: `psalm.PushConfig` makes a new config current,
`psalm.PopConfig` restores the previous one. Modules read the config that is current when they
are created, so push it before building the patch.

| Field | Default | Meaning |
|---|---|---|
| `SampleRate` | 44100 | Samples per second |
| `BufferSize` | 1024 | Samples processed per cycle |
| `BPM` | 120 | Tempo |
| `PPQ` | 240 | Control ticks per quarter note (the resolution of sequencing) |

Helpers: `psalm.SampleRate()`, `psalm.BufferSize()`, `psalm.BPM()`, `psalm.PPQ()`,
`psalm.TickDuration()` (seconds per tick) and `psalm.ControlRate()` (ticks per second).

## Modules

Everything is a `psalm.Modular`. Modules have numbered audio input and output sockets (each with
a number of channels: `psalm.Mono` or `psalm.Stereo`) and can send and receive values.

Useful methods on every module:

| Method | Meaning |
|---|---|
| `AddTo(p)` | Adds the module to a patch (required) |
| `In(...)` | Connects audio outputs of other modules to this module's inputs |
| `ValueIn(...)` | Connects value outputs of other modules to this module's value inputs |
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
