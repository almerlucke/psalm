# waveguide

Instruments made from a **wave travelling up and down** a string or a tube: plucked and bowed strings, drum skins and blown pipes (flute, clarinet). No samples; the wave is fed by a pluck, a bow, a strike or a breath, and heard through the body of the instrument. It plays from `notes`, polyphonic, with one audio output.

There are four models, chosen by the preset:

- **pluck**: a plucked string (guitar, harp, bass, banjo): how long it rings, how fast the highs die, how hard and where it is plucked;
- **bow**: a bowed string (violin, cello, glass harp): the bow sticks and slips and keeps the string going while the note is held;
- **membrane**: a struck drum skin (tom, conga, tabla);
- **pipe**: a blown tube (flute, clarinet, panpipe), where `breath` is the pressure of the air.

The settings you see in the module depend on the model of the preset you have chosen: a flute has `breath` and `embouchure`, a string has decay and pick position, a bow has pressure and bow position, a drum has strike and tension. The examples below are for the **first preset, `altoFlute`**, a pipe, because that is what a new waveguide starts as.

## Tutorial

1. Add a keyboard (or a [voice](voice.md)) and a waveguide and cable `notes` to it. You hear a flute; hold a key and it breathes in with a slow attack (`ampAttack`).
2. Vary the air. **breath** is the pressure: too little and it does not speak, too much overblows. **noise** is the hiss of the air, **embouchure** the offset of the air jet (it brings in even partials), **damping** darkens the tube.

```example A breathy, hissy flute
[set]
breath = 0.55
noise = 0.5
embouchure = 0.4
damping = 0.5
```

3. Turn on **reed**: a single reed on a closed tube, which gives odd partials and sounds an octave lower, like a clarinet:

```example A clarinet-like reed
[set]
reed = 1
breath = 0.6
noise = 0.05
embouchure = 0.5
```

4. A late **vibrato** and a slow attack for a singing line:

```example A slow, expressive lead
[set]
ampAttack = 350    # ms
ampRelease = 400
vibratoRate = 5
vibratoDepth = 0.2  # semitones
```

5. **sympathetic** feeds the sound into four strings tuned an octave below, a fifth, an octave and an octave and a fifth above, which shimmer along; **bodyMix** blends in the body of the instrument:

```example Shimmer
[set]
sympathetic = 0.5
bodyMix = 0.4
```

6. Now pick another preset to hear the other models and open the module: a pluck is mainly `decay`, `damping`, `brightness` (soft finger to hard pick) and `pickPosition`; a bow has `bowPressure` and `bowPosition`. Real-string extras (`inharmonicity`, `polarization`, `bodyCoupling`, `fretBuzz`, `harmonic`, `slide`) are in the editor of the string presets.

The mod wheel, pitch bend and aftertouch work as on the other synths. Like them, waveguide has the voice filter and an 8 slot modulation matrix (with destinations such as `breath`, `embouchure`, `brightness`, `decay`).

## Presets

Presets are chosen when the module is added, and again in its editor. Some names: `altoFlute` (the first), `flute`, `clarinet`, `guitar`, `harp`, `cello`, `koto`, `conga`, `djembe`, `glassHarp`, `eBow`.

## Settings (of a pipe)

| Setting | What it does |
|---|---|
| `breath` | 0-1, pressure of the air |
| `noise` | 0-1, the hiss of the air |
| `embouchure` | 0-1, offset of the jet (flute), stiffness of the reed |
| `reed` | 1: a clarinet reed instead of a flute jet |
| `damping` | 0-1, darkens the tube |
| `vibratoRate`, `vibratoDepth` | Hz and semitones |
| `bodyMix`, `sympathetic` | the body of the instrument; strings ringing along |
| `level` | output level |
| `ampAttack`, `ampRelease` | breath envelope, ms |
| `filterType`, `cutoff`, `resonance`, `filterEnvAmount`, `keyTrack`, `filterDrive` | the voice filter |
| `modNSrc`, `modNDest`, `modNAmount` | modulation matrix slot N (1-8) |

Related: [piano](piano.md), [modal](modal.md), [vocal](vocal.md), [shaper](shaper.md).
