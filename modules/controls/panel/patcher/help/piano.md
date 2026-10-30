# piano

A **physically modelled piano**, no samples. Every key has one to three strings, each a travelling wave on a string that loses energy and is slightly stiff; a felt hammer is simulated hitting them, and the sound goes through a soundboard. It is played polyphonically from `notes`, with one audio output.

What makes it sound like a piano rather than a plucked string:

- a note falls fast and then lingers on a long, quiet aftersound, as the strings trade energy through the bridge;
- a harder key is shorter, sharper and brighter, a soft one round;
- the dampers, the key and (if you send it) the sustain pedal are heard, and strings of keys that are not played ring along (sympathetic resonance);
- the octaves are stretch-tuned like a tuner does it, and no two keys are exactly alike.

## Tutorial

1. Add a keyboard (or a [noteroll](noteroll.md), a [voice](voice.md) or a [part](part.md)), a piano and a speaker. Cable the notes to the piano.
2. Play soft and hard. The velocity makes the tone brighter, and the level louder by `velocitySens`.
3. Pick another preset when you add it, or in the piano's editor (the window of the module). `grand`-style presets are warm and close; `upright` is boxier; `honkyTonk` has strings far out of tune with each other.
4. Change the **hammer**: `hardness` goes from soft felt (round, dark) to hard (bright, pop); `strikePosition` is where on the string it hits (1/8 is classic, nearer the end is thinner):

```example A bright pop piano
[set]
hardness = 0.75
strikePosition = 0.1
treble = 2     # dB for the top octave
```

```example A soft, dark piano for quiet music
[set]
hardness = 0.1
velocitySens = 0.85
width = 0.3
decay = 1.2
```

5. Give it age and character: **unison** is how many cents the strings of one key are apart (around 1 is in tune with a slow beat, 10 or more is honky tonk), **stiffness** is the inharmonicity (more is an upright or a toy), **stretch** the tuner's stretch:

```example Honky tonk
[set]
unison = 12
variation = 0.9
hardness = 0.6
```

6. The room: `position` is where you hear it from (0 player, 1 studio, 2 hall), `soundboard` and `body` the wood, `resonance` the strings ringing along, `mechanics` and `keyBed` the noise of the action.

```example In a hall, with the strings ringing along
[set]
position = 2
resonance = 0.8
soundboard = 0.9
body = 0.8
```

7. The pedals are not settings but messages: a [lane](lane.md) or a script can send `sustain` (0 up, 1 down, in between is half pedal), `sostenuto` and `soft` to the piano.

A piano is calibrated to be about the same loudness for the presets at velocity 0.8; played harder it can go over, so leave headroom in a mixer.

## Presets

Presets are chosen when the module is added, and again in its editor. Some names: `baby` (a small grand, the first), `steinway`, `bosendorfer`, `upright`, `honkyTonk`, `felt`, `cinematic`, `toy`, `ragtime`, `jazzDry`.

## Settings

| Setting | What it does |
|---|---|
| `hardness`, `strikePosition`, `velocitySens`, `variation`, `strikeRegister` | the hammer: felt, where it hits, how much velocity changes level, how different keys and strikes are |
| `stiffness`, `decay`, `damping`, `stretch` | the strings: inharmonicity, ring time, highs dying faster, tuner's stretch |
| `unison`, `strings` | cents between the strings of a key; strings per key (1-3) |
| `bark`, `wound`, `precursor` | the bark of hard bass notes, wound bass strings, the burst before a hard bass note |
| `resonance`, `damperTime` | strings ringing along; seconds a released key rings on |
| `soundboard`, `body`, `bridge` | the knock of the board, how much sound goes through it, how the strings share the bridge |
| `mechanics`, `keyBed` | noise of the action |
| `position`, `width`, `treble` | player, studio or hall; stereo spread of the keys; top octave in dB |
| `level` | output level |

Related: [waveguide](waveguide.md) (plucked and bowed strings), [modal](modal.md) (struck bars and bells), [dx](dx.md) (FM electric pianos), [part](part.md) to play chords with it.
