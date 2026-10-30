# vocal

A **singing voice**: a buzz like the vocal folds, shaped by the resonances (formants) of a mouth for bass, tenor, alto, soprano or child voices, with breath, vibrato and the small unsteadiness of a real singer. A choir is the same voice sung by several singers at once, each a little out of tune and time, spread over the stereo field. It plays from `notes` (polyphonic) and has one audio output.

## Tutorial

1. Add a keyboard (or a [voice](voice.md), a [part](part.md) or a [melodist](melodist.md)), a vocal and a speaker. Cable `notes` to the vocal. The first preset is `aah`, a section singing "a".
2. Choose who sings with **range**: `auto` picks by the pitch of the note, or force `bass`, `tenor`, `alto`, `soprano` or `child`. **formantShift** scales every formant: below 1 is a bigger throat (a giant), above 1 a smaller one (a child).
3. Choose what is sung. **vowel** runs from 0 to 4 over a, e, i, o, u and fractions in between morph from one to the next; **vowelGlide** is the seconds a change takes. Change it while you hold a note:

```example Sing "oo", gliding there slowly
[set]
vowel = 4
vowelGlide = 1.2
```

4. Or pick one of twelve exact vowels with **phoneme** (0 uses `vowel`; then i, ɪ, e, ɛ, æ, a, ɔ, o, ʊ, u, ʌ, ɜ). **phonemeTo**, **diphthongTime** and **diphthongHold** glide to a second vowel inside the note:

```example A diphthong, "a" to "i" inside the note
[set]
phoneme = 6      # a
phonemeTo = 1    # i
diphthongTime = 0.4
diphthongHold = 0.2
```

5. **brightness** goes from breathy to pressed, **breath** is the noise of the air, **whisper** at 1 is all air and no pitch. **vibratoRate**, **vibratoDepth** and **naturalVibrato** (rate and extent wander as a singer's do) make it human; **jitter** and **shimmer** add small unsteadiness in pitch and loudness.

```example A close, breathy solo
[set]
brightness = 0.25
breath = 0.3
naturalVibrato = 1
vibratoDepth = 0.2
range = 3        # alto
```

6. **consonant** adds a short burst at the start of each note, so notes become syllables: `t`, `k`, `s`, `sh`, `f`, `m`, `n`, `l`, `r`, `p`, `b`... **consonantLevel** is how strong; **coarticulation** is how long it takes to glide into the vowel.

```example Notes that start with a "t"
[set]
consonant = 1    # t
consonantLevel = 0.6
coarticulation = 0.03
```

7. Features for the advanced: `source` chooses the glottal model (`rosenberg-klatt` or `lf`, where `rd` 0.3 is pressed and bright and up to 2.7 is soft and breathy), `openQuotient`, `speedQuotient` and `returnPhase` set the pulse by hand, `formantTuning` lets the first formant follow a high pitch up as a soprano's does, `training` is the singer's formant near 3 kHz, and `nasal` opens the nose.

## Presets

Presets are chosen when the module is added, and again in its editor. Some names: `aah`, `choir`, `ooh`, `solo`, `monks`, `angels`, `robot`, `talkbox`, `whisper`, `giant`, `child`, `throatDrone`.

## Settings

| Setting | What it does |
|---|---|
| `range` | auto, bass, tenor, alto, soprano, child |
| `vowel`, `vowelGlide` | 0-4 over a e i o u; seconds a change takes |
| `phoneme`, `phonemeTo`, `diphthongTime`, `diphthongHold` | twelve exact vowels, a second one to glide to |
| `formantShift`, `formantTuning`, `training`, `nasal` | size of the throat; first formant follows high pitch; singer's formant; nose |
| `brightness`, `breath`, `whisper` | breathy to pressed; air; all air |
| `source`, `rd`, `openQuotient`, `speedQuotient`, `returnPhase`, `subglottal` | the glottal source |
| `jitter`, `shimmer` | pitch and loudness unsteadiness |
| `vibratoRate`, `vibratoDepth`, `naturalVibrato` | vibrato |
| `consonant`, `consonantLevel`, `coarticulation` | a burst at the note start |
| `wordSync` | beats per syllable when a word is sung |
| `level` | output level |
| `ampAttack`, `ampRelease` | envelope, ms |
| `filterType`, `cutoff`, `resonance`, `filterEnvAmount`, `keyTrack`, `filterDrive` | the voice filter |
| `modNSrc`, `modNDest`, `modNAmount` | modulation matrix slot N (1-8), for example the mod wheel to `vowel` |

Related: [waveguide](waveguide.md) for the wind and string sounds, [shaper](shaper.md) (it has vowel and choir presets), [melodist](melodist.md) to make it sing a melody.
