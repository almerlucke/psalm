# Rendering, meters and profiling

Tools to render patches offline, measure them and find out where the time goes. Create the patch
without an audio engine (`psalm.New(nil)`) to render it.

## Rendering

`tools/render`

```go
p := psalm.New(nil)
// ... build the patch, p.Out(...)

stats, err := render.ToFile(p, "song.wav", render.Options{Seconds: 180, Tail: 10})
fmt.Println(stats) // rendered 184.2 s in 31.0 s: 5.9x real time (17% of a core live), peak -1.3 dB, ...
                   // buffers: 99% under 2.3 ms, the longest 5.3 ms, 0 of 58227 over the 5.8 ms a buffer lasts
```

The last line is the time every buffer took (`P99`, `Worst`, `Late` of the `Stats`): the average of a whole render hides
the buffers that take longer than they last, and live those are dropouts. `Late` should be 0.

`Options`: `Seconds` to render, `Tail` up to this many seconds more while the output still sounds
(it stops once it is silent, `Silence` dB, -80), and `Until`: a function that keeps the render
going after `Seconds` until it returns true, for a patch whose length isn't known before (a
playlist mixing on bar lines: `Until: list.Done`).

| Function | |
|---|---|
| `ToFile(p, path, opts)` | Renders the patch output to a stereo 32 bit float WAV file |
| `Stems(p, path, opts, sources)` | The same, and in the same pass the output of every source module to its own file in `<name>-stems/` |
| `MixerStems(mix)` | Every strip of a mixer as stem sources (after volume and pan, so the channel stems add up to the mix) |
| `Run(p, opts, func(out psalm.Channels))` | Renders without writing, calling a function with the output of every buffer |
| `RunProfiled(p, opts)` | Renders measuring the time of every module, see below |
| `CPUProfile(path, func())` | Writes a Go CPU profile of a function, for `go tool pprof` |

`Options`: `Seconds` to render, `Tail` (up to this many seconds more while the output still sounds,
for reverb tails, stopping once it has been silent for a quarter second) and `Silence` (the level
the tail stops at, -80 dB by default). With an arrangement, `arrangement.Duration()` is the length
of one pass of the song, following its tempo changes.

The `Stats` hold the seconds rendered, the time it took, `Speed()` (times faster than real time),
`Load()` (the part of a CPU core playing it live would take), and the peak, RMS and integrated
loudness of the output.

## Repeatable renders

Every module that needs randomness (grains, humanized timing, random pans, masking, noise,
sample and hold LFOs) draws from its own random stream, taken from its patch's seed. Seed before
creating the patch and a render comes out the same, sample for sample:

```go
psalm.Seed(7)
p := psalm.New(nil)
```

Without a seed every run is different. In your own modules use `m.Rand()`.

## Parallel renders

Every patch has its own environment (config, tempo, random streams), so several patches can
render at the same time, each on its own goroutine, with the same result as rendering them alone.
Build them one after the other (a new `psalm.New` makes its patch current for the modules created
after it), then render:

```go
psalm.Seed(1)
a := buildSong()
psalm.Seed(2)
b := buildOtherSong()

go render.ToFile(a, "a.wav", render.Options{Seconds: 60})
go render.ToFile(b, "b.wav", render.Options{Seconds: 60})
```

## Profiling

```go
fmt.Print(render.RunProfiled(p, render.Options{Seconds: 60}))
```

reports the time every module spent synthesizing, by type and the slowest instances, as a share
of the render:

```
module type                                      count         time   share
poly.Poly[wavetable.Voice]                           3    9145.4 ms   19.6%
convolver.Convolver                                  1    5711.4 ms   12.2%
...
```

A module's time is its own work, not that of the modules feeding it; a poly synth includes its
voices. Effects rest while they and their input are silent (see
[conventions](conventions.md#tempo-randomness-and-silence)), so a reverb in a section where nothing
plays costs nothing. Modules show their ID when they have one (mixer strips are named after their strip, set
others with `SetID`). Underneath, `psalm.SetProfiling(true)` makes every module measure its time
(`CPUTime()`, `ResetCPUTime()`).

## Meter

`modules/meter`

A module that passes its stereo input through unchanged and measures it: peak, RMS and loudness
after EBU R128 (ITU-R BS.1770, with the K weighting computed for the sample rate).

```go
m := meter.New()
m.AddTo(p)
m.In(source)
// ... after rendering
fmt.Println(m) // peak -3.1 dB  rms -18.0 dB  integrated -16.2 LUFS  short-term -15.9 LUFS
```

| Method | |
|---|---|
| `Peak()`, `RMS()` | dB since the start or a reset |
| `Loudness().Momentary()` | LUFS over the last 400 ms |
| `Loudness().ShortTerm()` | LUFS over the last 3 s |
| `Loudness().Integrated()` | LUFS over everything, gated (silence below -70 LUFS and passages more than 10 LU below the rest are left out) |
| `Reset()` | Start again |

`meter.NewLoudness(sampleRate)` is the loudness meter on its own, fed with `Add(left, right)`. The
mixer master has one too: `mix.Master().Loudness()`, and `mix.Meters()` reports it.
