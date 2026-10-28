# organ

`synths/organ`

A tonewheel organ: 91 wheels spinning all the time, which the keys tap at the pitches of the nine
drawbars (16' 5 1/3' 8' 4' 2 2/3' 2' 1 3/5' 1 1/3' 1', 3 dB a step). Like the real one it is a
whole instrument, not a set of voices: keys sharing a wheel share its phase, the wheels fold back
at the ends, the percussion sounds on the first key of a phrase only (single trigger) and the
vibrato scanner moves the whole sound. Put the rotary speaker after it.

```go
org := organ.New(organ.Jazz()).AddTo(p)
leslie := rotary.New(rotary.DefaultConfig()).AddTo(p).In(org)
org.Receive(psalm.Message{"drawbars": "888000000"}, 0)
```

It takes the messages of a poly synth (triggers with a duration, or noteOn and noteOff), and the
sustain pedal.

| Setting | Meaning |
|---|---|
| `Drawbars` | `[9]float64`, 0-8; `organ.Bars("888000000")` reads the way organists write it |
| `Percussion` | `PercOff`, `PercSecond` (4'), `PercThird` (2 2/3'); with it on the 1' is silent |
| `PercFast`, `PercSoft` | A short decay, 6 dB softer |
| `Vibrato` | `VibratoOff`, `V1`-`V3`, `C1`-`C3` (chorus: mixed with the dry sound) |
| `VibratoHz` | The scanner's speed (0 is 6.9 Hz) |
| `Click` | 0-1, the key contacts |
| `Drive` | 0-1, the preamp saturating |

**Value inputs**: 1-9 the drawbars. **Messages**: `drawbars`, `drawbar1`-`drawbar9`,
`percussion`, `percFast`, `percSoft`, `vibrato`, `vibratoHz`, `click`, `drive`, `level`,
`sustain`.

| Preset | Character |
|---|---|
| `jazz` | 888000000, third percussion fast |
| `gospel` | 888800000, deep chorus |
| `full` | Every drawbar out |
| `rock` | 888000000 driven, chorus, more click |
| `ballad` | 806004000, soft, chorus |
| `flute` | 008400000, vibrato |
| `reed` | 800000888, nasal |
| `whistle` | 800000008 |
| `screamer` | Every drawbar into a preamp driven to its limit, all the click, deep chorus |
| `warble` | The scanner at 15 Hz: a fluttering, alien warble |
| `chime` | Only the high drawbars with the fast third percussion: a glassy stab |
| `growlBass` | 16' and 8' driven, all click: a growling bass organ |
| `seasick` | The scanner at 1.5 Hz, deep: everything sways out of tune and back |

## Filter, leakage, matrix

The organ has no voices, so the new parts act on the summed sound. The [filter](../../synths.md#the-voice-filter)
(`filterType`, `cutoff`, `resonance`, `filterEnvAmount`, `keyTrack`, `filterDrive` and the `filter`
envelope) is a single filter after the drawbars; its envelope starts with the first key of a phrase and
releases when the last key lets go, like the percussion's single trigger. `leakage` (0-1) adds the
faint crosstalk of three other tonewheels to every key, different on each key press. `clickRand` (0-1)
makes the key click vary in loudness and colour from key to key.

The 8 slot [modulation matrix](../subtractive.md#modulation-matrix) takes the sources filter env, amp (the loudest key),
LFO 1 (the scanner's), velocity and key of the last note, a random value per phrase, the wheel and Ext
1-4 (value inputs 10-13, `organ.InExt1`) to `drive`, `click`, `vibratoDepth`, `level`, `cutoff`,
`resonance`, `percLevel`, `leakage` and `pitch`. There is no LFO 2. The rotary of a Leslie is the
`rotary` effect after the organ.

The matrix has a second LFO: `lfo2Rate` (Hz, default 1) is a plain sine, available as the source LFO 2
next to the scanner's, running only when a slot uses it.

## The tonewheel generator, scanner and Leslie

The organ models the Hammond tonewheel generator. The 91 wheels turn at the frequencies of the gear
ratios (a 20 Hz shaft, twelve gear pairs, exact octaves), so the semitones are slightly off equal
temperament, and each wheel is a sine with a little second and third harmonic (`tonewheel` 0-1; the gear
table is from memory: check it against Hammond documentation if exact tuning matters). Each key has nine
contacts, one per drawbar bus, that close a few milliseconds apart (`contactSpread`), so the attack is
percussive; `click` sets how sharply they close. `leakage` bleeds the neighbouring wheels and the
wheels an octave and a twelfth up into the bus, at -40 dB or less.

The scanner is a nine-section line swept at 6.87 Hz: V1/V2/V3 use 3/6/9 sections (about +-4, +-8 and
+-12 cents), the C modes mix the dry sound with it, and `vibratoScale` makes it deeper. A tube preamp has
two asymmetric stages (`drive`) and a tone tilt (`tone`). `leslie` (0-1) puts the sound through a
built-in rotary speaker: an 800 Hz Linkwitz-Riley split, a horn and a drum that each have Doppler and
amplitude modulation, and two microphones (`lesliePosition`). `leslieSpeed` runs from 0 (stopped)
through 0.5 (slow, 0.8 Hz horn) to 1 (fast, 6.7 Hz), and the rotors speed up and slow down at their own
pace (the horn in about 1.2 s, the drum in about 3 s). `leslie`, `leslieSpeed` and `tone` are matrix
destinations.
