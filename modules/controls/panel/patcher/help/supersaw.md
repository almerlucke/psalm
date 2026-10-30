# supersaw

The **supersaw** of the Roland JP-8000: a stack of up to sixteen detuned sawtooths (seven by default), the sound of trance pads and leads. The detune and the level of the centre saw against the others follow the curves measured on the original. The saws are spread over the stereo field and go through a filter and an optional string-machine ensemble. It is built to be big with very few controls. Its `notes` input takes notes; the audio output goes to an effect or the output.

## Tutorial

1. Add a supersaw. Choose a **preset** when you add it and later in the editor (the module's window): `epicPad`, `stringMachine`, `anthemLead`, `reeseBass`, `houseStab`, `gatedPad`, `ravePluck`, `riseUp`. Cable a keyboard, [voice](voice.md) or [part](part.md) to `notes`.
2. Hold a chord and move **detune**. At 0 the saws are in tune and it is a plain fat saw; as it grows they pull apart into the shimmering wall. **mix** is the level of the centre saw against the others: low, it is all the spread; high, the centre stays solid.
3. **saws** is how many saws: 1 to 16. **stereo** spreads them left and right.

```example A wide trance pad
[set]
saws = 7
detune = 0.45
mix = 0.4
stereo = 1
sub = 0
cutoff = 3500
resonance = 0.1
hpCutoff = 200
ensemble = 0.5
ensembleRate = 0.5
```

4. **cutoff** and **resonance** are a low pass filter; **filterEnvAmount** opens it with the filter envelope at each note (envelopes are in the editor). **hpCutoff** takes the lows away (20 is off), which keeps a pad out of the bass line's way.
5. **ensemble** adds a chorus of three modulated delays, swept slowly at **ensembleRate**; it makes the sound wide and moving.

```example A hard lead with a sub
[set]
saws = 7
detune = 0.3
mix = 0.6
sub = 0.3
cutoff = 6000
resonance = 0.25
filterEnvAmount = 1.5
hpCutoff = 80
ensemble = 0
```

6. **lfoRate** with **lfoCutoff** and **lfoDetune** sways the filter and the detune; the matrix (8 slots) can send the wheel, velocity or the second LFO to detune, stereo, ensemble and more.

```example The detune breathes
[set]
saws = 7
detune = 0.25
lfoRate = 0.2
lfoDetune = 0.4
lfoCutoff = 0.5
cutoff = 3000
resonance = 0.1
ensemble = 0.6
```

## Settings

| Setting | What it does |
|---|---|
| `saws` | how many saws (1 to 16) |
| `detune` | how far the saws are pulled apart |
| `mix` | the centre saw against the others |
| `stereo` | the stereo width of the saws |
| `sub` | a square an octave down |
| `cutoff`, `resonance` | the low pass filter |
| `hpCutoff` | a high pass before the filter (20 is off) |
| `filterEnvAmount` | octaves the filter envelope opens the cutoff |
| `lfoRate`, `lfoCutoff`, `lfoDetune` | an LFO on the cutoff and the detune |
| `lfo2Rate`, `lfo2Shape` | a second LFO, for the matrix |
| `ensemble`, `ensembleRate` | the chorus: amount and speed |
| `mod1Src` ... `mod8Amount` | the modulation matrix |
| `level` | output level |

Related: [subtractive](subtractive.md), [wavetable](wavetable.md).
