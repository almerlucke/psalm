# streamer

`synths/streamer`

The sampler for sounds too long to keep in memory: a recording of an hour of rain, a piece of
music, a field recording. A note plays the file from the disk while it sounds. The first half
second of every file is kept in memory, so the note starts at once, and a goroutine of the
note reads the rest into a ring that the audio thread plays from. The audio thread never waits
for the disk: when the disk is too slow the sound has a gap and `Voice.Underruns` counts it.
Wav, aiff, mp3, flac and ogg play, in any sample rate (it is played at the rate of the file). The file is
read with the streaming reader of the `sndfile` package (`sndfile.Open`, `SeekFrame` and `ReadFrames`), in pure Go; formats
it doesn't read need its `libsndfile` build tag.

```go
inst, report, err := streamer.ParseInstrument("36 assets/sounds/various/rain-sound-188158.wav loop vol=-6")
rain := streamer.NewPoly(streamer.DefaultConfig(inst), 8).AddTo(p)
rain.NoteOn(0.8, psalm.Message{"pitch": 36.0}, psalm.CtrlTick{}, p.Config()) // or notes from a script or a keyboard
```

The sounds are a text, a line each: a key, a file, and anything of `loop` (round while the
key is down, between `loopstart=` and `loopend=` in seconds), `hold` (a note off fades the
sound out; a sound that is not held plays to its end, like a drum), `speed=2` (an octave up),
`start=` and `end=` (the part of the file to play, seconds), `release=` (seconds), `vol=` (dB),
`pan=` and `name=`. A single line plays on every key. `streamer.Open(path, key)` makes an
instrument of one file, `streamer.NewInstrument(&streamer.Entry{...})` one of entries.

In the patcher the kind `streamer` has the text in its window (`sounds`), with the report
of what every line made, and takes notes like the sampler. A loop is not crossfaded at its seam:
cut it where it goes round, or use a file that does. It plays forwards at a fixed speed
(interpolated between frames); for sounds you cut into pieces on keys, or play backwards or by
grains, use the sampler or the grains synth.
