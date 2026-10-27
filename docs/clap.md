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

## The plugin's own presets

A plugin that has presets (CLAP's preset discovery: the places it keeps them, and ones inside it) gets a **Preset** group at the top of its settings with a list to choose from: the kind of sound (its category, or the folder the file is in) and the name, sorted ("Plucks / Taped Plucks"). Choosing one loads it into the plugin on a thread of its own. The knobs of the editor keep the values they had: they do not follow what a preset sets inside the plugin. `go run ./cmd/clap -plugin "Surge XT" -presets` lists them, `-load "Taped Plucks" -notes 60,64,67 -render x.wav` plays one.

Surge XT keeps its presets (about 3000 of them, and the wavetables they use) in its data folder, which the plugins-only download does not have. Take the full download (`surge-xt-linux-x86_64-*.tar.gz` of the [Surge XT releases](https://github.com/surge-synthesizer/releases-xt/releases)): its `lib/clap/*.clap` go in `~/.clap` and its `share/surge-xt` in `~/.local/share/surge-xt`, or leave them where they are and run with `CLAP_PATH=<where>/lib/clap SURGE_DATA_HOME=<where>/share/surge-xt`.

## Presets of your own, made from text

The presets of the plugins on this page are in `modules/clap/presets/` as text, a line a setting, written the way the plugin shows its values: `a osc 1 type = Wavetable`, `a amp eg decay = 450 ms`, `a common a volume = -2 dB` (`@0.7` for a parameter that shows nothing: its value from 0 to 1; a key may be the end of a key). `go run ./cmd/clap` looks the values up (it asks the plugin what text each value shows), and saves a preset of the editor:

```
go run ./cmd/clap -plugin "Surge XT" -preset modules/clap/presets/surge-xt/pluck.txt -save editors-presets/clap
go run ./cmd/clap -plugin "Surge XT" -preset modules/clap/presets/surge-xt/pluck.txt -notes 60,64,67 -render pluck.wav
go run ./cmd/clap -plugin "Surge XT" -explore "a osc 1 type;a amp eg attack"     what a parameter shows over its range
```

Surge XT: Soft Pluck, Island Pluck, Kalimba, Plucked String, Glass Bell, Warm Pad, Sub Bass, Hawaiian Lead, set to about the same loudness (-18 dB). OB-Xf (Surge Synth Team's virtual analog polysynth, free; `modules/clap/presets/ob-xf/`, about -18 dB, the pluck quieter since its master volume is at the top): Warm Pad, Analog Bass, Brass Stab, Pulse Pluck, Slide Lead. Its filter works from the start (Surge XT's needs the state round trip below). Surge XT Effects is one effect at a time, its type chosen by the `fx type` setting (Delay, Reverb 1 and 2, Spring Reverb, Phaser, Flanger, Chorus, Ensemble, Rotary Speaker, Distortion, Tape, EQ and more than twenty others); a preset sets the type and the mix, and the knobs that follow it are named as for the delay whatever the type is (the host does not follow the plugin renaming them).

## What the host does for a plugin

- **State in the patch:** a plugin's own state is kept with the patch (`state` of its module in the patch file) and given back when it loads; the knobs of the editor are then set to what the plugin has. A plugin that loads a preset, or has a knob turned in its own window, tells the editor the same way.
- **Names that change:** a plugin that renames its settings (Surge XT Effects, when its type changes) tells the host (`params.rescan`); the editor has the new labels the next time its side panel is opened.
- **Ports:** every audio output of a plugin is an output of its module (a sampler's parts, Surge's scenes), the second input of an effect is its side chain.
- **Tempo and position:** the plugin is told the tempo of the patch and the position in beats (4/4) with every block.
- **Latency:** the latency a plugin says it has is shown on its module. It is not compensated: nothing delays the other paths of the patch to match.
- **The plugin's own window:** X11 only. A switch "show the plugin's window" at the top of its settings opens it (a plugin that can make its own window does; others are embedded in a window of ours). One thread, the main thread of the plugins, makes every call into a plugin that is marked main-thread and runs the timers, file descriptors and window events they ask for: plugins made with JUCE wait forever for such a thread. `PSALM_CLAP_DEBUG=1` traces it.
- **A plugin that crashes:** a fault or an abort (SIGSEGV, SIGBUS, SIGFPE, SIGILL, SIGABRT) in a call we made into it is caught on that thread, which jumps back out; the plugin is marked dead, makes no more sound, and the module's status says "crashed". Other plugins and the program go on. A plugin that damaged memory before it crashed can still hurt the program, and a plugin that hangs or crashes in a thread of its own is not caught.

## Limits

- The settings are the interface of the web page; the plugin's own window is a separate window on the screen, and only on X11 (Linux, also under Wayland through XWayland).
- Latency is shown, not compensated (see above).
- Stereo and mono ports; the plugins' MIDI output, and note and parameter events other than notes (with their tuning) and parameter values, are not hosted.
- A plugin is only let go when the module is taken away, at the next buffer.
- Linux only so far (the loading, the signals and the window are written for it).

## Tests

`go test ./modules/clap` compiles two small plugins (`modules/clap/testdata/testplugin.c`: a sine instrument and a gain effect) and plays them. With `CLAP_TEST_DIR` set to a directory with Surge XT and Surge XT Effects, `go test -run TestClapRealPlugins ./modules/controls/panel/patcher/catalog` plays a chord through both.
