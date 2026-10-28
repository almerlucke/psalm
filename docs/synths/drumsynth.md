# drumsynth

`synths/drumsynth`

Drums from oscillators and noise, the way the classic drum machines make them: a kick is a sine
falling in pitch, a snare two tuned heads and a burst of noise, a hihat six square waves at
clashing pitches through a high filter. A kit is a set of sounds by name, played like the sample
kit of `drums`: a step sequencer row plays a sound by name, `"speed"` changes its pitch, a note
off chokes it; a keyboard plays them on the General MIDI keys.

```go
kit := drumsynth.NewPoly(drumsynth.Config{Kit: drumsynth.Kit909()}, 16).AddTo(p)
seq.Row("kick", stepseq.Sound(kit, "kick"))
```

The kits `Kit808()`, `Kit909()`, `KitElectro()` (zapping kicks and toms, tight hats), `KitTrap()` (a long driven 808 boom, a big clap) and `KitIndustrial()` (everything distorted, metal for hats) have `kick`, `snare`, `hat`, `open`, `tomLow`, `tomMid`,
`tomHigh`, `clap`, `rim`, `cowbell` and `cymbal`; the hats choke each other. Every kit also has the rest of
the General MIDI percussion in its own character (driven in the industrial kit, tuned higher in
the electro kit): `crash`, `ride`, `rideBell`, `tambourine`, `shaker`, `bongoHigh`, `bongoLow`,
`congaMute`, `congaHigh`, `congaLow`, `timbaleHigh`, `timbaleLow`, `agogoHigh`, `agogoLow`,
`clave`, `woodblockHigh`, `woodblockLow`, `triangle` and `triangleMute`. `GeneralMIDI` maps the
General MIDI drum keys to them, so a drum track from a MIDI file plays every hit. A `Sound` has a
`Kind` (`Kick`, `Snare`, `Hat`, `Tom`, `Clap`, `Rim`, `Cowbell`, `Cymbal`, `Shaker`), `Pitch`, `Decay`,
`Sweep` and `SweepTime` (the pitch falling), `Tone` (the filter of its noise or metal), `Snap`
(the attack: a kick's click, a snare's wires), `Drive`, `Level`, `Pan` and `Choke` (a group).
Change one or add your own:

```go
kit := drumsynth.Kit808()
kit.Sounds["kick"] = drumsynth.Sound{Kind: drumsynth.Kick, Pitch: 45, Decay: 2, Sweep: 14, SweepTime: 0.03, Level: 0.7}
```

The sounds of the kits peak around -3 dB (kick), -5 (snare), -6 (toms, clap) and -10 (hats, rim).

**Messages**: `sound` (a name of the kit) or `pitch` (a key), `speed`, `velocity`, `level`, and the [pitch bend, mod wheel and aftertouch](../../synths.md#pitch-bend-mod-wheel-and-aftertouch).
Every sound of the kit by its name: `<sound><Setting>`, the settings `Pitch`, `Decay`, `Sweep`,
`SweepTime`, `Tone`, `Snap`, `Drive`, `Level` and `Pan` (`"kickDecay": 0.8`, `"hatTone": 0.9`),
taken from its next hit; the voices of a poly share the kit. `Describe` gives the kit's every sound
a group of its own: the editor of the panel is a drum kit editor.
