# shaper

`synths/shaper`, the chain language in `blocks/shapechain`

A synth that builds its wave from a **chain of shapers** (genny's `float/shape`): a phase from 0 to 1
goes in, the chain bends it into a wave. Up to four chains, A to D, are written as text, and the
**morph** (0-1) goes through them: 0 is chain A, 1 the last chain, in between a blend of two
neighbours. An envelope, an LFO, the velocity or the wheel can pull the morph, so the wave itself
moves. Two lines (octave, detune, level, a morph offset of their own) read the same chains, then go
through the [voice filter](../../synths.md#the-voice-filter) of every synth and an amp envelope.

```go
cfg := shaper.Glass()
cfg.Chains[1] = "bipolar | mirror -0.4 0.4 | mult 2.5 | sin" // chain B
v := shaper.NewPoly(cfg, 8).AddTo(p)
v.ReceiveMessage(psalm.Message{"chainC": "par(sin, mult 3 | sin | mult 0.4)", "morph": 0.7})
```

**A chain** is stages joined by `|`: `bipolar` is a saw, `sin` a sine, `mult 3 | sin` a sine of three
cycles, `bipolar | mirror -0.5 0.5` a saw folded back, `par(sin, mult 3 | sin | mult 0.4)` two
chains side by side with their results added. The stages are genny's shapers by the same names, and
a test holds them to genny's:

| Stage | x becomes |
|---|---|
| `thru`, `add a`, `mult m`, `linear scale shift` | `x`, `x + a`, `x * m`, `x * scale + shift` |
| `mod m`, `ripple m` | the remainder of `x / m` (`m` not 0); `x` plus it |
| `bipolar`, `unipolar` | 0..1 to -1..1, and back |
| `abs`, `sin`, `tri` | `abs(x)`; `sin(2πx)`; a triangle over 0..1 |
| `mirror lo hi` | folded back into the range, as a ball bounces (0 1 to start) |
| `pulse w`, `quantize q` | 1 below `w`, else 0; the nearest multiple of `q` |
| `clip lo hi`, `softclip a` | cut off at the edges (-1 1 to start); an arctangent curve |
| `cheby 1:1 3:0.3` | Chebyshev harmonics (1 to 64) at their levels |
| `table sine 256`, `table 0 1 0.5 -1` | a wave (`sine`, `cosine`, `sawtooth`, `triangle`, `square`, and `uni-` for 0..1) read over 0..1, or your points with straight lines between them |
| `pow p`, `tanh d` | `abs(x)^p` with the sign of `x`; `tanh(d x) / tanh(d)` (these two are not genny's) |
| `par(a, b, ...)` | each chain on `x`, the results added |

**FM** is a chain too: `par(thru, mult 3 | sin | mult 0.3) | sin` is a sine at the pitch whose phase a sine at three times the pitch
moves by 0.3 of a turn (about 1.9 radians): phase modulation, the sound of a DX. The ratio is a whole number (`mult 1`, `2`, `3`,
`4`...: the wave must come round once a cycle), the index a number, or a macro the envelope pulls down over the note
(`mult m1[0,0.4]`): an electric piano, a bell, a bass.

A number in a stage can be a **macro**: `m1` to `m4`, a value from 0 to 1 that the sound moves while it
plays (`macro1`...`macro4`, the matrix, the `shapeEnv` envelope on `m1`). `m1` is 0..1, `m1[1,5]` maps
it to 1..5, and `m1[5,1]` upside down. `mult m1[1,6] | mod 1 | bipolar` is a hard synced saw whose
sync ratio the macro sweeps. Numbers are separated by spaces or commas, new lines count as spaces,
and `#` starts a comment. A chain that can't be read is an error with its column (`shapechain.Parse`);
the synth keeps the chain it had.

| Setting | |
|---|---|
| `Chains` | the text of chains A to D; the list ends at the first empty one; chain A is a sine when empty |
| `Morph`, `Macros` | 0-1, where in the chains it is; the values of `m1` to `m4` |
| `Line1`, `Line2` | `Octave`, `Detune` (cents), `Level` (line 2 is off at 0), `Morph` (-1 to 1, added to the morph) |
| `MorphEnv`, `MorphAmount` | an envelope over the note that moves the morph by up to `MorphAmount` (-1 to 1); off at 0 |
| `ShapeEnv`, `ShapeAmount` | the same on macro `m1` |
| `AmpEnv`, `Width`, `Level` | the amp envelope; line 1 left and line 2 right; output level |
| `PitchEnv`, `PitchAmount` | an envelope that bends the pitch over the note by up to `PitchAmount` semitones (-60 to 60): a kick that falls three octaves, a laser, a riser; off at 0 |
| `Glide`, `Mono`, `Legato` | milliseconds a note takes to slide from the pitch the voice was at; with `Mono` (and `Legato`) `NewPoly` plays one note at a time, so a bass or lead slides from note to note (`set({mono: 1, legato: 1})` from a script does it too) |
| `Unison`, `Spread`, `UniStereo` | copies of each line (1 to 7, taken when a note starts on a silent voice; the level stays the same), the outer ones `Spread` cents (0-100) under and over the pitch, and spread across the stereo field by `UniStereo` (0-1). Each copy costs as much as a line, so a pad of eight voices at 3 copies is six times the work of one; keep `Oversample` at 1 there |
| `Drift` | 0-1: every copy, the morph and the cutoff wander slowly and at random (a few cents, a few hundredths of the morph), as the oscillators of an analog synth do, so chords are never the same twice |
| `Sub`, `Noise`, `NoiseColor` | a sine an octave under the note (0-1); white noise (0-1) made dark (under 0) or bright (over 0) by `NoiseColor` (-1 to 1): breath in a flute, air on a pad, grit on a bass. Both go through the voice filter |
| `XMod`, `XModType`, `XModSilent` | line 1 acting on line 2 (0 is off): type 0 bends the phase of line 2 by the wave of line 1 (FM: the amount is the index, up to a whole turn, and the octave and detune of the lines are the ratio, 3.5 for a bell with `Octave: 1, Detune: 968`), type 1 multiplies the two (a ring modulator, the amount blends it in). `XModSilent` hides line 1: it only modulates. `xmod` is also a destination of the matrix (an envelope that dies with the note makes a bell, an LFO a growl) |
| `Oversample` | how many times a sample the chains are read: 0 (the start) is automatic (see below), or 1 (off), 2, 4, 8 |
| `VibratoRate/Depth/Delay`, `LFO1Rate`, `LFO1Sync`, `LFO2Rate`, `LFO2Shape` | a vibrato and the two LFOs of the matrix |
| `Filter`, `Mods`, `Ext` | the [voice filter](../../synths.md#the-voice-filter) (off at first) and the [matrix](../subtractive.md#modulation-matrix): filter env, amp env, lfo 1, lfo 2, velocity, key, random, wheel, Ext 1-4 to `morph`, `macro1`-`macro4`, `pitch`, `vibratoDepth`, `width`, `level`, `detune2`, `cutoff`, `resonance`, `xmod` |

**Aliasing and cost.** Folds, `mod`, `quantize` and steep curves make harmonics far above the pitch. By default
(`Oversample` 0, "auto") the synth works out what it needs: `Chain.Bandwidth()` looks at the spectrum of each chain (with
each macro it uses at 0, half and 1) and finds the highest harmonic that has a say, and from that and the pitch of the
note (and its lines) the voice reads the chain once a sample when everything stays under half the sample rate (a sine, a
few partials: the organ, the vowels and the metal presets). A chain with more than that, a fold or a sharp edge (a saw, a
pulse, steps, hard sync: harmonics without an end), is read from **tables**: one cycle of the wave of each line is worked
out (the chains at the morph and the macros it has now), cut with an FFT to the harmonics that fit under half the sample
rate at the pitch, and played by a cubic reading, which is a few multiplications. All the copies of a unison read the same
table, so a unison with a narrow chain uses one too. When the morph or a macro has moved a little (0.01) a new table is
made and the sound fades from the old one to it, so a sweep or an LFO is a series of small steps no one hears (a line makes a table at most once every half table of samples, so a fast envelope on a macro costs a couple of chain evaluations a sample and not more, and the fades between the tables keep it smooth); a chain
swapped under a note fades over the twelve milliseconds it always did; a pitch that rises until the table would fold back,
or falls far enough to make it dull, gets a new one. A table has no aliasing and keeps the harmonics of the chain to a
tenth of a decibel (a test compares them with the wave read eight times a sample), and costs a fifth of reading it
four times: `supersawLead` costs 0.3 microseconds a sample against 2.0 with 4 times oversampling. Set `Oversample` yourself to
fix it: 1 reads the chain once a sample whatever it is (cheap, and it folds), 2, 4 and 8 read it that many times and low-pass
the result (`blocks/oversample`; 8 costs twice what 4 does). A voice also stops working out a line once its amp envelope
is below -100 dB, the end of a long release. The cross modulation does not take care of what it makes above half the sample
rate: keep the index moderate for high notes.

**Presets by message.** The message `preset` makes a voice a preset by name (`set({preset: "reese"})` from a script,
`shaper.NewPoly(...).ReceiveMessage(psalm.Message{"preset": "reese"})`): every setting of it, the chains, envelopes, lines, filter,
matrix and LFOs; the keys after it in the same message then change it (`{preset: "acid", cutoff: 600}`). What is not a preset's
is left as it is: `trim` (dB, where it sits in a mix; the presets are all about the same loudness), and the mono play of
the synth (`mono`, `legato`, of the poly: `acid` and the flute need them). A sounding voice goes on: the chains are faded to, the
levels glide, a filter that comes is faded in and one that goes is left on, wide open.

A note that starts on a voice still sounding (the next note of a mono synth) carries on from it: the old note sounds until the
sample the new one starts at, and the wave, the envelope and the settings continue.

**Changes under a sounding note.** A script (or a panel) can change `morph`, the macros, the levels and the width at any
moment, as often as every step. They glide to the new value in 5 ms, so a change never makes a click, and a chain swapped for
another is faded to over 12 ms while the old one fades out (a test changes each under a note and holds the second difference of
the wave to a few thousandths). What cannot glide is the type of the filter (off, low pass, ladder): changing it under a note
steps; keep a filter on, wide open, instead of off.

A lopsided wave (a narrow pulse sits mostly on one side of zero) has its mean taken off before the envelope, worked out exactly at
the start of each note and followed slowly after; otherwise the envelope would switch the offset on and off with every note, a thump
that a small speaker plays as a tick.

**Messages**: `preset`, `trim`, `pitch`, `frequency`, `velocity`, `glide`, `pitchAmount`, `morph`, `macro1`-`macro4`, `chainA`-`chainD` (text, or a
`*shapechain.Chain` read already, which is what the patcher sends), `octave1/2`, `detune1/2`,
`level1/2`, `morph1/2`, `morphAmount`, `shapeAmount`, `oversample` (0-4: auto, off, 2x, 4x, 8x), `width`,
`unison` (1-7), `spread`, `uniStereo`, `drift`, `sub`, `noise`, `noiseColor`,
`level`, `xmod`, `xmodType`, `xmodSilent`, `vibratoDepth`, `vibratoRate`, `lfo1Rate`, `lfo2Rate`, `lfo2Shape`, the envelopes `amp`,
`morphEnv`, `shapeEnv`, `pitchEnv` and `filter`, the filter and the matrix, and the [pitch bend, mod wheel and
aftertouch](../../synths.md#pitch-bend-mod-wheel-and-aftertouch). **Value input** 1: the morph; 5 to 8: Ext 1-4.

Presets: 66 in nine families (see [presets](../presets/shaper.md#shaper), and `go run ./cmd/examples/presets -synth shaper`
plays them all): basses (`subSine`, `kickSub`, `reese`, `acid`, `wobble`, `pulseBass`, `fmBass`, `growl`, `subFold`),
leads (`supersawLead`, `squareLead`, `syncLead`, `pluckLead`, `fmLead`, `fluteLead`, `brassLead`, `vowel`, `reed`), pads
(`warmPad`, `stringPad`, `choirPad`, `airPad`, `darkPad`, `evolve`, `glassPad`, `foldPad`, `morphPad`, `swarm`, `organ`), keys and
plucks (`pluck`, `harp`, `kalimba`, `marimba`, `ePiano`, `clav`, `vibes`, `stab`, `musicBox`, `pwm`), bells (`bellMorph`,
`glassBell`, `churchBell`, `gong`, `metal`, `glass`), drones (`drone5`, `subDrone`, `crush`) and effects (`laser`, `riser`,
`impact`, `zap`) cross modulation (`xmodBell`, `xmodGrowl`, `ringMetal`) and beyond, what only this synth does (`rainbow`, `crystalSea`, `fourBeasts`, `trumpetVoice`, `sevenLamps`, `crownsCast`, `thunder`, `lightning`, `lionBass`, `throneDrone`, `manyEyes`). A test plays every one and holds it to the same measures: no clipping, as loud as the others, nothing
shrill, silent after the key is let go, and not like another.

In the patcher the synth is a module of its own (Synths, "shaper"): its four chains are boxes in the
window (Ctrl+Enter takes one, an error shows in red and the sound goes on; choosing a factory
preset puts that preset's chains in the boxes). The same text is a shaper in a script:
`genny.shapers.chain("mult 3 | sin")` (see [scripting](../scripting.md#shapers)).
