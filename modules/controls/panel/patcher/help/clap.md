# clap

A **CLAP instrument** installed on this machine, played as a synth of the patch. [CLAP](https://cleveraudio.org) is an open plugin format; psalm hosts it (a prototype, on Linux). Every instrument it finds is a preset of this kind, so the list of presets of the module is the list of plugins. The kind is only there when psalm found a plugin: it searches `$CLAP_PATH`, `$PSALM_CLAP_PATH`, `~/.clap`, `/usr/lib/clap` and `/usr/local/lib/clap` once, when it starts.

## Tutorial

1. Add a clap, choose the instrument you want as its preset, and add a keyboard, or a [voice](voice.md) with some pitches. Cable the `notes` to the clap and its `out` to the mixer.
2. Play it. It takes the notes of the patch like any synth. A pitch between the keys (just intonation, a bend) goes to the plugin as a tuning of the note, so a plugin that supports it plays in tune with it.
3. Open the settings of the module. The parameters of the plugin are its settings, a knob for each, and a mod cable, a [lane](lane.md) or a [script](script.md) can move any of them. A plugin with more than forty gets a group **Main** at the top, with the ones that matter most (volume, cutoff, attack...).
4. If the plugin has presets of its own, the group **Preset** at the top has a list of them.
5. A double click on the module opens the plugin's own window (X11 only). With Alt held the double click opens our settings window instead.

## Good to know

- The state of the plugin is kept with the patch.
- Every audio output of the plugin is an output of the module.
- The plugin gets the tempo of the patch and the position in the bar.
- A plugin that crashes is caught: the module goes silent and its status says "crashed", the rest of the patch plays on.
- I cannot say from here which plugins you have or how they sound. The full story is in the docs, page `clap`.

See also [vst3](vst3.md) and [clap fx](clap%20fx.md).
