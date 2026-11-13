# midiplayer

Plays a **MIDI file**: its notes go to the synths it is cabled to, like a [keyboard](keyboard.md), with an output for each of the 16 MIDI channels, and what else the file holds (controllers, pitch bend, pressure) comes out as **mod** outputs.

## Tutorial

1. Add a midiplayer and a synth (any in the Synths group). Cable the player's `all` output (every channel) to the synth and the synth to the output.
2. Pick a file from the list above the **file** box (the MIDI files of `assets/midi` and below), or **Upload…** one from your computer (it is kept in that folder), or type a path (`assets/midi/jig-1.mid`). The line under the box shows the tracks and the length, and goes red when it can't be read.
3. Switch **play** on. It plays from where it is, and stops at the end; switch it off and the notes let go. **Rewind** goes back to the start. Typing another file while it plays swaps it, the settings staying.
4. **loop** starts over at the end, **speed** plays faster or slower (0.25 to 2), **follow the tempo** plays in step with the tempo of the patch (a [clock](clock.md), a song) instead of the tempo of the file.
5. A file of many channels: outputs `1` to `16` each play one channel, so cable channel 1 to a lead, channel 2 to a bass, channel 10 to a drum synth. All play from the same file in time. **transpose** moves the notes of every output.
6. **follow the song** (**sync**) puts the file where the song of the patch is: the bar of the form is the bar of the file, so the song's sections play their part of the file, a section jumped to too, and it stops where the song ends (play is the song's then). **velocity floor** lifts the soft notes (0 plays the velocities of the file).
7. The outputs besides `notes`:
   - The pitch bend of a channel reaches the synth cabled to that channel's output by itself (every synth takes it, a bend of two semitones), and the aftertouch and mod wheel too: there is nothing to cable. The mod outputs below are for moving *other* settings, and follow every channel, or the one **mod outputs of channel** says.
   - `bend` is the pitch bend, 0.5 at rest; `pressure` the channel pressure.
   - `ctrl 1` to `ctrl 6` are controllers, 0 to 1. **controllers** says which: `1 11 7 10 74 71` is the mod wheel, expression, volume, pan, brightness and resonance. Cable `ctrl 5` (74) to a cutoff and the filter follows what the file plays.

```example The brightness of the file moves a filter
[set]
play = 1
loop = 1
```
