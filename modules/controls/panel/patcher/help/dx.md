# dx

A six-operator **FM synthesizer** in the style of the Yamaha DX7: bells, electric pianos, bass, brass and glassy pads made by letting sine-like oscillators bend each other. It plays the notes cabled to its `notes` input and has one audio output.

## FM in plain words

Every **operator** is an oscillator with its own envelope. There are two jobs:

- A **carrier** is heard.
- A **modulator** is not heard itself; it bends the pitch (strictly, the phase) of the operator it is connected to, many times a second. That adds extra partials around the carrier, called sidebands.

How much a modulator is turned up decides how many sidebands appear (the *modulation index*): a little is a soft, slightly bright tone, a lot is hard and metallic. Because each operator has an envelope, a modulator that dies away quickly makes a note that starts bright and settles, like a struck string or tine.

The **frequency ratio** of the modulator to the carrier decides what kind of partials you get. Whole numbers (1, 2, 3) give harmonic, pitched tones; ratios like 1.41 or 3.5 give bells, bars and drums, because the sidebands no longer line up as harmonics.

The **algorithm** is the wiring: which operators are carriers and which modulates which. The DX7 has 32 of them. In one, a stack of operators feeds a single carrier (basses, leads, complex tones); in another, several short pairs sit side by side (layered tones, electric pianos); in the last ones all operators are carriers, which adds sines together like an organ. **Feedback** lets an operator modulate itself: a sine moves towards a saw and, at the most, towards noise.

## Tutorial

1. Add a keyboard (or a [voice](voice.md) or [part](part.md)) and a dx. Cable the `notes` output to the dx's `notes` input, and take the dx's audio out to a speaker or mixer.
2. The preset you choose when you add the module is the sound it starts as (see below). Play it, then open the module: its window is the dx editor, with the algorithm drawn as a diagram of the six operators and all their settings.
3. Change one thing at a time. First the **level** of a modulator (`op2Level`): the louder, the brighter and harder the tone. Then its **coarse** setting (`op2Coarse`), the ratio: whole numbers stay musical, other values turn metallic.
4. Many presets (the FM ones such as `accordion`, the first) play *exact* operators: their own ratios, levels and envelopes in plain numbers, and a routing of their own. The examples below switch `opNExact` off for operators 1 to 4, so that the classic 0-99 values and the `algorithm` number rule; `custom = 0` plays the numbered algorithm instead of the preset's own routing.

```example Two FM pairs (algorithm 5)
[set]
custom = 0
op1Exact = 0
op2Exact = 0
op3Exact = 0
op4Exact = 0
algorithm = 5    # 2>1, 4>3 and 6>5: three pairs, each carrier with its modulator
op2Coarse = 2    # modulator one octave above its carrier
op2Level = 75    # the index: lower is softer
op4Coarse = 7
op4Level = 60
```

5. **Feedback** lets the last operator of the chain modulate itself: a sine moves towards a saw, and more feedback towards noise. In algorithm 5 it is operator 6 that feeds back into itself and modulates operator 5:

```example Feedback on a pair
[set]
custom = 0
op1Exact = 0
op2Exact = 0
op3Exact = 0
op4Exact = 0
algorithm = 5
op5Level = 70
op6Level = 50
feedback = 5     # 0-7
```

6. **An algorithm of your own.** With `custom = 1` the editor plays the wiring you draw: `carrierN` marks the operators that are heard, `modFT` says that operator F modulates operator T (a modulator has a higher number than what it modulates), and `fbFrom` and `fbTo` are the feedback loop. This is the stack 4 into 3 into 2 into 1:

```example A stack of four, only operator 1 is heard
[set]
custom = 1
carrier1 = 1
carrier2 = 0
carrier3 = 0
carrier4 = 0
mod21 = 1
mod32 = 1
mod43 = 1
fbFrom = 4
fbTo = 4
```

7. The dx has the filter of the other synths (off at first) and an LFO for vibrato (`lfoPitchDepth`, scaled by `pitchModSens`) or tremolo:

```example A low-pass filter and a slow vibrato
[set]
filterType = 1     # low pass
filterCutoff = 60  # 0-99
filterResonance = 25
lfoSpeed = 30
lfoDelay = 40      # the vibrato fades in
lfoPitchDepth = 20
```

Hard FM (a high index, feedback, high notes) makes harmonics above what the sound card can hold, which fold back as harsh, inharmonic alias. **oversample** (0 off, 1 twice, 2 four times) plays the operators faster and filters, at more CPU cost.

## Presets

Presets are chosen when the module is added, and again in its editor (the window of the module), where the bank of presets is listed. A few of the names: `accordion`, `ePiano`, `bell`, `bellPad`, `glassKeys`, `fmBass`, `tubularBells`, `grandPiano`. The editor can also load `.syx` banks of voices if a SysEx folder is set up. No factory DX7 banks are included.

## Settings

| Setting | What it does |
|---|---|
| `algorithm` | 1-32, the wiring of the operators (with `custom` 0) |
| `custom`, `carrierN`, `modFT`, `fbFrom`, `fbTo` | an algorithm of your own: on, the heard operators, who modulates whom, the feedback loop |
| `feedback` | 0-7, an operator modulating itself |
| `transpose` | 0-48, 24 is in tune |
| `oversample` | 0-2, alias reduction |
| `oscSync` | restart the operators at every note |
| `lfoWaveform`, `lfoSpeed`, `lfoDelay`, `lfoSync` | the LFO |
| `lfoPitchDepth`, `pitchModSens` | vibrato depth |
| `lfoAmpDepth` | tremolo depth |
| `pitchRate1`-`4`, `pitchLevel1`-`4` | the pitch envelope (level 50 is no shift) |
| `opNCoarse`, `opNFine`, `opNDetune` | frequency of operator N (1-6) |
| `opNFixed` | 1: a fixed frequency instead of a ratio |
| `opNLevel` | 0-99, loudness of a carrier or modulation of a modulator |
| `opNWave` | 0-7, the wave of the operator: sine, half sine, abs sine, pulse sine, even sine, even abs, square, log saw |
| `opNRate1`-`4`, `opNLevel1`-`4` | the operator envelope: four rates and four levels |
| `opNVelocitySens`, `opNRateScaling`, `opNAmpModSens` | velocity, key and LFO response |
| `opNBreakPoint`, `opNLeftDepth`, `opNRightDepth`, `opNLeftCurve`, `opNRightCurve` | level scaling over the keyboard |
| `opNExact` | 1 plays the preset's exact values (ratio, gain, envelope in ms) instead of the 0-99 ones |
| `filterType`, `filterCutoff`, `filterResonance`, `filterEnvAmount`, `filterKeyTrack`, `filterAttack`-`filterRelease` | the voice filter, 0-99 |

Related: [shaper](shaper.md) makes FM too (chains of shapers), [piano](piano.md) and [modal](modal.md) are physical models of sounds dx imitates, [voice](voice.md) and [part](part.md) play it.
