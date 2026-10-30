# wavetable

A **wavetable synth**: the oscillator plays one single-cycle frame out of a table and morphs through the frames as the **position** moves from 0 to 1. A different table is a different family of sounds; moving the position (by hand, an envelope, an LFO, the wheel) makes the tone evolve. Unison stacks, a sub, a second oscillator, warp modes and a filter round it out. Its `notes` input takes notes; its audio output goes on to an effect or the output.

## Tutorial

1. Add a wavetable synth. Choose a **preset** when you add it and in the editor (the module's window): `subBass`, `wobbleBass`, `fmBass`, `syncLead`, `fmLead`, `glassPad`, `voxPad`, `harpPluck`, `fmBell`, `drawbarOrgan` and more. Each preset brings its own table or uses the default one. Cable a keyboard, a [voice](voice.md) or a [part](part.md) to `notes`.
2. Play a held note and sweep **position**. That is the heart of the instrument: each place in the table is a different timbre.
3. **positionEnvAmount** lets the filter envelope push the position at every note (the timbre changes as the note sounds); **positionLFORate** and **positionLFODepth** make it sway.

```example A moving, wide pad tone
[set]
position = 0.4
positionLFORate = 0.3
positionLFODepth = 0.3
unison = 5
detune = 18
spread = 0.8
filter = 1          # low pass
cutoff = 3500
resonance = 0.15
filterEnvAmount = 1
glide = 0
```

4. **unison** stacks detuned copies, **detune** (cents) spreads them and **spread** puts them across the stereo field. **subLevel** adds a sine an octave down.
5. **warpMode** bends the shape of each cycle: `bend +`, `bend -`, `sync`, `mirror`, `fm b`, `fm sine`; **warp** is how far. Sync with a high amount is the classic tearing lead.

```example A sync lead
[set]
position = 0.2
unison = 1
subLevel = 0
warpMode = 4        # sync
warp = 0.5
filter = 1
cutoff = 5000
resonance = 0.2
filterEnvAmount = 1.5
glide = 60
vibratoRate = 5.5
vibratoDepth = 0.15
```

6. A second oscillator, **B**, with its own table position, octave, semitone and detune: set **oscBLevel** above 0 to bring it in. A bass from two layers:

```example A layered bass
[set]
position = 0.1
unison = 2
detune = 12
subLevel = 0.5
oscBLevel = 0.4
oscBPosition = 0.5
oscBOctave = -1
filter = 1
cutoff = 400
resonance = 0.3
filterEnvAmount = 3
glide = 40
```

7. The **modulation matrix** has 8 slots (source, destination, amount): the velocity, an LFO, a random value or the mod wheel can move the position, the cutoff, the warp, the detune and more. Envelopes and the table choice are in the editor.

## Settings

| Setting | What it does |
|---|---|
| `position` | where in the table the oscillator plays (0 to 1) |
| `positionEnvAmount` | how far the filter envelope moves the position |
| `positionLFORate`, `positionLFODepth` | an LFO sweeping the position |
| `unison`, `detune`, `spread` | stacked voices, their detune in cents, their stereo width |
| `subLevel` | a sine an octave down |
| `warpMode`, `warp` | how each cycle is bent, and by how much |
| `oscBLevel`, `oscBPosition`, `oscBOctave`, `oscBSemi`, `oscBDetune` | the second oscillator |
| `filter`, `cutoff`, `resonance` | off, low, band or high pass; brightness; peak |
| `filterEnvAmount`, `keyTrack` | octaves the envelope opens the filter; cutoff following the note |
| `vibratoRate`, `vibratoDepth` | pitch vibrato in Hz and semitones |
| `glide` | ms to slide from the last note |
| `mod1Src` ... `mod8Amount` | the modulation matrix |
| `level` | output level |

Related: [subtractive](subtractive.md), [supersaw](supersaw.md), [voice](voice.md).
