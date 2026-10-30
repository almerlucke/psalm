# pulsar

A **pulsar synth**: every note is a train of tiny pulses at the pitch of the note, and every pulse is a short burst of a sine at the **formant** frequency, shaped by a window. The formant puts a peak in the spectrum that stays where it is when the pitch changes, like a vowel. At ordinary pitches this gives vowel pads, talking leads and basses; at very low notes (a few Hz) the pulses are heard one by one, so a note becomes a rhythm, a cloud of grains or a crackle. No samples are used. Its `notes` input takes notes; the audio output goes on to an effect or the output.

## Tutorial

1. Add a pulsar. Choose a **preset** when you add it and in the editor (the module's window): `ahPad`, `oohPad`, `talkLead`, `wahLead`, `reeseBass`, `harpPulsar`, `grainCloud`, `machineGun`, `crickets`, `fireCrackle`. The presets from `grainCloud` on are meant for very low notes (around 10 to 80 Hz). Cable a keyboard, [voice](voice.md) or [part](part.md) to `notes`.
2. Play a note and sweep **formant**. The pitch stays, the colour moves: low is "ooh", mid is "ah", high is thin and bright. **formant2** (0 is off) adds a second peak, and two peaks make real vowels. **formantTracking** is how much the formants follow the pitch (0 is a fixed vowel, 1 the same timbre on every note).

```example An "ah" vowel pad
[set]
formant = 700
formant2 = 1200
formant2Level = 0.5
formantTracking = 0
duty = 0.5
window = 0          # hann
filterType = 0      # no filter
noteSpread = 0.6
vibratoRate = 5
vibratoDepth = 0.1
glide = 0
```

3. **duty** is how much of each period the pulse fills: short is bright and thin, full is round. **window** shapes each pulse: `hann` is smooth, `gaussian` is a purer formant, `percussive` is struck, `rectangle` is buzzy.
4. **formantLFORate** and **formantLFODepth** (octaves) make the vowel drift; **formantEnvAmount** opens the formant with the envelope at every note: the "wah".

```example A talking lead
[set]
formant = 600
formant2 = 1800
formant2Level = 0.5
formantTracking = 0.3
formantLFORate = 3
formantLFODepth = 0.5
glide = 90
duty = 0.6
window = 1          # gaussian
filterType = 0
```

5. Go low: play notes around 10 to 40 Hz. Now **burst** and **burstRest** (pulses on, pulses off, repeated), **masking** (the chance a pulse is skipped) and **jitter** (a random period) make rhythms and textures. **stereoSpread** pans each pulse randomly.

```example Crackling grains (play notes around 40 Hz)
[set]
formant = 3000
duty = 0.2
window = 1
masking = 0.5
jitter = 0.3
stereoSpread = 0.8
formantTracking = 0
filterType = 0
```

```example Three ticks and a rest (play notes around 12 Hz)
[set]
formant = 5000
duty = 0.1
window = 2          # percussive
burst = 3
burstRest = 2
masking = 0
jitter = 0
filterType = 0
```

6. There is also a **filter** (**filterType**, **cutoff**, **resonance**, **filterEnvAmount**) after the pulses, and a modulation matrix of 8 slots; the first preset (`acidPulsar`) uses it to let the velocity move the level. Envelopes are in the editor.

## Settings

| Setting | What it does |
|---|---|
| `formant`, `formant2`, `formant2Level` | the frequency of the peak inside each pulse, a second one and its level |
| `formantTracking` | how much the formants follow the pitch |
| `duty` | the part of each period the pulse fills |
| `window`, `pulsaret2`, `shapeMix` | the pulse window; a second pulse shape and the mix between the sine and it |
| `burst`, `burstRest` | pulses sounding, pulses silent |
| `masking`, `jitter` | chance of a skipped pulse; randomness of the period |
| `stereoSpread`, `noteSpread` | random pan per pulse, per note |
| `formantEnvAmount`, `formantLFORate`, `formantLFODepth` | the formant moved by envelope and LFO |
| `vibratoRate`, `vibratoDepth`, `glide` | pitch vibrato and slide |
| `filterType`, `cutoff`, `resonance`, `filterEnvAmount`, `keyTrack` | a filter after the pulses |
| `mod1Src` ... `mod8Amount` | the modulation matrix |
| `level` | output level |

Related: [subtractive](subtractive.md) (its formant filter), [wavetable](wavetable.md), [voice](voice.md).
