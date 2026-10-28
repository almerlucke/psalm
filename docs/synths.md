# Synths

Synths (the presets of each are listed in [presets](presets.md)): [pitch bend, mod wheel and aftertouch](#pitch-bend-mod-wheel-and-aftertouch), [envelopes by message](#envelopes-by-message), [poly](#poly) (voice allocation, mono and legato), [subtractive](synths/subtractive.md), [dx](synths/dx.md) (6 operator FM, SysEx, and the 4 operator FM presets), [grains](synths/grains.md) (granular sampler), [pulsar](synths/pulsar.md), [wavetable](synths/wavetable.md), [piano](synths/piano.md), [drums](synths/drums.md), [sampler](synths/sampler.md) (SFZ), [organ](synths/organ.md), [waveguide](synths/waveguide.md) (plucked and bowed strings), [modal](synths/modal.md) (mallets and bells), [drumsynth](synths/drumsynth.md), [vocal](synths/vocal.md) (voices and choirs), [shaper](synths/shaper.md) (waves from chains of shapers, morphing), [supersaw](synths/supersaw.md).

Every synth describes its settings: `<synth>.Describe(cfg)` (`params`: each message key it takes,
its range, unit and group, where it is in the config, and its envelopes), so an editor on the panel
(`modules/controls/panel/editor`) can draw its controls, a sequencer pick what to step, a randomiser
what to move. The drumsynth's sounds and the sampler's regions are set per sound, not by message:
they describe only their level.

Presets of all synths are calibrated to the same loudness: a note at amplitude 1 plays at -12 LUFS
(or peaks at -1 dBFS for percussive sounds), see [conventions](conventions.md#levels).

## Pitch bend, mod wheel and aftertouch

Every synth but the piano (as a real one) takes the hands of a player while it sounds: the
messages `pitchBend` (-1 to 1, over `bendRange` semitones, 2 by default), and `modWheel` (0 to 1),
which does something of its own on each, and `pressure` (0 to 1): the aftertouch, pressing harder
on a held key, which adds to the wheel (the sampler leaves it to its instrument). They glide to
where they are moved in a few milliseconds, so moving them never zips or clicks, and reach every
voice of a poly synth (a message without a command). A keyboard sends them with
`keyboard.Config{PitchBend: "pitchBend", Pressure: "pressure", Controls:
map[int]string{midi.CCModWheel: "modWheel"}}`.

| Synth | The mod wheel |
|---|---|
| subtractive | opens the filter, two octaves |
| dx | a vibrato (as on a DX7) |
| wavetable | moves through the table, opens the filter |
| pulsar | lifts the formant |
| grains | sprays the grains wider (the bend reaches new grains) |
| sampler | controller 1, as the instrument says (crossfades, `gain_cc1`); the bend follows `bend_up`, `bend_down` |
| organ | the scanner vibrato, up to V3 |
| waveguide | a vibrato (a plucked string bends while it rings) |
| modal | a tremolo |
| drumsynth | longer decays (new hits); the bend moves the sounds ringing |
| vocal | opens the vowel |
| shaper | opens the filter (when it is on); a slot of the matrix can send it to the morph or a macro |
| supersaw | opens the filter, three octaves |

The code of it is `blocks/expression`, for synths of your own.

## Envelopes by message

Every synth with envelopes takes them by message the same way (`blocks/envelope`), so a panel, an
envelope widget, an automation lane or a sequence can shape them while it plays:
`<name>Attack`, `<name>Decay` and `<name>Release` in milliseconds, `<name>Sustain` the level held
while the key is down (0-1). A change takes effect from the next note.

| Synth | Envelopes |
|---|---|
| subtractive, supersaw | `amp`, `filter` |
| wavetable | `amp`, `filter`, `position` |
| pulsar | `amp`, `formant` |
| shaper | `amp`, `morphEnv` (the morph), `shapeEnv` (macro 1), `filter` |
| dx (exact operators) | `amp` (the carriers: the operators heard), `op1` to `op6` (each operator) |
| grains | `amp` |
| sampler | `amp`: over the envelopes of the regions (each part set replaces theirs) |
| vocal | `ampAttack`, `ampRelease` |
| modal | `ampDecay` (how long it rings), `ampRelease` (damped once let go) |
| waveguide | `ampAttack` (the bow), `ampDecay` (a plucked string's ring), `ampRelease` (let go) |

The piano, the organ and the drumsynth have no envelopes of this kind (the piano is its hammers
and dampers, the organ is on while a key is down, a drum is its decay); dx keeps the rates and
levels of the DX7.

```go
synth.ReceiveMessage(psalm.Message{"ampAttack": 400.0, "ampRelease": 2500.0, "filterSustain": 0.2})
```

## The voice filter

`blocks/voicefilter`

A filter after the oscillators of a voice, the same on the synths that have no filter of their own
(dx, and more as they get it): state variable (low, band, high, notch) or a ladder (24 or 12 dB),
with its own envelope, key tracking and drive. It is off until its type is set.

| Message | |
|---|---|
| `filterType` | 0 off, 1 low pass, 2 band pass, 3 high pass, 4 notch, 5 ladder 24, 6 ladder 12 |
| `cutoff`, `resonance` | Hz, 0-1 |
| `filterEnvAmount` | octaves the envelope opens the cutoff |
| `keyTrack` | 0-1, the cutoff follows the note |
| `filterDrive` | ladder input saturation |
| `filterAttack`, `filterDecay`, `filterSustain`, `filterRelease` | the envelope (ms, sustain 0-1) |

In a config it is `Config.Filter` (a `voicefilter.Config`): `cfg.Filter = voicefilter.Config{Type:
voicefilter.LowPass, Cutoff: 1200, Resonance: 0.3, EnvAmount: 2}`. The dx synth keeps it in the patch,
the way the DX7's own settings are, 0 to 99: `filterType`, `filterCutoff`, `filterResonance`,
`filterEnvAmount`, `filterKeyTrack`, `filterAttack`, `filterDecay`, `filterSustain`, `filterRelease`
(`Patch.Filter`, not part of the SysEx data), with a row in the dx editor.

## poly

`modules/poly`

Polyphonic voice allocation for any voice type. `poly.New(numChannels, voices)` mixes the voices
to one output and assigns notes to free voices. When all voices are busy it steals the oldest
one, with a short fade out so stealing doesn't click.

A voice implements `poly.Voice`:

```go
type Voice interface {
	psalm.Modular
	NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, config psalm.Config)
	NoteOff()
	Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, config psalm.Config)
	Clear()
	IsActive() bool
}
```

**Messages** (usually built with `templ.NewTrigger` / `templ.NewDurationTrigger`, see [controls](controls.md#templ)):

| Message | Effect |
|---|---|
| `{"command": "trigger", "tick", "amplitude", "message", "duration"}` | Plays a note of `duration` seconds on a free voice, the voice receives `message` (pitch, settings, ...) |
| `{"command": "trigger", "tick", "amplitude", "message", "noteOn": id}` | Starts a held note, remembered under `id` |
| `{"command": "trigger", "tick", "noteOff": id}` | Releases the held note `id` |
| `{"command": "note", "id": id, "message": {...}}` | Passes `message` to the voice of the held note `id` alone (in mono, when it is the note sounding): the polyphonic aftertouch of one key |
| `{"command": "voice", ...}`, or a message without a command | Passes the whole message to all voices (change settings of every voice, `AutomateMessage` of an arrangement works this way) |

**Value inputs**: plain values on inputs above 0 are passed to all voices on the same input, so
an LFO connected to input 1 of a subtractive poly synth modulates the cutoff of every voice.

`AllNotesOff()` releases all held notes, `CallVoices(func(v))` runs a function on every voice.

**Mono**: `SetMono(true, legato)` (or the message `{"mono": 1, "legato": 1}`) plays one note at a
time on one voice. A new note takes over from the one held, and when it is let go the note still
held comes back. Legato changes the pitch of the sounding note without starting it again, so a
synth with a glide (subtractive, supersaw) slides between the notes; without legato every note
starts the voice again, from the pitch it was at. Notes of a duration (from a score) that overlap
slide as well. `SetMono(false, false)` makes it polyphonic again.

```go
lead := supersaw.NewPoly(supersaw.GlideLead(), 4).SetMono(true, true).AddTo(p)
```
