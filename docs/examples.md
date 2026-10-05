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
| `subtractive` | Acid house jam: two 303 style lines with accents and slides (`!` and `~` step notation), internal and external LFOs on the cutoff, four on the floor drums |
| `subtractive2` | Guided tour of every feature of the subtractive synth and its presets, section by section (the schedule is printed at start) |
| `filters` | Tour of the filter modules: SVF, ladder lowpass and bandpass, biquad peak EQ wah and formant vowel morph, each swept by an LFO |
| `vocoder` | Speech vocoding a chord pad, with a flute melody, a pad, falling stars and panning, using `latency` to line everything up |
| `reverb` | Plucks and a pad through the FDN reverb (large dark hall) |
| `shimmer` | Slow notes through the shimmer reverb, rising an octave per pass |
| `convolution` | Plucks through the convolver with a recorded church impulse response (or a synthetic one) |
| `drums`, `drums2` | Sample based drum patterns with the `drums` synth, steps and templates (`drums2` also arranges parts over time with `dispatch`) |
| `sampler` | Speech regions played by a `player` with flanger, pingpong delay and freeverb |
| `granular/drops`, `granular/heavenly`, `granular/loopup` | Granular synthesis with custom parameter generators, sources and envelopes |
