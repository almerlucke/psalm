# recorder

Writes what goes through it to a **file**. The sound passes unchanged, and while the `record` switch is on it is also written to a 32 bit float WAV in the folder `recordings`, named after the module and the time. Put one on the master to keep the whole piece, or on a single voice to keep it dry.

## Tutorial

1. Put the recorder between the sound and where it goes: for example a synth into the recorder and the recorder into the mixer (or the mixer into the recorder and that into the output, to keep the whole piece).
2. Start the piece. Switch `record` on: the log of the patch (the **Log** button) says `recording to ...` with the path of the file.
3. Switch `record` off. The log says `saved ...` and how long the recording is. If it says buffers were lost, the disk could not keep up.
4. If it cannot make the file, the log says `error: can't record to ...`.

## Settings

| Setting | What it does |
|---|---|
| `record` | On: write a new file; off: close it. |

## Good to know

- With an [input](audiolooper.md) (a microphone) before the mixer, one recorder between the input and the mixer keeps the voice dry, and one on the master keeps the voice with the piece. The sound you hear is a little later than the piece (the sound card), so what you sing is a little late against it, in the recording too.
- To loop sound instead of keeping it, see [audiolooper](audiolooper.md).
