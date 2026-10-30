# keyboard

**Notes from you** into the patch: a MIDI keyboard, or the keys of the page. Its `notes` output goes to the synths it is cabled to, and it also gives two mod outputs: the **mod wheel** (`wheel`) and the **pressure** (`pressure`). The keyboard has no settings of its own.

## Tutorial

1. Add a keyboard and a synth (any in the Synths group). Cable the keyboard's `notes` to the synth's input, and the synth to the mixer or the output.
2. Play. A connected MIDI keyboard works at once; without one, the keys of the computer play (A W S E D F T G Y H U J K, with Z and X an octave down and up).
3. Hold the sustain pedal and notes stay until you let go; the pitch bend moves the pitch; how hard you play sets the velocity.
4. Use the wheel and pressure as modulation. Cable `wheel` to a setting, for instance the cutoff of a synth or the depth of a [chorus](chorus.md), and the mod wheel opens it; cable `pressure` to a level or a vibrato and pressing harder after the note starts moves it.
5. Several keyboards can be in a patch: each one plays what its cables reach, so one keyboard can play a bass and another a pad, or chords and a melody through an [arp](arp.md).
6. Send the notes through something before they reach the synth: an [arp](arp.md) turns held notes into a figure, a [looper](looper.md) records and repeats what you play, and a [noteroll](noteroll.md) can show them.

For notes that nobody plays, use a [part](part.md), a [voice](voice.md) or a [melodist](melodist.md).
