# modal

**Struck things**: bars, tubes, bells, bowls, plates and gongs. A struck object rings in several **modes**, partials with their own pitch and their own decay time; the module models these modes and a mallet that hits them. A soft mallet touches long and wakes few high modes, a hard one is bright; where it strikes mutes the modes that have a node there. It plays from `notes` (polyphonic) and has one audio output.

## Tutorial

1. Add a keyboard (or a [voice](voice.md), or a [part](part.md)) and a modal and cable `notes` to it. The first preset, `alienBell`, rings in modes no real bell has.
2. Choose what is struck with **instrument**: `custom` plays the modes of the preset, the others are physical models of a bar, a marimba or vibraphone bar, a xylophone bar, a glockenspiel bar, a tine (kalimba), a tubular bell, a church bell, a handbell, a plate, a gong, a cymbal, a singing bowl, a steel pan, a wood block, a temple block or a bottle.

```example A marimba
[set]
instrument = 2   # marimba
resonator = 1    # the tube under the bar lengthens the fundamental
hardness = 0.5
position = 0.25  # struck near the middle: a round tone
ampDecay = 1500  # ms the note rings
```

3. **hardness** is the mallet (the velocity adds to it); **position** is where it hits along the bar or from the middle of a plate to its rim; hit the node of a mode and it falls silent. **noise** is the knock of the mallet.
4. A vibraphone adds a motor: **tremolo** (Hz) and **tremoloDepth**:

```example A vibraphone with its motor
[set]
instrument = 3   # vibraphone
resonator = 1
hardness = 0.35
tremolo = 5
tremoloDepth = 0.6
ampDecay = 4000
```

5. Bells, bowls and plates have pairs of near-equal modes: **spread** splits them so they beat, and **drift** makes every note a little different.

```example A church bell, beating
[set]
instrument = 8   # church bell
spread = 1.5
drift = 0.3
hardness = 0.8
ampDecay = 8000
```

6. **inharmonicity** stretches the modes upward for stiff metal and glass; **highDamping** makes the high modes die sooner (a dull, wooden sound); **excitation** can also be a bow (`bow`, keeps the modes singing while the key is held) or a noise burst (for brushes and scrapes).

```example Glass, struck and bowed
[set]
excitation = 1   # bow
inharmonicity = 0.4
highDamping = 0.1
ampDecay = 3000
```

7. A gong or cymbal can be given **nonlinearity**, a hard blow starts a little sharp and the energy flows from low modes to high ones, so a crash blooms into a hiss.

## Presets

Presets are chosen when the module is added, and again in its editor. Some names: `marimba`, `vibraphone`, `glockenspiel`, `tubularBells`, `churchBell`, `handbell`, `gong`, `singingBowl`, `kalimba`, `steelPan`, `crystal`, `alienBell`.

## Settings

| Setting | What it does |
|---|---|
| `instrument` | the physical model: custom, bar, marimba, vibraphone, xylophone, glockenspiel, tine, tubular bell, church bell, handbell, plate, gong, cymbal, singing bowl, steel pan, wood block, temple block, bottle |
| `resonator`, `undercut` | the tube under a bar; how much the bar is undercut (tuned arch) |
| `spread`, `drift` | splitting of near-equal mode pairs so they beat; random variation per note |
| `nonlinearity` | gongs and cymbals: a pitch glide and energy moving upward |
| `excitation` | strike, bow or noise burst |
| `hardness`, `position`, `noise` | the mallet: hard or soft, where it hits, its knock |
| `contact`, `mass` | how a harder blow brightens; how heavy the mallet is (longer contact) |
| `highDamping`, `inharmonicity` | highs dying sooner; modes stretched upward |
| `tremolo`, `tremoloDepth` | a motor, Hz and depth |
| `level` | output level |
| `ampDecay` | how long the note rings, ms (also `ampRelease`...) |
| `filterType`, `cutoff`, `resonance`, `filterEnvAmount`, `keyTrack`, `filterDrive` | the voice filter |
| `modNSrc`, `modNDest`, `modNAmount` | modulation matrix slot N (1-8): velocity to `hardness`, an LFO to `pitch`... |

Related: [waveguide](waveguide.md), [piano](piano.md), [dx](dx.md) for FM bells, [shaper](shaper.md).
