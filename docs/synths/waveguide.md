# waveguide

`synths/waveguide`

Strings that are plucked or bowed: a wave travelling up and down a string, fed by a pluck or kept
going by a bow, heard through the body of the instrument. The pluck is an extended Karplus-Strong
string (in tune within a few cents over the whole range); the bow is the McIntyre, Schumacher and
Woodhouse model after the STK: the bow sticks and slips, which keeps the string going as long as
it moves.

```go
guitar := waveguide.NewPoly(waveguide.Guitar(), 8).AddTo(p)
cello := waveguide.NewPoly(waveguide.Cello(), 4).AddTo(p)
```

| Setting | Model | Meaning |
|---|---|---|
| `Model` | | `Pluck` or `Bow` |
| `Decay` | pluck | Seconds the string rings at 220 Hz; lower strings ring longer |
| `Damping` | pluck | 0-1, how much faster the high partials die |
| `Brightness` | pluck | 0-1, a soft finger to a hard pick; the velocity adds |
| `PickPosition` | pluck | 0-0.5 from the bridge: thin near it, round in the middle (the partials with a node there are missing) |
| `Release` | pluck | Seconds the string dies in once let go, 0 lets it ring (a harp) |
| `Pressure` | bow | 0-1, airy to gritty |
| `BowPosition` | bow | 0.02-0.5 from the bridge |
| `BowAttack`, `BowRelease` | bow | Seconds |
| `VibratoRate`, `VibratoDepth`, `VibratoDelay` | bow | Hz, semitones, seconds |
| `Body`, `BodyMix` | | `GuitarBody`, `ViolinBody`, `CelloBody`, `HarpBody`, or `NoBody`; 0-1 |

The bow's speed is its loudness: the note amplitude sets it (faster than the model allows, a
string jumps to its octave, as a real one does). Bowed notes are in tune within a few cents from C2 to C7 (the
friction's pull is measured and corrected). **Value inputs**: 1 pressure, 2 vibrato depth.

| Preset | Character |
|---|---|
| `guitar`, `steel`, `muted` | Nylon, steel with a pick, palm muted |
| `harp` | Plucked in the middle, rings on |
| `bass` | A fingered electric bass |
| `pizzicato` | A violin plucked |
| `violin`, `cello` | Bowed, vibrato coming in |
| `slowStrings` | Swelling in and out |
| `ponticello` | Bowed near the bridge, glassy |
| `flautando` | Bowed lightly over the fingerboard |
| `glassHarp` | A string without losses: shimmers for many seconds |
| `wire` | A steel wire picked at its very end: thin, jangling, metallic |
| `eBow` | A string kept singing by a steady bow: endless sustain |
| `ghostBow` | Bowed so lightly the note floats, a slow wide vibrato |
| `scrape` | Pressed hard near the bridge: gritty, metallic, on the edge |

**Messages**: `pitch`, `frequency`, `velocity`, `level`, `decay`, `damping`, `brightness`, `pickPosition`, `release`, `bowPressure`, `bowPosition`, `vibratoRate`, `vibratoDepth`, `bodyMix`, and the [pitch bend, mod wheel and aftertouch](../../synths.md#pitch-bend-mod-wheel-and-aftertouch).

## Filter, matrix and sympathetic strings

`Config.Filter` is the [voice filter](../../synths.md#the-voice-filter) after the string and body; `Config.Mods` an 8
slot [modulation matrix](../subtractive.md#modulation-matrix). Sources: amp env (the bow speed, or the level of a
plucked string), filter env, lfo 1 (the vibrato), lfo 2 (a plain sine, `lfo2Rate`), velocity, key,
random, wheel and ext 1-4 (value inputs 3 to 6, `waveguide.InExt1`). Destinations: `decay`, `damping`,
`brightness`, `pickPosition`, `bowPressure`, `bowPosition`, `vibratoDepth`, `bodyMix`, `pitch`,
`level`, `cutoff`. Brightness and pick position are the pluck itself: they are read at the start of
the note, so velocity, key, random, wheel and ext are the sources for them.

`sympathetic` (0-1) feeds the sound into four strings tuned an octave below, a fifth, an octave and
an octave and a fifth above the note, and adds them back: a shimmering, resonant string. It costs
nothing at 0.

## Membrane and pipe

Two more models join the pluck and the bow. `Membrane` is a struck drum skin: 14 modes of an ideal
circular membrane (1, 1.59, 2.14, 2.30, 2.65, 2.92 ... times the pitch), each a damped oscillator,
started with the loudness the place of the strike gives it (`strike`, 0 the middle to 1 the rim; the
middle leaves only the round modes). `hardness` (plus velocity) brightens the stick, `decay` is the
seconds the lowest mode rings and `damping` how much faster the higher ones die. `tension` raises the
pitch at the strike and lets it fall back (a tom's thump, a tabla's bend), `snare` adds wires buzzing
along, and `release` is a hand laid on the skin (conga). Presets: `tom`, `floorTom`, `conga`, `tabla`,
`frameDrum`, `snareDrum`.

The stick has a contact time: it presses on the skin for a moment (a fraction of a millisecond for a
hard, fast stick, several for a soft one, set by `hardness` and the velocity) instead of hitting in an
instant, so the sound begins at nothing and rises, not with a step (which was a click, and most of what
a drum sounded like). A mode that turns over in less than the contact hardly feels it, so a soft stick
leaves the highs out.

`Pipe` is a blown tube: by default a flute (an air jet driving a delay-line tube through a saturating
curve), with `reed` on a clarinet (a single reed on a closed tube, odd partials, an octave lower).
`breath` is the pressure (too little does not sound, too much overblows or chokes the reed), `noise`
the air, `embouchure` the jet's offset (even partials) or the reed's stiffness, `damping` darkens the
tube; the attack and release (`ampAttack`, `ampRelease`) shape the breath and the vibrato is the bow's.
A pitch keeper trims the tube so it plays in tune whatever the blowing: it moves the length a little
at a time (a delay line whose length jumps clicks) and waits until the tone has stopped growing. A note
speaks within a fraction of a second at any pitch (a reed's tube starts with a small puff in it, a flute's
jet has more gain while the tone is weak), and a flute that starts to sound an octave too high is eased
off the jet, as a player would (not for a breath above 1, which asks for the overblown note). Presets: `flute`, `clarinet`,
`panpipe`. Both take the voice filter, the body, and the matrix (new destinations `breath` and
`embouchure`; the membrane's `decay`, `damping` and `pitch`).

## Real strings

The waveguide string can be made more real, each part off at 0. `inharmonicity` is the stiffness B at
220 Hz: allpass sections stretch the partials to f0 * n * sqrt(1 + B * n^2) while the fundamental stays
in tune, and low strings are stiffer (nylon about 3e-5, steel 1e-4, wound bass 3e-4; the fit is
accurate to a few cents up to B = 5e-4 at 220 Hz). `polarization` adds a sideways string slightly lower
in pitch and weaker in damping, giving a prompt sound, an aftersound and beats. `bodyCoupling` lets the
bridge give way to the body's modes: notes on a body resonance die faster and the body hands energy
back. `tensionMod` makes a hard pluck start sharp and glide down (the twang), scaled by velocity.

`pluckWidth` pulls the string aside like a finger or pick, `plectrum` goes from soft finger to hard
pick, and `fretNoise` and `fretBuzz` add the finger rustle and the string hitting the fret. `harmonic`
puts a finger on a node (0.5 sounds the octave, 1/3 the twelfth), and `slide` glides a still-ringing
string to the next pitch for legato (only when the same voice gets the next note).

For the bow, `bowForce` switches to a friction curve with stick-slip and a memory of the rosin's heat,
with `bowSpeed` as its partner; the force has to fit the speed: too light is airy, too heavy scratches.
`bowNoise` adds the rosin's grain and `bowReverse` sets the seconds per stroke before the bow turns
round. `bowForce` and `bowSpeed` are matrix destinations. New presets: `uprightBass`, `harmonics`,
`rosin` (the friction-curve violin); the plucked presets are re-voiced with the new parts.

## Stiff strings, the friction bow, harmonics, fret contact

Stiff strings (`inharmonicity` up to about 2e-3 at 220 Hz: a wound bass or a piano string) use a chain of
up to 40 first-order allpass sections whose coefficient is searched against the loop's exact phase, so
the first twelve partials sit within a few cents of f0 * n * sqrt(1 + B * n^2) with the fundamental in
tune; gentle strings keep the cheap fit.

The bow has a switch, `bowModel`: 0 is the table (`pressure`), 1 the friction curve with stick-slip and a
rosin memory (`bowForce`, `bowSpeed`, `bowNoise`, `bowReverse`). The violin, cello, slowStrings,
ponticello, flautando, ghostBow and rosin presets use friction, with a weight that follows the speed,
which follows the velocity (rosin has a fixed speed and force, in the one regime of the friction curve that
stays steady over most of the range, and its bow does not turn round: `bowReverse` 0). Its pitch is held to within about 6 cents from pitch 36 to 96: when a note
starts the bow looks up a tuning for its pitch, force, position and speed, found by running the bow loop
on a background worker (the cache warms about 1-2 s after the synth is made; `waveguide.WarmBow(cfg)`
and `WaitBow()` let a test or patch wait for it). A first note at a pitch not yet in the cache plays on
an interpolated estimate (within about 20 cents) and is exact from the next. If no weight holds the
string's motion (the highest notes), the note falls back to the table bow. The cello and the others
swing in loudness between pitches (about -12 LUFS at pitch 57, louder at 69).

Slide also works in mono legato: a pitch message to a sounding string glides over `slide` seconds
without a new attack, for plucked and bowed strings. `harmonic` puts a finger on the node, a comb in the
loop that damps the other partials, also on a string that is already ringing. `fretBuzz` is a stiff
spring and damper at `fretPosition` (0.05 to 0.98 of the string, default the fret beside the finger): it
buzzes only when the action is low (`fretBuzz` high) and the pluck is hard.
