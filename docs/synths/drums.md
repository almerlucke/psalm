# drums

`synths/drums`

`drums.NewDrums(soundBank, numVoices, numChannels)` is a `poly` synth of sample `player` voices.
Each trigger plays a sound from the bank, chosen with the `"sound"` key in the note message
(`"speed"` changes its pitch). A note off fades the sound out in 8 ms (a choke): the step
sequencer (`stepseq`, see [Controls](../controls.md#stepseq)) uses it for choke groups.

```go
drms := drums.NewDrums(soundBank, 16, 2).AddTo(p)
drms.ValueIn(templ.NewTrigger(0.5, template.Template{"sound": "kick", "speed": 1.0}).
	ValueIn(timer.WithTicks(float64(psalm.PPQ())).AddTo(p)))
```
