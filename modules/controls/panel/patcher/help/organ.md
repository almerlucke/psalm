# organ

A **tonewheel organ**: 91 wheels spinning all the time, which the keys tap at the pitches of the **nine drawbars**. It is a whole instrument rather than a set of independent voices: keys that share a wheel share its phase, the percussion sounds on the first key of a phrase only, and the vibrato scanner moves the whole sound. It has the key click, a preamp that can be driven, and a built-in Leslie rotary speaker. Its `notes` input takes notes; the audio output goes on to an effect or the output.

## Tutorial

1. Add an organ. Choose a **preset** when you add it and in the editor (the module's window): `jimmySmith`, `gospelFast`, `bluesGrit`, `churchPlenum`, `softFlute`, `chorusV1`, `wheelLeslie`, `percOrgan`. The names in the presets tell the registration (888000000 is the drawbars). Cable a keyboard, [voice](voice.md) or [part](part.md) to `notes`.
2. Play a chord. The nine **drawbar1** to **drawbar9** go from 0 (out) to 8 (full), from the 16' sub, the 5 1/3', the 8' fundamental up to the 1' at the top; each step is 3 dB. Organists write them as nine digits: `888000000` is the 16', 5 1/3' and 8' all out.

```example Full gospel registration
[set]
drawbar1 = 8
drawbar2 = 8
drawbar3 = 8
drawbar4 = 8
drawbar5 = 0
drawbar6 = 0
drawbar7 = 0
drawbar8 = 0
drawbar9 = 0
percussion = 0      # off
vibrato = 5         # C2
leslie = 1
leslieSpeed = 0.3
```

3. **percussion** adds a short, decaying harmonic on the first key of a phrase: `second` (4') or `third` (2 2/3', the bite of the jazz organ). **percFast** shortens the decay, **percSoft** makes it quieter.

```example A jazz organ with third percussion
[set]
drawbar1 = 8
drawbar2 = 8
drawbar3 = 8
drawbar4 = 0
drawbar5 = 0
drawbar6 = 0
drawbar7 = 0
drawbar8 = 0
drawbar9 = 0
percussion = 2      # third
percFast = 1
percSoft = 0
vibrato = 0
leslie = 0.8
leslieSpeed = 0.4
click = 0.3
drive = 0.1
```

4. **vibrato** chooses the scanner: `V1` to `V3` is pure vibrato, deeper each step, and `C1` to `C3` is chorus (the vibrato mixed with the dry sound, the classic organ shimmer). **vibratoHz** is its speed.
5. **leslie** is how much of the rotary speaker is in the sound; **leslieSpeed** goes from slow (the chorale) to fast (the tremolo), and the mod wheel can speed it up when sent through the matrix. **drive** pushes the preamp into saturation, **tone** is the brightness and **click** the contacts of the keys.

```example A soft flute registration
[set]
drawbar1 = 0
drawbar2 = 0
drawbar3 = 4
drawbar4 = 8
drawbar5 = 4
drawbar6 = 0
drawbar7 = 0
drawbar8 = 0
drawbar9 = 0
percussion = 0
vibrato = 2         # V2
leslie = 0
click = 0.1
```

6. A low pass **filter** (**filterType**, **cutoff**) can tame the top, and the modulation matrix (8 slots) lets the wheel, velocity or an LFO move the drive, the click, the cutoff and more.

## Settings

| Setting | What it does |
|---|---|
| `drawbar1` ... `drawbar9` | the nine drawbars, 0 to 8 |
| `percussion`, `percFast`, `percSoft` | off, 4' or 2 2/3'; fast decay; softer |
| `vibrato`, `vibratoHz`, `vibratoScale` | the scanner type, its speed and depth |
| `tonewheel`, `contactSpread`, `leakage` | the character of the wheel generator |
| `leslie`, `leslieSpeed`, `lesliePosition` | the rotary speaker: amount, speed, microphone position |
| `click`, `clickRand` | key contact click and its randomness |
| `drive`, `tone` | preamp saturation, brightness |
| `filterType`, `cutoff`, `resonance` | an optional filter |
| `mod1Src` ... `mod8Amount` | the modulation matrix |
| `level` | output level |

Related: [subtractive](subtractive.md), [voice](voice.md), [part](part.md).
