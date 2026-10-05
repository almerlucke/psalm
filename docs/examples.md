# Examples

All examples live in `cmd/examples` and play live through PortAudio until you press Ctrl+C.
Run them from the repository root (they load sounds from `assets/`):

```
go run ./cmd/examples/<name>
```

Most have a commented out `p.ToSoundFile(...)` call: uncomment it (and comment out
`p.Listen()`) to render to a file instead.

| Example | What it shows |
|---|---|
| `showcase` | "Aurora", the showcase: a slow orchestral piece in D minor (about 4 minutes) using every synth and effect, no drums. Wavetable violins with chorus and an LFO on the position, subtractive celli, FM pizzicato bass, harp (into freeverb and a granulator that freezes in the interlude), flute with a pitch shifted piccolo shadow, horn solo, celesta (pingpong delay into shimmer) and tubular bells, DX horns and a flangered bell pad lined up with a vocoder by latency, a grains choir and pulsar choir and soprano in a convolved church, pulsar raindrops, a wavetable drone on a generated table. A ladder filter opens the strings at dawn and closes them at dusk, an FDN hall and a soft clipper finish the mix. Prelude, theme, chorale, interlude, climax and coda, the schedule is printed while it plays |
| `subtractive` | Acid house jam: two 303 style lines with accents and slides (`!` and `~` step notation), internal and external LFOs on the cutoff, four on the floor drums |
| `subtractive2` | Guided tour of every feature of the subtractive synth and its presets, section by section (the schedule is printed at start) |
| `fm` | FM synth: a jam (electric piano chords with random velocities, FM bass, sparse bells) followed by a tour of all FM presets, each showing settings that change note by note (velocity, operator ratios and levels, mod depth, feedback, all 8 algorithms, an LFO on the mod depth), then a parade of the General MIDI inspired presets, two bars each with a phrase that suits the instrument. The schedule is printed at start |
| `dx` | Plays every voice of a DX7 style SysEx bank, two bars each: `go run ./cmd/examples/dx path/to/bank.syx` (without a file it plays the built in patches) |
| `granulator` | Electric piano, mallets and drums through the granulator, changing scene every 4 bars: a cloud, an octave shimmer, a freeze, a reverse smear, a stutter and a feedback cascade |
| `grains` | Tour of the granular sampler on recordings from assets: choir pad, frozen drone with an LFO on the position, glass keys with rising velocity, sea texture with rising density, vocal stutter, shimmer, scan speeds and a plucked arpeggio |
| `pulsar` | Tour of the pulsar synth: a vowel lead changing vowels per note, a formant pad swept by an LFO, a buzz bass with rising duty, one melody through all four windows, pulses rising from a rhythm into a tone, crackles with rising masking and laser chirps, then a parade of the other presets (choir, strings, brass, mallet, sub bass, wobble bass, insects, raindrops, motor), two bars each. The schedule is printed at start |
| `wavetable` | Tour of the wavetable synth on the tables in `assets/wavetables2048`: a pad drifting through DN Harm, a position sweep through OPFoldFMFB, a lead changing position per note, plucks with a growing position envelope, a bass with an opening filter, a growl with a speeding position LFO, bell keys, an anthem growing from 1 to 7 unison oscillators, an LFO on value input 1 scanning FMAdditive1 and drum hits at random positions, then a parade of the other presets (strings, vox, brass, organ, reese, acid, stab, drone, glass, riser), two bars each on a table that suits them. The schedule is printed at start |
| `wavegen` | Tour of the generated wavetables: a brightness lead, PWM organ chords, a sync lead, FM bells, a wavefolded bass, a voice singing vowels, opening drawbars and a pad through random spectra (4 bars each), then two bars each of a resonant filter sweep, CZ resonance, odd and even harmonics, a comb, saturation, bit reduction, steps, a shape crossfade and tables sliced from a choir and a vocal recording, the position rising note by note. `go run ./cmd/examples/wavegen tables` also saves the tables as WAV files in `tables` |
| `filters` | Tour of the filter modules: SVF, ladder lowpass and bandpass, biquad peak EQ wah and formant vowel morph, each swept by an LFO |
| `vocoder` | Speech vocoding a chord pad, with a flute melody, a pad, falling stars and panning, using `latency` to line everything up |
| `reverb` | Plucks and a pad through the FDN reverb (large dark hall) |
| `shimmer` | Slow notes through the shimmer reverb, rising an octave per pass |
| `convolution` | Plucks through the convolver with a recorded church impulse response (or a synthetic one) |
| `drums`, `drums2` | Sample based drum patterns with the `drums` synth, steps and templates (`drums2` also arranges parts over time with `dispatch`) |
| `sampler` | Speech regions played by a `player` with flanger, pingpong delay and freeverb |
| `granular/drops`, `granular/heavenly`, `granular/loopup` | Granular synthesis with custom parameter generators, sources and envelopes |
