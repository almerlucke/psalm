# vst3

A **VST3 instrument** installed on this machine, played as a synth of the patch. It works as the [clap](clap.md) host does; every instrument found is a preset of this kind. The kind is only there when psalm found a plugin: it searches `$VST3_PATH`, `$PSALM_VST3_PATH`, `~/.vst3`, `/usr/lib/vst3` and `/usr/local/lib/vst3`.

## Tutorial

1. Add a vst3 and choose the instrument as its preset. Add a keyboard or a [voice](voice.md), cable its `notes` to the vst3 and the `out` to the mixer.
2. Play it. The parameters of the plugin are the settings of the module (a mod cable, a [lane](lane.md) or a [script](script.md) can move one), with a group **Main** on top when there are more than forty.
3. The **Preset** list at the top has the programs of the plugin and the preset files made for it. Many plugins have neither.
4. A double click opens the plugin's own window (X11 only); with Alt held, our settings window.

## What differs from CLAP

- A pitch between the keys is played on the nearest key, with its tuning in cents, and plugins made with JUCE ignore the tuning, so use their CLAP build when the tuning matters.
- A plugin built for a newer system than this one does not load: it is skipped.

The state in the patch, the tempo, the side chain and the crash containment are as for [clap](clap.md). See also [vst3 fx](vst3%20fx.md).
