# piano

`synths/piano`

A physically modelled piano, no samples. Every key has one to three strings (one in the low bass,
two up to the tenor, three above), each a digital waveguide: a delay line closed by

- a **loss filter** that sets how long the string rings (about 20 seconds in the low bass, under 2
  at the top) and makes the highs die faster;
- a chain of **allpass filters** for the stiffness of the string: they stretch the partials sharp
  (inharmonicity), fitted per key to the stretch of a real grand (strongest at the top);
- a **tuning allpass** that puts the fundamental on pitch to a fraction of a cent.

What makes it a piano:

- **Two-stage decay.** Every string vibrates in two polarizations: one passes its energy to the
  soundboard fast (the loud prompt sound, the first second), the other keeps it (the long, quiet
  aftersound). The strings of a key are a little out of tune with each other (`Unison`) and share
  the bridge: what they do together drains into the soundboard, what they do against each other
  lasts, and the bridge passes energy between them and to the other polarization (Weinreich's
  coupled strings). So a note falls fast, then lingers and breathes as the strings trade energy.
- **A felt hammer that meets the string.** Every strike is simulated: the hammer (its mass, heavier
  in the bass) flies at the string, its felt stiffens steeply as it is compressed and pushes harder
  going in than coming out (hysteresis), the string gives way under it and the waves leaving the
  strike point come back from both ends. The force that results is the strike: a few milliseconds
  in the bass, a fraction in the treble, shorter and sharper the harder the key is played (a soft
  note is round, a loud one bright), and on some keys the hammer bounces and hits twice. The felt
  of every key is fitted once (when the piano is made) so velocity 0.8 sounds as voiced. The strike
  position leaves out the harmonics with a node where the hammer hits, and the soundboard knock is
  heard with it: a short impact, shorter in the treble (from about 80 ms in the bass to about 12 ms
  at the top), so it never outlasts the short high notes.
- **A string played hard settles.** Its own motion stretches it, so it starts a little sharp (up to
  about 5 cents in the bass at full velocity, a cent at the top) and settles within a few hundred
  milliseconds.
- **Irregular partials.** The partials of a real string don't die away evenly: the bridge takes
  more at some frequencies than at others. Every string has a slightly different pattern of
  partials that decay faster (up to 3 dB per second, with `Variation`).
- **A soundboard** every string sounds through: a board of spruce made of its modes (a few hundred:
  sparse in the bass, one every 20-odd Hz higher up, with the low Q of wood, radiating the bass
  weakly and rolling off the highs), heard from a different place on every side of the stereo image.
  The strings sound through it by convolution (`Body`), or through a measured soundboard response
  (`SoundboardIR`). Every key is voiced for the board, like a technician voicing the hammers: the
  gain that evens out where the key's partials land on the board's modes.
- **The bark of the bass.** A string played hard stretches as it moves, which rings its longitudinal
  mode and adds phantom partials at the sums of its partials: the metallic bark of loud bass notes,
  growing with the velocity (`Bark`).
- **The action** is heard: the dampers landing on the still vibrating strings, the key coming up,
  the sustain pedal lifting all dampers (a soft swish) and letting them land (a thud) (`Mechanics`).
- **Sympathetic resonance.** The strings of all 88 keys ring along with what is played when their
  dampers are up: all of them with the sustain pedal down, the keys held down (also silently), and
  the top keys, which have no dampers (`Resonance`). With the pedal down a halo blooms as the notes
  die away.
- **Dampers** stop the strings when a key is released (slower in the bass). The **sustain pedal**
  lifts them; between up and down it lifts them partly (**half pedaling**): the dampers touch the
  strings lightly, a released note rings on shorter, and the heavy bass strings push the light touch
  away more easily than the treble, so half pedaling clears the treble and lets the bass ring. The
  **sostenuto pedal** keeps the dampers of the keys held when it goes down up until it comes up:
  a bass note caught by it rings under chords played dry above it. The **soft pedal** (una corda)
  shifts the hammers so they strike one string less with a softer part of the felt: quieter and
  veiled.
- **Stretch tuning.** The partials of stiff strings are sharp, so the instrument is tuned like a
  tuner does it: octaves widened until they are clean (the 2nd partial of the lower note on the upper
  note in the treble, 4:2 and 6:3 lower down), which gives the Railsback curve: the low bass some 15
  cents flat, the top some 45 cents sharp, derived from the strings' own inharmonicity (`Stretch`).
- **No two notes alike.** Every key's hammer is a little harder or softer, hits a little elsewhere,
  every key rings a little longer or shorter and is a little louder or softer, as on any real
  instrument; every strike varies a little in force and felt, and the hammer reaches the strings of
  a key a fraction of a millisecond apart (`Variation`). The strikes draw from the random source of
  the patch, so `psalm.Seed` repeats them.
- **A listening position**: on the bench (`Player`), close microphones over the open lid
  (`Studio`), or in the audience (`Hall`): how wide the keys spread, the early reflections of the
  lid, the floor and the stage, and from the audience a little air (`Position`). A reverb adds the
  room after that.
- **Calibrated against a recorded concert grand** (single notes of the University of Iowa Steinway
  at pianissimo, mezzo forte and fortissimo): the decay of every register (a fast first second,
  roughly 6 to 18 dB per second, then a slow aftersound), the brightness of the attack at every
  dynamic (a near pure tone in the treble), the dynamic range (pianissimo to fortissimo some 25 dB
  in the middle, the level follows the speed of the hammer), the stretch tuning and the level over
  the keyboard (even up to C5, a few dB softer from C6, the top octave some 13-16 dB softer).

```go
grand := piano.NewPoly(piano.GrandPreset(), 24).AddTo(p) // a voice per sounding key

// the pedals: messages without a command reach the whole instrument
arrangement.AutomateMessage(arrange.Lane{Sections: map[string]arrange.Curve{"verse": arrange.Const(1)}}, grand, "sustain")
grand.ReceiveMessage(psalm.Message{"sustain": 0.5})   // half pedal
grand.ReceiveMessage(psalm.Message{"sostenuto": 1.0}) // catch the keys held now
grand.ReceiveMessage(psalm.Message{"soft": 1.0})
```

`piano.NewPoly(cfg, n)` creates the instrument (a `*piano.Piano`: `n` voices, the strings that ring
along and the soundboard), it takes the messages of a poly synth (triggers with a duration, or
`noteOn` and `noteOff`). `piano.New(cfg)` is a single voice, without soundboard and resonance. A
voice sounds until its strings have died away, the dampers stopped them, or as long as its key is
held; with the pedal down that can take many seconds. The voices are handed out like piano keys: a
key that still sounds is struck again on its own strings (adding to their vibration, no new voice,
nothing cut off); a new key takes a free voice or, when all sound, the quietest one (released keys
before held ones). The strings and the hammers of all keys are fitted when the piano is created
(some tens of milliseconds), so playing it does no fitting: a silent piano rests from the start, a
note costs a fraction of a millisecond. Sixteen sounding voices with the pedal down take under 10%
of a core (the strings ringing along run at half the sample rate); a piano that is silent rests. The
instrument sounds 128 samples (3 ms) late, the time the soundboard convolution takes; it reports
that latency, so a mixer lines the other strips up with it.

**Note messages**: `pitch` (or `frequency`), `velocity` (0-1, default 0.8: the tone always, the
level by `VelocitySens`), and every setting below. **Instrument messages**, the pedals: `sustain`
(0 up, 1 down, half pedaling in between: from 0.3 to 0.7 of the travel the dampers lift), `sostenuto`
and `soft` (above 0.5 is down).

| Setting | Default | Meaning |
|---|---|---|
| `Hardness` | 0.5 | 0-1, hardness of the hammer felt: soft and round to hard and bright |
| `StrikePosition` | 0.125 | Where the hammer hits, 0.05-0.3 of the string: the harmonics with a node there are weaker (1/8 is classic, nearer the end is brighter, thinner) |
| `Soundboard` | 0.6 | 0-1, the knock of the soundboard in the strike |
| `Body` | 0.6 | 0-1, how much of the sound comes through the soundboard (at 1 a little of the direct strings stays, for clarity) |
| `SoundboardIR` | nil | A measured soundboard response (one slice per side, at the sample rate) instead of the modelled board |
| `Bark` | 0.5 | 0-1, the bark of the bass strings played hard (phantom partials, longitudinal mode) |
| `Mechanics` | 0.5 | 0-1, the noise of the action: dampers, keys, the sustain pedal |
| `Stretch` | 1 | How much of a tuner's stretch tuning, 0 is equal temperament |
| `Variation` | 0.5 | 0-1, how different the keys and every strike are, and how irregularly the partials decay |
| `Treble` | 0 | dB, the top octave louder or softer than voiced |
| `Position` | `Player` | `Player`, `Studio` or `Hall`: where the piano is heard from |
| `Resonance` | 0.5 | 0-1, sympathetic resonance of the strings whose dampers are up |
| `Stiffness` | 1 | Inharmonicity, 1 is a grand, more an upright (or a toy, a struck rod), 0 perfectly harmonic |
| `Decay` | 1 | Scales the ring times |
| `Damping` | 1 | How much faster the highs die |
| `Unison` | 1 | Cents the strings of a key are apart: around 1 is in tune with a slow beat, 10+ honky tonk |
| `Strings` | 3 | Strings per key at most (1-3): fewer are thinner and cheaper |
| `DamperTime` | 0.25 | Seconds a released key rings on (longer in the bass) |
| `Width` | 0.6 | 0-1, the keys spread from left (low) to right (high), as the player hears it |
| `VelocitySens` | 0.7 | 0-1, how much the velocity changes the level (0.7 is a recorded grand: pianissimo to fortissimo some 25 dB) |
| `Level` | 1 | Output level |

**Presets** (`piano.Presets` by name):

| Preset | Character |
|---|---|
| `grand` | A concert grand as it is usually recorded, from close microphones: warm hammers, a lot of soundboard and wood, strings that beat a little |
| `concert` | The grand from the audience: the lid throwing the sound into the hall, brighter hammers to carry |
| `warm` | A grand in a jazz club: close, round hammers, a rich body, the strings ringing along freely |
| `baby` | A small grand: shorter, stiffer strings, less sustain, a boxier soundboard |
| `bright` | Hard hammers for pop and rock, clean and present |
| `upright` | Stiffer, shorter, boxier, a little out of tune |
| `vintage` | An old upright: worn hammers gone hard, strings drifting apart, a noisy action |
| `felt` | A felt piano, soft and close, for quiet music and ambient |
| `cinematic` | Soft, dark hammers from the hall, a lot of sympathetic resonance: the slow film-score piano |
| `honkyTonk` | Strings far out of tune with each other |
| `toy` | One short string per key, struck hard in the middle: a bright, hollow, bell like plink |

A struck string peaks high above its loudness, so the
presets are calibrated to a -1 dB peak at velocity 0.8 (about -14 LUFS); played harder they go
over, leave headroom or use the mixer's limiter.

**Messages**: `pitch`, `frequency`, `velocity`, `level`, the pedals `sustain`, `sostenuto`, `soft`, and the voicing: `hardness`, `strikePosition`, `velocitySens`, `variation`, `body`, `bark`, `treble`, `soundboard`, `resonance`, `stiffness`, `stretch`, `decay`, `damping`, `damperTime`, `unison`, `strings`, `mechanics`, `position`, `width`.

## The action

The piano is a physical model of a grand: every key has one to three strings (a waveguide with a loss
filter, dispersion for the inharmonicity of its register and a tuning allpass), slightly detuned and
sharing a bridge, so a note falls fast and then lingers (double decay) and beats. A felt hammer is
simulated at the velocity of the key (stiffness k * x^p), so a harder strike is shorter, peakier and
brighter and can re-contact the string in the treble. The soft pedal strikes one string less; the
dampers, the sustain pedal (half pedalling), the sostenuto pedal and the sympathetic resonance are
modelled; the bass barks when played hard (phantom partials at sums of partials); octaves are stretch
tuned (`stretch`, `stiffness`). `mechanics` is the noise of the action: the dampers landing at release
(not with the pedal down), the key coming up, the pedal, and `keyBed` (0-1), the thud of a key struck
hard landing on the keybed.

## Bridge and wound strings

The strings of all keys end on one bridge. Its admittance is a shared bank of about 60 soundboard
modes, sparse in the bass and denser above, damped like wood. Every string puts its force on the bridge
each sample, and the bridge's motion goes back into every string: a partial that meets a mode drains
fast and one between modes keeps its energy. A chord is therefore not the sum of its notes, and with
the sustain pedal down the washed tail is longer. `bridge` (0-1, default 0.6) blends from each key
draining on its own (0) to full coupling (1); the mean drain per key stays what it was. `wound` (0-1,
default 1) makes the keys below the second string break (key 44) wound strings: much less inharmonic
than plain strings, with a step at the break about a key wide, and their copper wrap damps the highs
and loads the bridge more. `precursor` (0-1, default 0.5) adds the short burst of longitudinal waves that
outruns a hard-struck bass note (velocity cubed, gone in the treble). `strikeRegister` (0-1, default 1)
moves the hammer strike point from 1/8 in the bass towards 1/12 in the treble. The mode bank is generic,
not measured soundboard data.
