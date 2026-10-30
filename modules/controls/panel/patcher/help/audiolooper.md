# audiolooper

A **looper for sound**, like a looper pedal. What goes through it is recorded into a loop of some bars on the beat of the patch, and the loop plays on, pass after pass, over the sound that comes in. Record again and the new layer lies on the others. (For notes, see [looper](looper.md).)

## Tutorial

1. Cable a sound into the audiolooper (a microphone input, or a synth) and its `out` to the mixer or the output. Start the clock of the patch (the bar at the top).
2. Choose `bars`, the length of the loop (1 to 16, in 4/4).
3. Switch `record` on. The recording starts at the start of the next pass, and the page shows how long until then: a cell for each bar, its four beats lit as they pass (the first beat red while a layer is recorded). Play or sing.
4. Switch `record` off: the layer ends with the pass. The loop plays on. Switch it on again for another layer on top. A row for each layer shows its sound and a button that takes that layer away alone.
5. Balance with `input level` (the sound coming in) and `loop level` (the loop).

## Settings

| Setting | What it does |
|---|---|
| `bars` | The length of the loop, in bars (1 to 16). |
| `record` | Record a layer, from the start of the next pass until the pass after you switch it off. |
| `play` | The loop plays. |
| `loop level` | How loud the loop is. |
| `input level` | How loud the sound coming in is. |
| `play layers earlier` | Milliseconds. The sound card is late: what you play in time is recorded a little after the beat, so a layer would play late. Try 30 to 120 and listen. |
| `undo` | Takes away the last layer. |
| `clear` | Takes away all the layers. |
| `export to WAV` | Writes the layers to `recordings` (one pass each, so they loop). |
| `export files` | `one per layer` or `one file`. |

## Good to know

- A layer plays at the speed it was recorded at: after a change of tempo, clear and record again.
- The module runs only when its output reaches the output of the patch.
- To keep a whole performance as a file, use the [recorder](recorder.md).
