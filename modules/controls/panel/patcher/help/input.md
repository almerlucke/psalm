# input

The **sound input** of the computer (a microphone or a line in): a voice to sing over the piece, or any sound to send through the effects and the mixer. Its `out` is audio.

## Tutorial

1. Use headphones. If the speakers play the piece, the microphone hears them and the sound sings back into itself.
2. Add an input and cable its `out` to an effect (a [reverb](reverb.md), a [granulator](granulator.md), a [vocoder](vocoder.md)) or to a [mixer](mixer.md) channel. Speak: you hear yourself.
3. Set the **gain** (dB, -60 to 24) so the loudest part does not clip. **mono** takes one channel of the input and plays it on both sides, which is right for a single microphone.

```example A microphone, a little boosted
[set]
gain = 6
mono = 1
```

4. Put your voice through the [vocoder](vocoder.md): this is its modulator (the `voice` input), with a synth as the carrier, and the synth seems to speak.
5. To keep what you sing, put a [recorder](recorder.md) after it.

## Settings

| | |
|---|---|
| gain | dB, -60 to 24 |
| mono | 1 takes one channel of the input, 0 keeps both |
