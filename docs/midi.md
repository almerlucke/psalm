# MIDI

Psalm plays from MIDI live: a keyboard, a sequencer, a controller, or MIDI generated in your own
code. It has two parts:

- `midi`: the messages, a parser and the inputs. The inputs are a raw device, a JACK MIDI port, a
  queue you push to, and several of them merged.
- `modules/controls/keyboard`: a control module that plays synths from an input. It sends notes
  with a velocity curve, the pedals, controllers, pitch bend and pressure. It follows a MIDI clock
  and passes every event to your own handlers.

MIDI files play the same way. A `midi.Player` is an input that plays a file in time, so a
keyboard reading it plays synths from the file as it would from a keyboard (see
[MIDI files](#midi-files)).

```go
d, err := midi.OpenDevice("") // the first keyboard
if err != nil {
	log.Fatal(err)
}
defer d.Close()

grand := piano.NewPoly(piano.GrandPreset(), 24).AddTo(p)
keyboard.New(d, keyboard.Config{Curve: keyboard.Soft}, grand).AddTo(p)
```

## Messages

`midi.Message` is one complete message, status byte first. Its `Kind()` is one of these:

| Group | Kinds |
|---|---|
| Channel | `NoteOn`, `NoteOff`, `PolyPressure`, `ControlChange`, `ProgramChange`, `ChannelPressure`, `PitchBend` |
| System common | `SysEx`, `TimeCode`, `SongPosition`, `SongSelect`, `TuneRequest` |
| Real time | `Clock`, `Start`, `Continue`, `Stop`, `ActiveSensing`, `Reset` |

A note on with velocity 0 is a `NoteOff`.

Accessors read the data:

- `Channel()`: 1-16, or 0 for a system message.
- `Note()` and `Velocity()`.
- `Controller()` and `Value()`.
- `Program()` and `Pressure()`.
- `Bend()`: 0-16383. `BendAmount()` gives -1 to 1.
- `SongPosition()`.

`String()` prints a message readably.

Constructors build messages: `NoteOnMsg`, `NoteOffMsg`, `ControlChangeMsg`, `ProgramChangeMsg`,
`ChannelPressureMsg`, `PolyPressureMsg`, `PitchBendMsg`, and `ClockMsg`, `StartMsg`, `ContinueMsg`
and `StopMsg`. Constants such as `CCBankSelect`, `CCSustain`, `CCSostenuto`, `CCSoft`, `CCModWheel`,
`CCExpression` and `CCAllNotesOff` name the common controllers.

`midi.Parser` turns a byte stream into messages:

- It follows running status.
- It takes real time messages out wherever they come, even in the middle of another message.
- It collects system exclusive messages up to F7. `MaxSysEx` sets the longest one kept (64 KiB by
  default).

## Inputs

An input hands over the events of each audio buffer:

```go
type Input interface {
	Read(events []Event, frames int, sampleRate float64) []Event
}
```

Each `Event` has a `Message` and an `Offset`, the frame of the buffer it falls on. The keyboard
module passes that offset to the synth as the `SampleOffset` of the trigger, so notes start on the
right sample.

| Input | What it is |
|---|---|
| `midi.OpenDevice(path)` | A raw MIDI device: `/dev/snd/midiC<card>D<device>` on Linux, which is how a USB keyboard shows up. It works with the portaudio engine and needs no extra libraries. An empty path opens the first device that isn't virtual. `midi.Devices()` lists the devices with their names. `Send` writes to the device (MIDI out). `OnMessage` is called for every message as it arrives. |
| `engine.MidiIn(name)` | A JACK MIDI input port on the JACK engine (`audio/jack`). Register it before `Start`. JACK gives every event its exact frame, so timing is sample accurate. |
| `midi.NewQueue()` | Push messages from anywhere: another goroutine, the network, or generated input. `PushOffset` places a message at a frame of the next buffer. |
| `midi.Inputs{a, b}` | Several inputs merged into one, in order of offset. |
| `midi.NewPlayer(file)` | A MIDI file, played in time (see [MIDI files](#midi-files)). |
| `midi.Tee(input, n)` | One input shared by `n` readers, such as a keyboard for every channel of a file. Each reader gets every event. |
| `midi.Record(input)` | Passes another input on and, between `Start` and `Stop`, keeps its channel messages with the time they played. `File(bpm)` turns them into a MIDI file at a tempo: the silence before the first note is left out, notes still held are let go at the end. |

A device or a queue stamps each message with the time it arrives, and plays it a constant latency
later, at the frame that latency puts it on. The timing of the playing is kept: a quick run of
notes doesn't land all on the first sample of a buffer.

Audio callbacks don't always come evenly. PipeWire, for example, often asks for a few buffers
right after each other and then waits. The queue follows when each buffer is read against how much
audio it holds. It keeps the latency just long enough to cover that pattern, which is one buffer
when the callbacks come evenly, and a message can wait for a later buffer.

Choose a small buffer for playing live: the keyboard example uses 256 frames, about 6 ms.

## Keyboard

`keyboard.New(input, cfg, targets...)` plays its targets. A target is any receiver of trigger
messages, such as a poly synth or the piano.

- A note on becomes a trigger message with a `noteOn` id, and a note off a trigger with the same
  `noteOff` id.
- The note message holds the `pitch` (transposed) and the `velocity` (after the curve).
- A key struck again while it still sounds is released first. The piano then strikes the same
  strings again.

| Config | |
|---|---|
| `Channel` | The channel to listen to (1-16). 0 listens to all of them. |
| `Transpose` | Semitones. |
| `Curve` | The velocity curve: the velocity (0-1) is raised to this power. `Soft` (0.6) plays loud with a light touch, `Linear` (1, the default) leaves it unchanged, and `Hard` (1.6) needs more force. Any exponent works. Every keyboard responds differently, so choose the one that suits yours. |
| `VelocityMap` | A curve of your own, from 0-1 to 0-1. It replaces `Curve`. |
| `FixedVelocity` | When above 0, every note plays at this velocity (for an organ, or a keyboard without velocity). |
| `Dynamics` | How much the velocity sets the trigger amplitude. At 0 the amplitude is always 1, and the synth reads `velocity` from the note: use this for the piano and dx, which have their own response. At 1 the amplitude is the velocity: use this for synths that only scale by amplitude. |
| `Controls` | Maps controllers to messages for every voice, as `{name: value}` with the value 0-1. Nil means `DefaultControls`: 64 to `sustain`, 66 to `sostenuto` and 67 to `soft`, the piano's pedals. Because sustain is sent as a continuous value, half pedaling works. An empty map sends nothing. |
| `HoldNotes` | The keyboard holds released notes itself while the sustain pedal is down, instead of sending the pedal. Use this for synths without a sustain pedal of their own. |
| `PitchBend`, `Pressure` | The message names for pitch bend (-1 to 1) and channel pressure (0-1), sent to every voice. Empty sends none. |
| `PolyPressure` | The message name for the pressure of one key (polyphonic aftertouch, 0-1), sent to the voice of that note alone (a poly's `"command": "note"`). Empty sends none. |
| `Routes` | Sends controllers to any setting of a synth, scaled to its range (see below). A routed controller is sent as its routes instead of by `Controls` (`PitchBend`, `Pressure`). |
| `FollowClock` | Sets the tempo of the patch from a MIDI clock, averaged over the last beat. `Tempo()` reads it. |
| `Hits` | Plays notes as hits that play to their end: a note on becomes a note of no duration, and a note off sends nothing. Use this for drums played from a file or a pad, whose short note offs would otherwise choke the drumsynth. |

**Routes** send a controller to any setting the synth takes by message, scaled to that setting's
range: the mod wheel sweeping a cutoff from 300 to 8000 Hz, a slider moving the vowel, the
aftertouch folding a wave. One controller can move several settings. The keys `RouteBend` and
`RoutePressure` route the pitch bend (its middle the middle of the range) and the aftertouch.

```go
kb := keyboard.New(input, keyboard.Config{Routes: map[int][]keyboard.Route{
	midi.CCModWheel: {{Name: "cutoff", Min: 300, Max: 8000, Exp: true}, {Name: "resonance", Min: 0.1, Max: 0.6}},
	keyboard.RoutePressure: {{Name: "fold", Min: 0, Max: 1}},
}}, synth)
kb.SetRoutes(74, keyboard.Route{Name: "vowel", Min: 0, Max: 4}) // or later, or by message
```

A route sets the setting (the wheel at half sets the middle of the range) where the synth's own
`modWheel` adds on top of the preset. `Exp` goes exponentially (frequencies, times), `Int` in whole
numbers (a waveform, a mode), and `Max` under `Min` turns it round. As text, for a flag or a
message: `keyboard.ParseRoute("1:cutoff:300:8000:exp")` (the source a controller, `bend` or
`pressure`); the messages `{"route": "1:cutoff:300:8000:exp"}` and `{"unroute": "1"}` change the
routes of a keyboard while it plays. The settings each synth takes are listed under **Messages** in
[synths](synths.md); `cmd/examples/keyboard -route ...` tries them, and its panel has a Routes section to choose where the mod wheel and the aftertouch go while playing.

The keyboard always handles these messages:

- All notes off (123) and all sound off (120) release the notes of their channel.
- Reset all controllers (121) puts the mapped controls, the routes, the bend and the pressure back to rest.
- A system reset releases everything.

`AllNotesOff(tick)` and `SetTargets(...)` let you switch instruments while playing.

For everything else, add handlers. They are called on the audio thread with the tick of the event,
so they can send messages with sample accurate timing:

```go
kb.OnEvent(func(e midi.Event, tick psalm.CtrlTick) { log.Println(e.Message) })  // every event
kb.OnControl(midi.CCModWheel, func(v float64, tick psalm.CtrlTick) {             // a controller, 0-1
	chorus.ReceiveMessage(psalm.Message{"mix": v})
})
kb.OnProgram(func(program int, tick psalm.CtrlTick) { /* switch the sound */ })
```

The keyboard reads its input on the first control tick of every buffer, so a tick has to be
shorter than a buffer. It is with the default 240 PPQ. A note off takes effect at the start of the
buffer it falls in, because the synths release a note immediately.

The keyboard also takes messages to change its settings: `channel`, `transpose`, `curve`,
`fixedVelocity` and `dynamics`.

## MIDI files

`midi.ReadFile(path)` reads a Standard MIDI File (`.mid`, or `.rmi`). It handles format 0 and 1,
ticks per quarter note or SMPTE time, running status, tempo changes, meta events and system
exclusive. It takes files as they come: a track cut short ends where the file does, and other
chunks are skipped. `ParseFile` reads from bytes, and `Write` and `WriteFile` write a file back.
Build one in code with `Track.Add(tick, msg)` and `Track.AddMeta(tick, midi.MetaTempo,
midi.TempoData(96))`.

| File | |
|---|---|
| `Format`, `Division` | The format (0 one track, 1 tracks played together) and the ticks per quarter note (`FPS` and `TicksPerFrame` for SMPTE time). |
| `Tracks` | The tracks. Each `Track` has `Events` in order of tick, and `Name()`. An event holds a `Message` (a channel or system exclusive message) or a meta event (`Meta`, `Data`; `Tempo()`, `TimeSignature()`, `Text()`). |
| `Seconds(tick)`, `Duration()` | Time in seconds, following the tempo changes. |
| `Channels()`, `Programs(ch)` | The channels that play, and the programs each one plays (from its program changes) in order. `midi.ProgramName(p)` gives the General MIDI name. |

`midi.NewPlayer(file)` plays a file as an input. Each message lands on the exact frame its time
falls on, following the tempo map. A file of many channels plays a synth for each channel: give
every channel a keyboard with its `Channel` set, all reading one player through `midi.Tee`.

```go
f, err := midi.ReadFile("song.mid")
if err != nil {
	log.Fatal(err)
}
player := midi.NewPlayer(f)
ins := midi.Tee(player, 2)
grand := piano.NewPoly(piano.GrandPreset(), 32).AddTo(p)
drums := drumsynth.NewPoly(drumsynth.Config{Kit: drumsynth.Kit909()}, 16).AddTo(p)
keyboard.New(ins[0], keyboard.Config{Channel: 1}, grand).AddTo(p)
keyboard.New(ins[1], keyboard.Config{Channel: 10, Hits: true}, drums).AddTo(p)
player.Play()
```

| Player | |
|---|---|
| `Play()`, `Stop()`, `Playing()` | Play from where it is (from the start once at the end). Stop where it is; the notes sounding are released and the sustain pedal comes up. |
| `Seek(seconds)`, `Position()`, `Duration()` | Jump to a time. Every channel is then set as the file has it there: the controllers are reset, then the bank, the program, the controllers, the bend and the pressure are sent as they last were. A note started before that point doesn't sound. |
| `SetLoop(bool)` | Start over at the end. |
| `SetSpeed(x)` | 1 plays at the file's tempo, 0.5 at half of it (0.05 to 8). |
| `SetTempo(bpm)`, `FollowTempo(func() float64)` | Play every beat at a tempo instead of the file's, so it stays in step with whatever keeps that tempo: a fixed one, or one asked for every buffer, such as the patch's tempo (which a keyboard with `FollowClock` sets from a MIDI clock). The file's own tempo changes are left out. 0 or nil plays it at its own tempo again. `Tempo()` is the file's tempo where it is. |
| `PlayTime(from)` | The seconds it takes to play from a time to the end, at the speed and tempo now (how long to render). |
| `MuteChannel(ch, bool)`, `MuteTrack(i, bool)` | Mute the notes of a channel or a track. Its controllers still go through, so it plays correctly when unmuted. |

All of these can be called from any goroutine, while it plays.

## Example

`cmd/examples/keyboard` plays the synths live from a MIDI keyboard, in the Musikverein: the piano,
subtractive, pulsar, wavetable, fm, dx and grains, each with all its presets:

```
go run ./cmd/examples/keyboard -list                    # the MIDI devices
go run ./cmd/examples/keyboard                          # the first keyboard, through portaudio
go run ./cmd/examples/keyboard -device /dev/snd/midiC1D0 -preset felt -curve soft
go run ./cmd/examples/keyboard -synth subtractive -preset brass
go run ./cmd/examples/keyboard -curve 1.3 -transpose -12 -monitor
go run ./cmd/examples/keyboard -jack                    # JACK audio and a JACK MIDI port
go run ./cmd/examples/keyboard -demo                    # no keyboard: a phrase played as MIDI
```

Bank select (controller 0) chooses the synth the next program change switches to, in the order
above, and the program change chooses its preset, in alphabetical order. Typed in the terminal, a
synth name, a preset name or both (`wavetable glass`) switch too, and `list` shows them all. The
piano takes the velocity itself and has its pedals, fm and dx take it too; the other synths play
the velocity as the amplitude, and the sustain pedal holds their notes (`HoldNotes`).
`-monitor` prints every event.

A control panel runs alongside on a web page, `http://localhost:8087` (`-addr` to move it, empty to
leave it out): the sound (every synth and preset, previous and next, all notes off), the
transposition, the velocity curve, the volume and the hall, and meters. A switch from the keyboard
or the terminal moves the panel along. Press MIDI learn on the page, tap a control and turn a knob
on the keyboard: from then on the knob moves it; the bindings are kept in `keyboard-midi.json`
(`-bindings`). See [panel](controls.md#panel).

`cmd/examples/midifile` plays a MIDI file on the synths. Every channel gets a synth chosen by the
General MIDI program it plays, and channel 10 plays the drumsynth:

```
go run ./cmd/examples/midifile song.mid
go run ./cmd/examples/midifile -list song.mid           # the tracks, the channels and their synths
go run ./cmd/examples/midifile -synth 1=supersaw/lead -kit 909 -loop song.mid
go run ./cmd/examples/midifile -render song.wav song.mid
go run ./cmd/examples/midifile -orchestra assets/Virtual-Playing-Orchestra3 song.mid
go run ./cmd/examples/midifile -tempo 128 song.mid                 # every beat at 128
go run ./cmd/examples/midifile -clock /dev/snd/midiC3D0 song.mid   # in step with a sequencer
```

With `-orchestra` the strings, brass, woodwinds, choirs, harp, timpani and mallets play the samples
of the Virtual Playing Orchestra. On the panel every channel can play any other sound while it
plays: a synth and preset, a drum kit or an orchestra instrument. `cmd/examples/keyboard -record`
(or Record on its panel) records what you play to a MIDI file, to play again with it.

See [examples](examples.md).
