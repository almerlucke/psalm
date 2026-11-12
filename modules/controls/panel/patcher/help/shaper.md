# shaper

A synth that makes its wave from a **chain of shapers**. A phase (a ramp from 0 to 1 over every cycle) goes in, the chain bends it step by step, and what comes out is the wave. You write the chains as text, up to four of them (A to D), and the **morph** knob moves between them: 0 is chain A, 1 the last chain, in between a blend of two neighbours. An envelope, an LFO, the velocity or the mod wheel can pull the morph, so the wave itself moves while a note plays.

## Tutorial

1. Add a keyboard (or a [voice](voice.md) or [part](part.md)), a shaper and a speaker. Cable `notes` to the shaper.
2. Open the shaper. Its window has four boxes for the chains. A chain is stages joined by `|`: `bipolar` is a saw, `sin` a sine, `mult 3 | sin` a sine three times as fast, `bipolar | mirror -0.5 0.5` a saw folded back on itself. Edit a chain and press Ctrl+Enter to take it; an error shows in red and the sound goes on with the old chain.
3. Write a sine in chain A and a saw in chain B, then drag **morph**:

```example From a sine to a saw
[chainA]
sin
[chainB]
bipolar
[set]
morph = 0.5     # halfway between the two
filterType = 0  # filter off, to hear the wave itself
```

4. Add a third and fourth chain and the morph walks through all of them. The stages that can fold, quantize or clip a wave add many harmonics:

```example Folds, steps and a sine
[chainA]
sin
[chainB]
bipolar | mirror -0.5 0.5
[chainC]
bipolar | quantize 0.25
[chainD]
par(sin, mult 3 | sin | mult 0.4)
[set]
morph = 0.4
filterType = 0
```

5. **FM is a chain too.** `par(thru, mult 3 | sin | mult 0.3) | sin` is a sine at the pitch whose phase is moved by a sine at three times the pitch: phase modulation, the DX sound. Make the index a **macro**: `m1` to `m4` are values from 0 to 1 that you can move, and `m1[0,0.6]` maps them to 0 up to 0.6. With the macro low it is nearly a sine; high, it is bright:

```example Electric piano by FM, the index on a macro
[chainA]
par(thru, mult 3 | sin | mult m1[0,0.6]) | sin
[set]
macro1 = 0.35
filterType = 0
ampDecay = 600
ampSustain = 0.3
```

The `shapeAmount` setting lets an envelope move `m1` over the note (a bell or a piano is bright at the start and soft after).

6. A second **line** reads the same chains. Give it an octave or some cents of detune and a level and you have a fat, moving sound; **unison** makes up to seven copies of each line, **spread** apart in cents:

```example A wide detuned pad
[set]
level2 = 0.8
detune2 = 7       # cents
unison = 3
spread = 18
uniStereo = 0.8
drift = 0.3
```

7. **xmod** lets line 1 act on line 2: as FM (`xmodType` 0, phase) or as a ring modulator (1, ring). Set `xmodSilent` to 1 to hear only the result.

## The editor

The chains are a rack of cards (one for each stage) with plots: the wave of the chain, the curve of the stage you pick and its harmonics. A number on a card is a constant or a macro (the menu on it); drag cards to move them, copy, take away or leave one out (`●`). The text is under the rack and is the same chain. `Start from` gives a chain to begin with, `Freeze` turns the wave into a table, `Vary in B-D` makes a morph from one chain. The four chains are shown along the morph, with the macros. The envelopes (amp, morph, m1, pitch) are curves to drag, with buttons for fades (`Fade in`, `Swell`, `Fade out`...). `Line 1` / `Line 2` switch the oscillator whose chains you edit: line 2 reads line 1's chains until you give it chains of its own (a different wave to fatten the sound, to modulate or to ring against line 1). The `stereo` sliders make the left and the right read other places in the morph and the macros: one chain, a wide sound. `Play` plays a note.

## Stages you can use

| Stage | What it does to x |
|---|---|
| `thru`, `add a`, `mult m`, `linear scale shift` | x, x + a, x * m, x * scale + shift |
| `mod m`, `ripple m` | the remainder of x / m, and x plus it |
| `bipolar`, `unipolar` | 0..1 to -1..1, and back |
| `abs`, `sin`, `tri` | absolute value, a sine, a triangle |
| `mirror lo hi` | folds the value back into the range |
| `pulse w`, `quantize q` | a pulse of width w; steps of size q |
| `clip lo hi`, `softclip a` | cut off, or rounded off |
| `cheby 1:1 3:0.3` | Chebyshev harmonics at levels |
| `table sine 256` | reads a wave table, or your own points |
| `pow p`, `tanh d` | a power curve, a tanh curve |
| `par(a, b, ...)` | each chain on x, the results added |
| `fold g`, `trifold g`, `tube drive bias` | wave folders; an asymmetric tanh |
| `halfwave`, `diode thr knee`, `deadzone w`, `softdead w` | rectifiers and dead zones |
| `curve t`, `ucurve t`, `smoothstep e0 e1`, `smootherstep e0 e1` | bends and S curves |
| `czsaw d`, `czsquare d w`, `czpulse d`, `czres d h` | Casio CZ phase distortion |
| `hann`, `hamming`, `blackman`, `welch`, `tukey a`, `gauss s`, `kaiser b` | windows over 0..1 |
| `harm 1:1 3:0.4`, `poly c0 c1...`, `legendre n`, `funnel v1 v2...` | sums of sines, polynomials, snapping to values |

Numbers may be macros (`m1` to `m4`, or `m1[1,5]` mapped to 1..5). `#` starts a comment.

## Presets

Presets are chosen when the module is added and in its editor; choosing one also puts its chains in the four boxes. A few names: `acid`, `supersawLead`, `fmBass`, `bellMorph`, `warmPad`, `ePiano`, `kalimba`, `laser`, `riser`. There are 86 in ten families.

## Settings

| Setting | What it does |
|---|---|
| `morph`, `macro1`-`macro4` | where in the chains, and the values of m1 to m4 |
| `morphAmount`, `shapeAmount` | how far the morph envelope and the m1 envelope move them (-1 to 1) |
| `octave1`, `detune1`, `level1`, `morph1` | line 1 (octave, cents, level, morph offset) |
| `octave2`, `detune2`, `level2`, `morph2` | line 2 (off at level 0) |
| `unison`, `spread`, `uniStereo`, `drift` | copies, their spread in cents, stereo width, slow wander |
| `sub`, `noise`, `noiseColor` | a sine an octave down; noise, dark to bright |
| `xmod`, `xmodType`, `xmodSilent` | line 1 on line 2: FM (0) or ring (1) |
| `oversample` | 0 auto, 1 off, 2, 4, 8 times |
| `pitchAmount` | an envelope bends the pitch, up to this many semitones |
| `glide`, `width`, `level`, `trim` | portamento (ms), stereo width, output level, dB |
| `vibratoDepth`, `vibratoRate`, `lfo1Rate`, `lfo2Rate`, `lfo2Shape` | the LFOs |
| `filterType`, `cutoff`, `resonance`, `filterEnvAmount`, `keyTrack`, `filterDrive` | the voice filter |
| `modNSrc`, `modNDest`, `modNAmount` | the 8 slot modulation matrix, to morph, macros, cutoff, xmod... |
| `ampAttack`, `ampDecay`, `ampSustain`, `ampRelease` | the amp envelope (also `morphEnv...`, `shapeEnv...`, `pitchEnv...`, `filter...`) |

Related: [dx](dx.md) for classic FM, [modal](modal.md), [waveguide](waveguide.md) and [piano](piano.md) for physical models, [lane](lane.md) to move the morph through a song.
