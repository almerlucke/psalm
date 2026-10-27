# CLAP plugins

Psalm hosts [CLAP](https://cleveraudio.org) plugins (a prototype, on Linux): an instrument or an effect from a `.clap` file is a module like any other. The host is `modules/clap` (C glue over the CLAP headers in `modules/clap/include`, which are MIT licensed), the patcher has two kinds for what it finds, and `cmd/clap` lists and tries plugins.

## Where plugins are searched for

`$CLAP_PATH` (colon separated), `$PSALM_CLAP_PATH`, `~/.clap`, `/usr/lib/clap` and `/usr/local/lib/clap`, with their subdirectories. The patcher scans once when it starts; the kinds are left out when there is nothing to find.

## In the patcher

- **clap** (Synths): the instruments, each a preset of the kind. It takes the notes of the patcher like any synth, so a keyboard, a sequence or a script can play it. A pitch between the keys (just intonation, a bend) goes to the plugin as the *tuning* note expression of the note, so a plugin that supports it sounds as tuned (Surge XT does).
- **clap fx** (Effects): the effects, with the audio on their input.

A plugin with more than forty parameters gets a group "Main" at the top of its settings (the first parameter with volume, gain, cutoff, resonance, attack, decay, sustain, release, drive, mix, reverb, delay or width in its name, the ones of its filter, amplifier or master first); in the editor the groups of a synth with more than a hundred settings are closed, a click on a title opens one. The parameters of a plugin are its settings, described like those of any synth (the editor draws a knob for each, a mod cable can move one). A message moves one by the key, the lower case name of the parameter with the group in front: `set({"filter 1 cutoff": 0.4})` from a script. `go run ./cmd/clap -plugin "Surge XT" -params` lists the keys. Surge XT has 775 of them.

## cmd/clap

```
go run ./cmd/clap                                        list the plugins
go run ./cmd/clap -plugin "Surge XT" -params             and its parameters
go run ./cmd/clap -plugin "Surge XT" -render chord.wav -notes 60,64,67.02 -seconds 4 -set "key=value;..."
```

## Presets

The presets of the plugins on this page are in `modules/clap/presets/` as text, a line a setting, written the way the plugin shows its values: `a osc 1 type = Wavetable`, `a amp eg decay = 450 ms`, `a common a volume = -2 dB` (`@0.7` for a parameter that shows nothing: its value from 0 to 1; a key may be the end of a key). `go run ./cmd/clap` looks the values up (it asks the plugin what text each value shows), and saves a preset of the editor:

```
go run ./cmd/clap -plugin "Surge XT" -preset modules/clap/presets/surge-xt/pluck.txt -save editors-presets/clap
go run ./cmd/clap -plugin "Surge XT" -preset modules/clap/presets/surge-xt/pluck.txt -notes 60,64,67 -render pluck.wav
go run ./cmd/clap -plugin "Surge XT" -explore "a osc 1 type;a amp eg attack"     what a parameter shows over its range
```

Surge XT: Soft Pluck, Kalimba, Plucked String, Glass Bell, Warm Pad, Sub Bass, Hawaiian Lead. Surge XT Effects is one effect at a time, its type chosen by the `fx type` setting (Delay, Reverb 1 and 2, Spring Reverb, Phaser, Flanger, Chorus, Ensemble, Rotary Speaker, Distortion, Tape, EQ and more than twenty others); a preset sets the type and the mix, and the knobs that follow it are named as for the delay whatever the type is (the host does not follow the plugin renaming them).

## Limits

- **Surge XT's filters and waveshaper do not respond** when played from here (the settings are accepted and read back, but the sound does not change; its oscillators, unison, envelopes, volume and mixer do respond, and the effects plugin responds to everything). I read Surge's source for the path of a parameter event and could not find why: the presets are made of what works.

- No plugin windows: the parameters are the interface. No presets of the plugin either (CLAP's preset discovery is not hosted yet): a plugin starts from its default sound, and Surge XT's own patches need its data directory (the full install, not the plugins only download). State and presets of the plugin are not saved with the patch yet.
- A plugin runs on the audio thread, so a plugin that crashes takes the program with it, and one that is slow drops out the sound.
- Stereo and mono ports only (the first port of each); no sidechain, no transport (tempo and position), no latency compensation, no parameter changes back from the plugin (its own knobs), no MIDI output.
- A plugin is only let go when the module is taken away, at the next buffer.

## Tests

`go test ./modules/clap` compiles two small plugins (`modules/clap/testdata/testplugin.c`: a sine instrument and a gain effect) and plays them. With `CLAP_TEST_DIR` set to a directory with Surge XT and Surge XT Effects, `go test -run TestClapRealPlugins ./modules/controls/panel/patcher/catalog` plays a chord through both.
