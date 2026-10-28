# modal

`synths/modal`

Struck things: bars, tubes, bells, tines, each ringing in its modes (partials with their own
pitch ratios and decay times), struck by a mallet. The mallet is a force as long as it touches:
a soft one touches long and wakes few high modes, a hard one is bright. Where it strikes mutes the
modes with a node there.

```go
mallets := modal.NewPoly(modal.Marimba(), 8).AddTo(p)
bell := modal.New(modal.Config{Modes: []modal.Mode{{1, 1, 1}, {2.76, 0.5, 0.6}}, Decay: 3, Level: 1})
```

| Setting | Meaning |
|---|---|
| `Modes` | `[]Mode{Ratio, Amp, Decay}`: pitch over the note, loudness, decay relative to `Decay` |
| `Decay` | Seconds the lowest mode rings |
| `HighDamping` | The higher modes die sooner: decay times (f0/f)^HighDamping |
| `Hardness` | 0-1, the mallet; the velocity adds |
| `Position` | 0-1, where it is struck |
| `Release` | Seconds it dies in once let go (a damper), 0 rings on |
| `Tremolo`, `TremoloDepth` | A vibraphone's motor |
| `Beat` | Each mode splits in two this fraction apart: bells beat |
| `Noise` | 0-1, the knock of the mallet |

**Value inputs**: 1 hardness, 2 tremolo depth.

| Preset | Character |
|---|---|
| `marimba`, `xylophone`, `vibraphone`, `glockenspiel` | The bars of the orchestra |
| `tubularBells` | Chimes |
| `churchBell` | Hum, prime, tierce, quint and nominal, beating |
| `kalimba` | A thumb piano |
| `singingBowl` | Long and beating |
| `woodBlock` | A pitched knock |
| `alienBell` | Ratios no bell has, each mode beating |
| `gamelan` | A bronze metallophone shimmering in tuned pairs (ombak) |
| `crystal` | A glassy cluster of long high modes with a gentle tremolo |
| `steelPan` | Tuned harmonics, a little out of tune with themselves |
| `metalPlate` | A dozen dense modes: a long, industrial ring |
| `spaceTine` | A tine ringing for ages in a fast tremolo |

**Messages**: `pitch`, `frequency`, `velocity`, `level`, `decay`, `highDamping`, `hardness`, `position`, `release`, `tremolo`, `tremoloDepth`, `noise`, and the [pitch bend, mod wheel and aftertouch](../../synths.md#pitch-bend-mod-wheel-and-aftertouch).

## Excitation, inharmonicity, filter, matrix

`excitation` chooses what rings the modes: `strike` (the default mallet), `bow` (a noise-driven bow
that fades in and keeps the modes singing while the note is held, and lifts on release) or `noise
burst` (a short windowed noise burst, shorter the harder the mallet, for brushes and scrapes).
`inharmonicity` (0-1) stretches the mode ratios relative to the first mode, the higher ones more, for
stiff metal and glass; it applies at the next strike. The material presets are `wood` (dull and
short), `metal` (long, stretched) and `glass` (pure and high).

After the modes there is a [voice filter](../../synths.md#the-voice-filter) whose envelope starts at the strike. The
8 slot [modulation matrix](../subtractive.md#modulation-matrix) takes filter env, amp (the decay), LFO 1 (a plain sine at
`lfoRate`, running when used), velocity, key, a random value per note, the wheel and Ext 1-4 (value
inputs 3-6, `modal.InExt1`) to `pitch`, `decay`, `hardness`, `position`, `tremoloDepth`, `beat`,
`noise`, `level`, `cutoff`, `resonance`, `inharmonicity` and `highDamping`. The destinations that shape
the strike are read when the note is struck, from the sources known then.

The matrix has LFO 2, a plain sine from the strike at `lfo2Rate` Hz (default 1), running when a slot uses
it. The pitch from the matrix is followed every 8 samples (re-tuning the modes only when pitch is a
destination), so a fast LFO on pitch is a smooth vibrato.

## Instruments: bars, bells and plates

The modal synth can ring in a physical model of its body instead of a hand-written mode list: the
`instrument` choice picks a uniform bar (1 : 2.756 : 5.404 : 8.933 ...), an undercut marimba or
vibraphone bar tuned 1 : 4 : 10, a xylophone bar (1 : 3 : 6), a glockenspiel bar, a clamped tine
(kalimba), a tubular bell (modes 2 : 3 : 4 : 5 : 6 over a strike tone that is not itself sounding), a
church bell, a handbell, a singing bowl, a free circular plate, a gong, a cymbal or a steel pan. Each
mode dies at a rate that grows with the square of its frequency, and the mallet wakes each mode by the
mode shape at the strike `position` (along the bar from its end, from the centre of a plate to its
rim, from the crown to the lip of a bell), so hitting the node of a mode silences it.

The mallet is a Hertz contact: it touches for a time that grows with its `mass` and shrinks with the
velocity, `contact` sets how strongly a harder blow brightens the sound, and `hardness` still picks the
mallet. `resonator` adds the tube under a marimba or vibraphone bar, which strengthens and lengthens
the fundamental, and the vibraphone tremolo swings the tube; the mod wheel deepens the tremolo. Bells,
bowls and plates have pairs of near-equal modes: `spread` splits them so they beat, and `drift` makes
each note a little different (the pairs turn, the split and the pitch vary); it is off by default and
draws random numbers only when on. Presets reworked onto the models: marimba, vibraphone, xylophone,
glockenspiel, tubularBells (now pitched at the strike tone), churchBell, singingBowl, kalimba,
spaceTine, metalPlate, steelPan, wood, metal; new: handbell, gong, cymbal, celesta. The handbell, gong
and cymbal ratios are approximations; the nonlinear pitch glide of real gongs is not modelled.

## Undercut bars, nonlinear gongs

Undercut bars are no longer uniform beams. At first use the marimba, vibraphone and xylophone solve
an Euler-Bernoulli beam with an arched thickness profile (40 finite elements, subspace iteration) and
tune the arch so the modes sit at 1:4:10 (marimba, vibraphone) or 1:3:6 (xylophone), within 0.2 cents.
The strike position reads the solved mode shapes. `undercut` (0.01-1, default 1) blends from a uniform
bar to the tuned arch.

Gongs and cymbals have `nonlinearity` (0-1; 0.6 and 0.8 in their presets, off elsewhere), all of it
scaling with the blow's velocity squared. For a **cymbal** (and the hi-hats and the thunder sheet) a hard
blow starts up to 5% sharp and glides down as the modes decay, each mode passes energy up to the modes
above it with a random phase (a crash blooms into a hiss), and the modes shimmer with a random detuning.
A **gong** (`gong`, `gongAgeng`, `gongSoft`, `tamTam`, `bowedGong`) is gentler, since the same would make
it noisy and steer its pitch: the push is at most 2% (about 15 to 25 cents on a hard blow), the energy
moves up at half the rate and joins what already rings, in its phase, so the high modes swell instead
of being kicked about, and the shimmer is smaller (about a cent) and drifts over a second. The church
bell uses ratios after Rossing and Perrin (hum 0.5, prime 1, tierce 1.2, quint 1.5, nominal 2, ...) with
beating pairs; the handbell, gong and cymbal ratios are approximations. New instruments and presets:
`woodBlock` (re-voiced), `templeBlock` and `bottle`, a Helmholtz cavity at the pitch plus the wall's modes.
