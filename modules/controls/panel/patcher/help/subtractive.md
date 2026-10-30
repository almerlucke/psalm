# subtractive

The classic **subtractive synth**: oscillators make a bright, harmonically rich sound and a filter carves it down, shaped over time by an amp envelope and a filter envelope. Two oscillators plus a sub oscillator, noise, ring modulation, sync and FM, a choice of filters (state variable, Moog style ladder, formant), two LFOs and a modulation matrix. Basses, leads, pads and plucks all come from the same module. Its `notes` input takes notes; its audio output goes to an effect or the output.

## Tutorial

1. Add a subtractive synth. Pick a **preset** when you add it (and change it later in the editor, the module's window): `acidBass`, `moogBass`, `bass808`, `monoLead`, `warmPad`, `junoPad`, `pluckBass`, `wideReese`, and many more. Cable a keyboard, or a [voice](voice.md) or [part](part.md), to its `notes` input.
2. Play a note and move **cutoff**. The filter lets the low harmonics through and cuts the high ones: low is dark and muffled, high is open and buzzy. **resonance** boosts the harmonics right at the cutoff, from a soft edge to a whistle.
3. **filterEnvAmount** is how far the filter envelope opens the cutoff at the start of each note, in octaves. A big value with a low cutoff gives the "wow" of a bass or a pluck. The envelopes themselves (attack, decay, sustain, release) are in the editor.
4. Mix the sound: **osc1Level** and **osc2Level** set the two oscillators, **osc2Octave** and **osc2Detune** (in semitones; small values thicken, 7 is a fifth, 12 an octave) tune the second, **subLevel** adds a square one octave below.

```example A fat bass
[set]
osc1Waveform = 5   # saw
osc1Level = 0.5
osc2Waveform = 5
osc2Level = 0.5
osc2Octave = -1
osc2Detune = 0.07
subLevel = 0.3
filterType = 1     # ladder
ladderMode = 0     # low 24
cutoff = 260
resonance = 0.3
filterEnvAmount = 3
glide = 0
```

5. **unison** plays that many detuned copies of both oscillators; **unisonDetune** (cents) spreads them, **drift** lets the pitch wander slowly like an analog oscillator. Together they make pads wide:

```example A wide, slowly drifting saw pad
[set]
osc1Waveform = 5
osc1Level = 0.4
osc2Waveform = 2   # triangle
osc2Level = 0.4
osc2Detune = 0.1
unison = 5
unisonDetune = 20
drift = 8
filterType = 0     # state variable
filterMode = 0     # low pass
cutoff = 1400
resonance = 0.15
filterEnvAmount = 1
```

With the envelopes of the first preset a note is short; lengthen the amp attack and release in the editor to make it a pad, or start from a `pad` preset.

6. **glide** (ms) slides from the previous note: a lead or a 303 style line. **pulseWidth** matters for the pulse waveforms (rectangle); a LFO on it gives the classic PWM.

```example A gliding lead
[set]
osc1Waveform = 5
osc1Level = 0.5
osc2Waveform = 5
osc2Level = 0.4
osc2Detune = 0.12
filterType = 1
cutoff = 2400
resonance = 0.25
filterEnvAmount = 1.5
glide = 80
lfo1Waveform = 0   # sine
lfo1Rate = 5.5
lfo1Pitch = 0.2    # a light vibrato, in semitones
lfo1Delay = 400
```

7. The **modulation matrix** (8 slots: a source, a destination, an amount) cables anything to anything inside the synth: the `wheel`, the `velocity`, an LFO. Slot 1 below lets the mod wheel open the cutoff:

```example Mod wheel opens the filter
[set]
mod1Src = 8        # wheel
mod1Dest = 2       # cutoff
mod1Amount = 0.5
```

## Settings

| Setting | What it does |
|---|---|
| `osc1Waveform`, `osc2Waveform` | the shape of each oscillator (sine, triangle, square, saw, ...) |
| `osc1Level`, `osc2Level`, `subLevel`, `noiseLevel` | the mix before the filter |
| `osc2Octave`, `osc2Detune` | tune osc 2 against osc 1 |
| `sync`, `fmAmount`, `ringModLevel` | hard sync, osc 1 modulating osc 2's frequency, osc 1 x osc 2 (metallic) |
| `drive` | saturation before the filter |
| `unison`, `unisonDetune`, `drift` | detuned copies, their spread in cents, slow wandering |
| `filterType`, `filterMode`, `ladderMode` | state variable, ladder or formant, and its slope or kind |
| `cutoff`, `resonance`, `keyTrack` | brightness, peak, how much the cutoff follows the note |
| `filterEnvAmount` | octaves the filter envelope opens the cutoff |
| `filter2Mode`, `filter2Cutoff` | a second filter, in series or parallel |
| `glide` | ms to slide from the last note |
| `lfo1...`, `lfo2...` | two LFOs: rate, delay, and how much they move pitch, cutoff, pulse width, level |
| `mod1Src` ... `mod8Amount` | the modulation matrix |
| `level` | output level |

Related: [voice](voice.md), [part](part.md), [wavetable](wavetable.md) and [supersaw](supersaw.md) for other oscillators.
