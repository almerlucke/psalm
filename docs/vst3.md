# VST3 plugins

Psalm hosts [VST3](https://steinbergmedia.github.io/vst3_dev_portal) plugins (a prototype, on Linux), next to the [CLAP plugins](clap.md): an instrument or an effect from a `.vst3` bundle is a module like any other. The host is `modules/vst3` (a C++ layer over the interface headers of the VST3 SDK, which are MIT licensed and kept in `modules/vst3/include/pluginterfaces`), the patcher has two kinds for what it finds, and `cmd/plugins` lists and tries plugins. **Building it needs a C++ compiler** (`g++`) as well as a C compiler.

It does what the CLAP host does, in the same way: what carries over is described in [CLAP](clap.md#what-the-host-does-for-a-plugin) (state in the patch, the knobs following the plugin, ports and a side chain, tempo and position, the plugin's own window by a double click, crash containment, one main thread for the plugins). This page is about what is different.

## Where plugins are searched for

`$VST3_PATH` (colon separated), `$PSALM_VST3_PATH`, `~/.vst3`, `/usr/lib/vst3` and `/usr/local/lib/vst3`, with their subdirectories: every `Name.vst3` folder with its library in `Contents/x86_64-linux/`. A plugin built for a newer system than this one (a newer libc) does not load: it is skipped.

## In the patcher

- **vst3** (Synths): the instruments, each a preset of the kind; **vst3 fx** (Effects): the effects.
- The parameters of a plugin are its settings (a message moves one by the lower case name of the parameter: `set({"filter cutoff": 0.4})`; `go run ./cmd/plugins -format vst3 -plugin "Name" -params` lists them). A plugin with more than forty parameters gets a **Main** group at the top, as CLAP plugins do, and the parameters of a plugin are grouped by its units.
- **Presets**: a plugin's programs (its list of sounds, selected by a parameter) and the files made for it (`presets/<vendor>/<plugin>/*.vstpreset` in `~/.vst3`, `/usr/share/vst3` and `/usr/local/share/vst3`, and `$VST3_PRESET_PATH`) are in a **Preset** list at the top of its settings. Many plugins have neither: Surge XT keeps its patches in its own folders and offers them to a host as CLAP presets, not as VST3 ones.

## What is different from CLAP

- **Pitch between the keys**: a note is played on the nearest key with its tuning in cents (the tuning of the note-on, or the note expression for tuning when the plugin has one). Plugins made with JUCE (Surge XT, OB-Xf) ignore both, so on them a just-intoned pitch is played on the nearest key: use their CLAP builds for those.
- **No preset discovery or parameter text tables**: the text a plugin shows for a value is asked for as in CLAP; the presets are the programs and the files above.
- **The window**: X11 only, embedded in a window of ours (the plugin asks the host for a run loop for its timers and descriptors, which the main thread of the plugins gives it).

## Tests

`go test ./modules/vst3` compiles a VST3 plugin of its own (`modules/vst3/testdata/testplugin.cpp`: a sine instrument with three outputs and programs, and a gain effect with a side chain and a latency) and plays it. With `VST3_TEST_DIR` set to a directory with the VST3 builds of Surge XT and Surge XT Effects, `go test -run TestVST3RealPlugins ./modules/controls/panel/patcher/catalog` plays a chord through both; with `VST3_TEST_DIR` (OB-Xf) and `CLAP_TEST_WINDOW`, `TestPageVST3NativeWindow` opens a plugin window.
