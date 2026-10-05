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
| `jungle` | "Rainfall", a jungle track at 172 BPM: programmed breakbeats with ghost notes, fills and a build roll, a resampled tabla loop, a reese bass with a sine sub, a PWM pad, glass arpeggios, rave stabs, a sync lead through delay and shimmer, granular rain in a deep space impulse response, a riser and impact hits. Parts fade per section, a filter sweeps the drum bus, a soft clipper keeps the master safe |
| `song` | "Tides", a complete song using most of psalm: a granular texture through a convolution reverb, DX pads and arpeggios, an FM electric piano, a subtractive bass, lead and riser, vocoded speech, sample drums, chorus, a filter sweep, pingpong delay, shimmer and FDN reverbs. Parts run through faders that an arrangement module fades in and out per section |
| `subtractive` | Acid house jam: two 303 style lines with accents and slides (`!` and `~` step notation), internal and external LFOs on the cutoff, four on the floor drums |
| `subtractive2` | Guided tour of every feature of the subtractive synth and its presets, section by section (the schedule is printed at start) |
| `fm` | FM synth: a jam (electric piano chords with random velocities, FM bass, sparse bells) followed by a tour of all FM presets, each showing settings that change note by note (velocity, operator ratios and levels, mod depth, feedback, all 8 algorithms, an LFO on the mod depth), then a parade of the General MIDI inspired presets, two bars each with a phrase that suits the instrument. The schedule is printed at start |
| `dx` | Plays every voice of a DX7 style SysEx bank, two bars each: `go run ./cmd/examples/dx path/to/bank.syx` (without a file it plays the built in patches) |
| `granulator` | Electric piano, mallets and drums through the granulator, changing scene every 4 bars: a cloud, an octave shimmer, a freeze, a reverse smear, a stutter and a feedback cascade |
| `grains` | Tour of the granular sampler on recordings from assets: choir pad, frozen drone with an LFO on the position, glass keys with rising velocity, sea texture with rising density, vocal stutter, shimmer, scan speeds and a plucked arpeggio |
| `pulsar` | Tour of the pulsar synth: a vowel lead changing vowels per note, a formant pad swept by an LFO, a buzz bass with rising duty, one melody through all four windows, pulses rising from a rhythm into a tone, crackles with rising masking and laser chirps, then a parade of the other presets (choir, strings, brass, mallet, sub bass, wobble bass, insects, raindrops, motor), two bars each. The schedule is printed at start |
| `filters` | Tour of the filter modules: SVF, ladder lowpass and bandpass, biquad peak EQ wah and formant vowel morph, each swept by an LFO |
| `vocoder` | Speech vocoding a chord pad, with a flute melody, a pad, falling stars and panning, using `latency` to line everything up |
| `reverb` | Plucks and a pad through the FDN reverb (large dark hall) |
| `shimmer` | Slow notes through the shimmer reverb, rising an octave per pass |
| `convolution` | Plucks through the convolver with a recorded church impulse response (or a synthetic one) |
| `drums`, `drums2` | Sample based drum patterns with the `drums` synth, steps and templates (`drums2` also arranges parts over time with `dispatch`) |
| `sampler` | Speech regions played by a `player` with flanger, pingpong delay and freeverb |
| `granular/drops`, `granular/heavenly`, `granular/loopup` | Granular synthesis with custom parameter generators, sources and envelopes |
