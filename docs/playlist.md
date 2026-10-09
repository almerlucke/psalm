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

**Songs with sections**: a song can hand its transport (step 16 of the tutorial, `arrange`) to the
playlist, `Built{Out: mix.Master(), Length: arrangement.Duration(), Transport: form.Transport()}`.
Then the song ends when its transport ended, wherever its sections took it: a section skipped
ends it sooner, a repeated one later (its form doesn't loop; the song ends with its last
section). `Length` is still how long it plays as written: how far the song is, and the duration
of a render. With a cross fade the next song fades in from the end, over the song ringing out.

`list.SectionControls(ctl, "Sections")` puts the sections of the song playing on a panel: a tile
per section that changes with the song (tap one to play it next, its dot to switch it off), Repeat
and Jump at (they count for every song: `SetSectionRepeat`, `SetSectionJump`). The tiles' buttons
are named by position, `"Sections/1"`, `"Sections/2"`, ..., so a MIDI pad plays the second section
of whatever song plays. Which sections are on is not in presets: every song has its own, and a
song played again is built again, its sections as written. `list.Sections()` is the transport of
the song playing, from code. Step 18 of the tutorial is a setlist like this.

**Songs with controls of their own**: a song can make its own panel in `Build`, in its own
patch, with controls bound to its own modules, and hand it over: `Built{..., Panel: ctl}`.

```go
ctl := panel.New(p, "Groove") // p is the song's patch
ctl.Fader("Tempo", 90, 120, panel.Linear, panel.Func(func(v float64) { p.Env().SetBPM(v) }), panel.Default(104))
return playlist.Built{Out: mix.Master(), Length: arrangement.Duration(), Panel: ctl}
```

`list.SongControls(ctl, "Song")` shows the panel of the song playing on the playlist's page (a
`Mount`, see [panel](controls.md#panel)): its sections and controls under the song's name, and the
next song's when it starts. The song's controls apply on the song's own clock. MIDI reaches the
song playing (the control changes and notes go to the playlist's panel and on to the song's), MIDI
learn on the page binds its controls, and what is bound is kept for the song: built again, it gets
its bindings back. The values of its controls are kept too: a song that plays again has them as
you left them, and the presets of the playlist's panel hold the controls of every song that played
(as `"Song/Groove/Tempo"`, after the name of the mount). Loading a preset sets the song playing at
once and the other songs when they start. `list.SongPanel()` is the panel of the song playing,
from code.

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

Step 17 of the tutorial ([`cmd/examples/tutorial/17-playlist`](../cmd/examples/tutorial/17-playlist/main.go))
plays three short songs this way, with the tiles on a panel.

## Patches within a patch

The songs are made with `p.NewSub()`: a `psalm.Psalm` playing within another one. It has the
sample rate and buffer size of the patch around it, and a clock and an environment of its own
(tempo, PPQ, random streams). It ticks its own modules when it runs, from tick 1, whatever time
the patch around it is at; the control ticks of the patch around it don't reach them. Its `Do`
goes to the patch around it (the audio thread there), `Restart` restarts only it, and
`HoldClock(true)` makes its clock stand still while its sound goes on. `NewSubWith(cfg)` takes a
config of its own instead of the one pushed (`PushConfig`), for building on another goroutine.
