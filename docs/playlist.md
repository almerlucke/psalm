# Playlist

`modules/playlist`

Songs one after the other in one patch. Every song is a patch of its own (`p.NewSub()`), with
its own tempo, its own random streams and its own clock that starts at 0 when the song starts. A
song is built when it is needed: while one plays, the next is built in the background, so it
starts in time, and a song that rang out is taken away again.

```go
list := playlist.New(p, playlist.Config{Gap: 1},
	playlist.Song{Name: "Morning", Build: morning},
	playlist.Song{Name: "Groove", Build: groove},
)
p.Out(list) // or a channel of a mixer

func morning(p *psalm.Psalm) playlist.Built {
	p.Env().SetBPM(72) // the song's own tempo
	form := arrange.NewForm(...)
	// instruments, parts, an arrangement, a mixer: added to p, the song's patch
	return playlist.Built{Out: mix.Master(), Length: arrangement.Duration()}
}
```

A `Song` has a name and a `Build` function. It gets the song's patch and returns `Built`: the
module to hear (stereo, usually the master of the song's mixer) and the seconds the song plays
(`arrangement.Duration()`; 0 plays until it is skipped).

**Endings**: when a song reaches its length its clock holds (`HoldClock`): no new notes, the
song doesn't move on (and doesn't loop back to its start), while what sounds rings out: the last
chord, the hall. A song is taken away once it has been silent for half a second, or faded away
after `Config.Tail` seconds (8).

**Between songs** (`Config`):

| Field | Meaning |
|---|---|
| `Gap` | Seconds between the end of a song and the start of the next; the song before rings on |
| `Crossfade` | Seconds the next song starts before the end of one, the two fading over each other (no gap) |
| `Tail` | Seconds a song that ended may ring before it fades away (0 is 8) |
| `FadeOut` | Seconds a skipped song fades out (0 is 1.5) |
| `Loop` | After the last song the first again |

**Playing it** (from any goroutine): `Queue(i)` / `QueueSong(name)` asks for a song next (it
starts when the song playing ends), `Skip()` ends the song playing now (it fades out, the next
starts at once), `SetEnabled(i, false)` passes a song over, `Playing()` returns the song playing
and how far it is, `Done()` whether the playlist ended. `p.Restart()` (and Restart on a panel's
page) goes back to the start of the playlist: the first song that is on plays from its start at
once, the songs sounding fade away in a moment and a song asked for next is forgotten (in the
first song, it starts again in place).

**On a panel**: `list.Controls(ctl, "Songs")` adds a tile per song (tap one to play it next, its
dot to pass it over, the song playing shows how far it is; every tile is a button
`"Songs/<song>"` a MIDI pad can play) and a Next button (`"Songs next"`). With Start/Stop and
Restart in the page's header that is a whole player.

**Building**: in a patch that plays through a sound card the first song is built by `New`, the
next ones in the background while a song plays, and the first song is kept built too, so a
restart starts it right away (build the rest of the patch before it plays:
building a song switches the current environment for a moment). A song that isn't built yet when
it is due (just asked for) starts as soon as it is. Offline every song is built by `New`, so a
render is the same every time and `Duration()` knows the length of the list:

```go
render.ToFile(p, "list.wav", render.Options{Seconds: list.Duration(), Tail: 8})
```

`Close()` stops the background building (it returns once a song being built is done).

## Patches within a patch

The songs are made with `p.NewSub()`: a `psalm.Psalm` playing within another one. It has the
sample rate and buffer size of the patch around it, and a clock and an environment of its own
(tempo, PPQ, random streams). It ticks its own modules when it runs, from tick 1, whatever time
the patch around it is at; the control ticks of the patch around it don't reach them. Its `Do`
goes to the patch around it (the audio thread there), `Restart` restarts only it, and
`HoldClock(true)` makes its clock stand still while its sound goes on. `NewSubWith(cfg)` takes a
config of its own instead of the one pushed (`PushConfig`), for building on another goroutine.
