# Mixer

`modules/mixer`

A mixing desk for a patch: channel strips with volume, pan, mute, solo, sends and meters, effect
buses, groups with insert effects and a master with a lookahead limiter. The mixer wires its
strips; the effects and inserts are modules you create and add to the patch.

```go
mix := mixer.New(p)

// effect buses: the sends of the strips are summed into the effects, set them fully wet
hall := fdnreverb.DefaultConfig()
hall.Mix = 1
mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))

// groups: sub mixes with inserts, here a filter on all the strings
mix.Group("strings", mixer.StripConfig{Sends: map[string]float64{"hall": 0.4}}, filter)

// channels: any module with an audio output, mono or stereo
mix.Channel("violins", violins, mixer.StripConfig{Pan: -0.4, Group: "strings"})
mix.Channel("harp", harp, mixer.StripConfig{Volume: -3, Pan: 0.5, Sends: map[string]float64{"hall": 0.5}})

p.Out(mix.Master())
```

Create buses and groups before the strips that send or play into them. A strip that refers to a
bus or group that doesn't exist, or a name used twice, panics with the names, without changing
the mixer.

## Strips

`mix.Channel(name, source, config)`, `mix.Group(name, config, inserts...)` and
`mix.Bus(name, config, effects...)` all return a `*mixer.Strip`; groups and bus returns are strips
too, with the same settings. `mix.Strip(name)` finds one.

| Setting | Default | Meaning |
|---|---|---|
| `Volume` | 0 | dB, -120 is silence. Changes glide over 20 ms |
| `Pan` | 0 | -1 left to 1 right, equal power (a balance for stereo sources, unity in the centre) |
| `Mute` | false | Mutes with a 10 ms fade |
| `Solo` | false | When any channel is soloed only soloed channels sound. Groups and bus returns are solo safe |
| `Sends` | | Amount (0-1) sent to each bus |
| `PreFader` | false | Take the sends before the volume and pan (they keep sending when the fader is down) |
| `Group` | "" | The group the strip plays into, "" is the master |

| Method | |
|---|---|
| `SetVolume(db)`, `SetPan(p)`, `SetMute(b)`, `SetSolo(b)`, `SetSend(bus, amount)` | Change a setting |
| `FadeTo(db, seconds)` | Fade in even dB steps, for fades in and out |
| `Levels()` | Peak and RMS in dB since the meters were reset |

**Value inputs**: 1 volume (dB), 2 pan, 3 mute (above 0.5), so the `arrange` automation can drive
them: `arrangement.Automate(lane, mix.Strip("strings"), 1)`. **Messages**: `volume`, `pan`,
`mute`, `solo` and `send.<bus>`.

## Latency

A strip whose signal is late (a vocoder, a fully wet convolver: anything implementing
`psalm.Latent` along its sources and inserts) makes the mixer delay the other strips playing into
the same group or the master, so they arrive together. The strips delay their input, so their
sends stay lined up too. Effect bus returns are not compensated: the latency of a reverb is a
natural pre delay. It happens by itself before the first buffer; `mix.Align()` does it again after
adding strips, `strip.Delay()` returns the delay of a strip in samples.

## Master

`mix.Master()` sums everything into the master volume and a lookahead limiter: it sees peaks
coming and lowers the gain smoothly before they arrive, so nothing clips and nothing distorts
(what little the smoothing lets through is clipped at the ceiling). Set it with
`mix.Master().SetConfig(cfg)`:

| Setting | Default | Meaning |
|---|---|---|
| `Volume` | 0 | dB (value input 1, message `volume`) |
| `Ceiling` | -1 | dB, peaks stay below it (message `ceiling`), 0 turns the limiter off |
| `Lookahead` | 3 | Milliseconds, also the latency of the master (`Latency()` in samples) |
| `Release` | 150 | Milliseconds the limiter takes to let go |

## Meters

Every strip and the master measure peak and RMS. `mix.Meters()` returns a report of all of them
and the most gain reduction of the limiter, `mix.ResetMeters()` starts again. With an arrangement
you get a report per section:

```go
arrangement.OnSection(func(pos arrange.Position) {
	fmt.Print(mix.Meters())
	mix.ResetMeters()
})
```
