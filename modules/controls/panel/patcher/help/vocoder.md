# vocoder

An FFT **vocoder**. It has two inputs: the **voice** (the modulator, usually speech or singing) and the **carrier** (usually a bright synth). The spectral envelope of the voice is imprinted on the carrier: the synth seems to speak. Both inputs are mono (a stereo source is mixed down), the output is stereo.

## Tutorial

1. Add a sound input (a microphone, use headphones) or a voice sample, a bright synth and a vocoder. Cable the voice's audio `out` to the vocoder's `voice` input, the synth's audio `out` to the `carrier` input, and the vocoder's `out` to the mixer.
2. Play chords on the synth while speaking. Without a carrier playing the vocoder is silent.
3. Carriers need many harmonics: detuned saws, a little noise, an open filter. A dull carrier gives a muffled voice.
4. **attack** and **release** (ms) are how fast the bands follow the voice: short is clearer, long is smoother and softer.
5. **flatten** removes the carrier's own tone first (1 all of it), which makes the words clearer.
6. **sibilance** lets the voice above **sibilance freq** pass straight through, for the consonants (s, t, f). **formant shift** above 1 makes a smaller voice, below 1 a bigger one. **gain** is the level.

```example Clear robot voice
[set]
attack = 3
release = 40
flatten = 1
sibilance = 0.4
sibilanceFreq = 5000
formantShift = 1
```

```example Big dark voice
[set]
attack = 8
release = 120
formantShift = 0.7
sibilance = 0.2
gain = 1.2
```

7. The vocoder sounds a frame late; a mixer lines up its other strips with it by itself. Outside the mixer that is not done.

## Settings

| Setting | Range | What it does |
|---|---|---|
| attack | 1 to 500 | ms, how fast the bands rise with the voice |
| release | 5 to 2000 | ms, how fast they fall |
| flatten | 0 to 1 | how much of the carrier's own tone is removed first |
| formant shift | 0.5 to 2 | above 1 a smaller voice, below 1 a bigger one |
| sibilance | 0 to 1 | amount of the voice above the frequency that passes through |
| sibilance freq | 1000 to 16000 | Hz |
| gain | 0 to 4 | output level |

## See also

[eq](eq.md), [reverb](reverb.md), [lane](lane.md).
