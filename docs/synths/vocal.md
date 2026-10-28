# vocal

`synths/vocal`

Sung vowels: a band limited buzz like the vocal folds, shaped by the five formants of a bass,
tenor, alto or soprano (after the formant table of Csound), with breath, vibrato and the small
drift of a real voice. A choir is a voice sung by several singers at once, each a little out of
tune and time, spread over the stereo field.

```go
choir := vocal.NewPoly(vocal.Choir(), 8).AddTo(p)
choir.Receive(psalm.Message{"vowel": 4.0}, 0) // all sing "u", gliding there
```

| Setting | Meaning |
|---|---|
| `Range` | `Auto` (by the pitch of the note), `Bass`, `Tenor`, `Alto`, `Soprano` |
| `Vowel` | 0-4: a e i o u, fractions morph |
| `VowelRate`, `VowelSync`, `VowelDepth`, `VowelSteps` | The vowel moving by itself: Hz or beats a cycle, steps up and down; `VowelSteps` jumps to a new vowel every cycle (talking) |
| `VowelGlide` | Seconds a change of vowel takes |
| `Brightness` | 0-1, breathy to pressed; the velocity adds |
| `Breath` | 0-1, the noise of the air |
| `Attack`, `Release` | Seconds |
| `VibratoRate`, `VibratoDepth`, `VibratoDelay` | Hz, semitones, seconds |
| `Singers`, `Detune`, `Spread` | Voices per note, cents they spread over, 0-1 how wide |
| `Steady` | The pitch held perfectly still |

The first formant follows the pitch up when it is higher, as singers do. **Value inputs**: 1 vowel,
2 vibrato depth.

| Preset | Character |
|---|---|
| `solo` | One voice, natural vibrato |
| `choir`, `aah`, `ooh` | Sections on "a" and "u" |
| `breathy` | Close and airy |
| `monks` | Low men's voices on "o", no vibrato |
| `angels` | High voices, wide and airy |
| `robot` | Steady and bright |
| `talkbox` | A new vowel every sixteenth note, with the tempo: talking in rhythm |
| `wahChoir` | A choir sweeping from "a" to "u" and back |
| `throatDrone` | Sixteen low voices far apart in pitch: a huge drone |
| `whisper` | All air, no tone |
| `alien` | A soprano's throat on every note, a fast wide vibrato |
| `babble` | A crowd of twelve, each jumping between vowels |

**Messages**: `pitch`, `frequency`, `velocity`, `level`, `vowel`, `vowelGlide`, `brightness`, `breath`, `range`, `vibratoRate`, `vibratoDepth`, and the [pitch bend, mod wheel and aftertouch](../../synths.md#pitch-bend-mod-wheel-and-aftertouch).

## Filter, matrix and consonants

`Config.Filter` is the [voice filter](../../synths.md#the-voice-filter) after the formants; `Config.Mods` an 8 slot
[modulation matrix](../subtractive.md#modulation-matrix). Destinations: `vowel`, `brightness`, `breath`,
`vibratoDepth`, `detune`, `spread`, `pitch`, `level`, `cutoff`. Sources: the envelopes, lfo 1 (the
vibrato), lfo 2 (a plain sine at 2 Hz), velocity, key, random, wheel and ext 1-4 (value inputs 3 to
6, `vocal.InExt1`).

`consonant` (none, t, k, s, sh, f) with `consonantLevel` adds a short burst of filtered noise as the
note starts: a clipped click for t and k, a swell of hiss for s, sh and f. It sits in the middle and
is not shaped by the vowel or the filter, which makes syllables of the notes. The default is none.

## Words and formant shift

The vocal synth can sing a word. Set `Word` (or send `word: "ta ke so"`) and each new note takes the
next syllable and loops at the end. A syllable is a consonant (`t k s sh f`, or none) plus a vowel
(`a e i o u`); `vocal.ParseWord` turns text into syllables. With `WordSync` (`wordSync`) beats per
syllable, a held note goes on to the next syllables on the tempo. In a Poly the voices share one place
in the word, so a note sings the next syllable whichever voice it lands on. `syllable` (alias
`wordStep`) sets the next syllable to sing. `formantShift` scales every formant: below 1 is a bigger
throat (a giant), above 1 a smaller one (a child). Presets: `chant`, `giant` and `child`.

## A glottal source, a vocal tract, twelve vowels

The voice sings with a glottal source: a Rosenberg-Klatt flow pulse whose open quotient, speed
quotient and return phase come from `brightness` (breathy, modal, pressed) or are set directly
(`openQuotient`, `speedQuotient`, `returnPhase`). `jitter` (cycle length) and `shimmer` (cycle
loudness) are off until set, and the breath noise swells with the open folds. The pulse runs through a
cascade of five formants from Peterson-Barney data for `bass`, `tenor`, `alto`, `soprano` and `child`
voices (`Range`). Sing the old scale with `vowel` (0-4: a e i o u) or pick one of twelve vowels
(i ɪ e ɛ æ a ɔ o ʊ u ʌ ɜ) with `phoneme`. `phonemeTo`, `diphthongTime` and `diphthongHold` glide to a
second vowel inside a note (`diphthong` sets a named pair). `formantTuning` lets F1 follow a high
pitch up, as a soprano's does; `training` gathers F3-F5 near 3 kHz (the singer's formant); `nasal`
opens the nose.

Words take `m n ng l r w y` as well as `t k s sh f`, and the diphthongs `ai au ei ou oi`, so
"mai la ro ne yu" speaks (the `speak` preset). Each consonant starts the formants at its locus, a nasal
also with a pole and zero pair, and they glide into the vowel over `coarticulation` seconds. The
glottal model is the Rosenberg-Klatt approximation, not a true LF fit, and bass, alto and soprano are
scaled from the male and female data. All presets are re-voiced with a little jitter and shimmer.

## The LF source, whisper and more consonants

`source` chooses the glottal model: 0 is the Rosenberg-Klatt pulse, 1 the Liljencrants-Fant model, with
one voice-quality parameter `rd` from 0.3 (pressed, bright) to 2.7 (breathy, soft); brightness sets it
unless `rd` is set. Rd gives tp, te and ta, and the growth rate is solved so the flow derivative
integrates to zero. The presets Breathy, Ooh, Monks and Angels use LF; the others keep Rosenberg-Klatt.
`whisper` (0-1) cross-fades the voice into turbulence at the glottis, shaped by the same formants, with
independent noise per ear; at 1 it has no pitch (the `Whisper` preset).

p, t and k are aspirated and keep the voice out for 50 to 65 ms; b, d and g start it within 15 ms; s, sh
and f hold it back through the noise; z, zh and v hum through it with a voice bar (new letters
p b d g z zh v). `subglottal` adds a faint windpipe pole and zero at 600 and 1500 Hz. `naturalVibrato`
makes the vibrato rate 5 to 6.5 Hz, rising with pitch and wandering slowly, with a wandering extent
and a late onset. The new features use a private noise stream.
