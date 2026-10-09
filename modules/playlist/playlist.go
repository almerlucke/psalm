// Package playlist plays songs one after the other in a patch: every song is a patch of its own
// (psalm.NewSub), with its own tempo and its own clock, built when it is needed and taken away when
// it has rung out. While a song plays the next one is built in the background, so the next starts
// in time; a song that ends holds its clock (no new notes) and rings out under the next one, or
// the songs cross fade. Songs can be asked for next, switched off and skipped, from code or from a
// panel (Controls)
package playlist

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/panel"
)

// Song is a song of a playlist: a name, and a function building it in the patch it gets (its own
// patch: set its tempo with p.Env().SetBPM, create its modules, add them to p)
type Song struct {
	Name  string
	Build func(p *psalm.Psalm) Built
}

// Built is what building a song gives: the module to hear and how long the song plays
type Built struct {
	Out    psalm.Modular // what to hear, stereo (output 0): the master of the song's mixer
	Length float64       // seconds the song plays (an arrangement's Duration), 0 is until it is skipped

	// Transport makes the song end when its transport ended, wherever the sections took it (a
	// section skipped, repeated or asked for): its form doesn't loop then. Length is still how
	// long it plays as written: for how far the song is, and the duration of a render. The section
	// controls of the playlist (SectionControls) steer the transport of the song playing
	Transport *arrange.Transport

	// Panel is the song's own panel (panel.New(p, ...) in Build): its controls, bound to the song's
	// modules. The panel of the playlist shows the one of the song playing (SongControls); MIDI
	// reaches it, and what MIDI learn binds is kept for the song when it is built again
	Panel *panel.Panel
}

// Config sets how the songs follow each other
type Config struct {
	Gap       float64 // seconds between the end of a song and the start of the next (the song rings on)
	Crossfade float64 // seconds the next song starts before the end of one, the two fading over (no gap)
	Tail      float64 // seconds a song that ended may ring before it fades away (0 is 8)
	FadeOut   float64 // seconds a skipped song fades out (0 is 1.5)
	Loop      bool    // after the last song the first again
}

// built is a song built, ready to play
type built struct {
	index int
	sub   *psalm.Psalm
	Built
}

// voice is a song sounding: playing, or ringing out
type voice struct {
	*built
	elapsed  int64   // samples played
	length   int64   // samples it plays, 0 for no end
	gain     float64 // now
	target   float64
	step     float64 // per sample toward the target
	ended    bool    // its clock holds: ringing out
	handed   bool    // the next song was asked for
	ringing  int64   // samples since it ended
	quiet    int64   // samples it has been silent while ringing
	fading   bool    // fading away, then gone
	finished bool
}

// Playlist is a module playing songs one after the other, on its output 0 (stereo). Add it to the
// patch and hear it: p.Out(playlist), or a channel of a mixer
type Playlist struct {
	*psalm.Module
	host  *psalm.Psalm
	cfg   Config
	songs []Song
	live  bool // building in the background (a patch that plays through a sound card)

	// set from anywhere
	off     []atomic.Bool
	queued  atomic.Int32
	skip    atomic.Bool
	playing atomic.Int32  // the song playing, -1 none
	prog    atomic.Uint64 // float64 bits, through it (0-1)

	// the sections of the song playing (a song with a transport)
	sections      atomic.Pointer[arrange.Transport]
	sectionRepeat atomic.Bool
	sectionJump   atomic.Int32

	// the panel of the song playing, and what MIDI learn bound per song
	panel    atomic.Pointer[songNow]
	bindings sync.Map // per song name what MIDI learn bound
	values   sync.Map // per song name the values of its controls (a snapshot of its panel)

	// the audio thread
	voices  []*voice
	pending int   // the song to start, -1 none
	gapLeft int64 // samples before it starts
	current *voice
	started bool
	done    atomic.Bool // the playlist ended

	// building
	buildMu   sync.Mutex   // guards ready and want
	ready     *built       // the song built ahead
	first     *built       // the first song built, kept ready for a restart (live)
	want      int          // the song the audio thread waits for, -1 none
	last      int          // the song taken last: the builder builds the one after it
	config    psalm.Config // the config songs are built with (taken when the playlist was made)
	wake      chan struct{}
	all       []*built // offline: every song, built up front
	once      sync.Once
	closeOnce sync.Once
	stopped   chan struct{} // closed when the builder stopped
}

// building keeps building songs one at a time: building switches the current environment
var building sync.Mutex

// New creates a playlist playing songs in a patch and adds it to the patch. The first song is built
// at once. In a patch that plays through a sound card the next song is built in the background
// while a song plays (build the rest of the patch before it plays: building switches the current
// environment); offline every song is built here, so a render is the same every time
func New(p *psalm.Psalm, cfg Config, songs ...Song) *Playlist {
	if len(songs) == 0 {
		panic("playlist: no songs")
	}
	if cfg.Tail <= 0 {
		cfg.Tail = 8
	}
	if cfg.FadeOut <= 0 {
		cfg.FadeOut = 1.5
	}
	pl := &Playlist{
		Module: psalm.NewModule(nil, []int{2}), host: p, cfg: cfg, songs: songs, live: p.Live(),
		off: make([]atomic.Bool, len(songs)), pending: 0, want: -1, last: -1, wake: make(chan struct{}, 1),
		config: psalm.CurrentStackConfig(),
	}
	pl.SetSelf(pl)
	pl.queued.Store(-1)
	pl.playing.Store(-1)
	if pl.live {
		pl.ready = pl.build(0)
	} else {
		for i := range songs {
			pl.all = append(pl.all, pl.build(i))
		}
	}
	pl.AddTo(p)
	return pl
}

// build builds a song in a patch of its own, the current environment kept
func (pl *Playlist) build(i int) *built {
	building.Lock()
	defer building.Unlock()
	prev := psalm.CurrentEnv()
	defer psalm.SetCurrentEnv(prev)
	sub := pl.host.NewSubWith(pl.config)
	b := pl.songs[i].Build(sub)
	if b.Out == nil {
		panic(fmt.Sprintf("playlist: song %q has nothing to hear (Built.Out)", pl.songs[i].Name))
	}
	sub.Out(b.Out)
	if b.Transport != nil {
		b.Transport.Form().Loop = false // the song ends with its last section
	}
	if b.Panel != nil { // what was learned for the song the time before
		if bs, ok := pl.bindings.Load(pl.songs[i].Name); ok {
			b.Panel.SetBindings(bs.(map[string]panel.Binding))
		}
		if vs, ok := pl.values.Load(pl.songs[i].Name); ok { // its controls as they were (or a preset)
			b.Panel.Restore(vs.(map[string]float64))
		}
	}
	return &built{index: i, sub: sub, Built: b}
}

// builder builds the song that comes next, in the background
func (pl *Playlist) builder() {
	defer close(pl.stopped)
	for range pl.wake {
	build:
		for {
			pl.buildMu.Lock()
			next := pl.want
			if next < 0 {
				next = pl.next(pl.last)
			}
			first := pl.firstOn()
			has := func(b *built, i int) bool { return b != nil && b.index == i }
			needNext := next >= 0 && !has(pl.ready, next) && !has(pl.first, next)
			needFirst := first >= 0 && !has(pl.first, first)
			pl.buildMu.Unlock()
			switch {
			case needNext: // the song due first
				b := pl.build(next)
				pl.buildMu.Lock()
				pl.ready = b
				pl.buildMu.Unlock()
			case needFirst: // then the first song, for a restart
				b := pl.build(first)
				pl.buildMu.Lock()
				pl.first = b
				pl.buildMu.Unlock()
			default:
				break build
			}
		}
	}
}

// poke wakes the builder up
func (pl *Playlist) poke() {
	select {
	case pl.wake <- struct{}{}:
	default:
	}
}

// take returns a song built, nil while it isn't there yet (it is asked for)
func (pl *Playlist) take(i int) *built {
	if !pl.live {
		return pl.fresh(i)
	}
	if !pl.buildMu.TryLock() {
		return nil
	}
	var b *built
	switch {
	case pl.ready != nil && pl.ready.index == i:
		b, pl.ready = pl.ready, nil
	case pl.first != nil && pl.first.index == i: // the first song (a restart): the builder makes another
		b, pl.first = pl.first, nil
	default:
		pl.want = i
	}
	if b != nil {
		pl.want, pl.last = -1, i
	}
	pl.buildMu.Unlock()
	pl.poke()
	return b
}

// fresh returns an offline song, from its start (a song played again is restarted)
func (pl *Playlist) fresh(i int) *built {
	b := pl.all[i]
	b.sub.HoldClock(false)
	b.sub.Restart()
	return b
}

// next returns the song after one: the one asked for, else the next that is on (from the first
// again when the playlist loops), -1 for none
func (pl *Playlist) next(after int) int {
	if q := pl.queued.Load(); q >= 0 {
		return int(q)
	}
	n := len(pl.songs)
	for k := 1; k <= n; k++ {
		i := after + k
		if i >= n {
			if !pl.cfg.Loop {
				return -1
			}
			i -= n
		}
		if !pl.off[i].Load() {
			return i
		}
	}
	return -1
}

func (pl *Playlist) seconds(s float64) int64 {
	return int64(math.Round(s * pl.Config().SampleRate))
}

// fade sets a voice moving to a gain over some seconds
func (pl *Playlist) fade(v *voice, to, seconds float64) {
	v.target = to
	n := float64(max(1, pl.seconds(seconds)))
	v.step = (to - v.gain) / n
}

// Synthesize plays the songs sounding into the output, and moves the playlist on
func (pl *Playlist) Synthesize() {
	if !pl.started {
		pl.started = true
		if pl.live {
			pl.once.Do(func() {
				pl.stopped = make(chan struct{})
				go pl.builder()
			})
		}
		switch q := pl.queued.Load(); { // the song asked for, else the first that is on
		case q >= 0:
			pl.pending = int(q)
		case pl.off[0].Load():
			pl.pending = pl.firstOn()
		}
	}
	buffer := int64(pl.Config().BufferSize)
	out := pl.Output(0).Channels
	out.Clear()

	// a skip: the song playing fades out, the next starts now
	if pl.skip.Swap(false) && pl.current != nil && !pl.current.ended {
		cur := pl.current
		pl.end(cur)
		pl.fade(cur, 0, pl.cfg.FadeOut)
		cur.fading = true
		if !cur.handed {
			cur.handed = true
			pl.pending, pl.gapLeft = pl.next(cur.index), 0
		}
	}

	// a song with a transport ends when its transport ended: the next one now (fading over it)
	if cur := pl.current; cur != nil && !cur.ended && cur.Transport != nil {
		if cur.Transport.Ended() {
			pl.end(cur)
			if !cur.handed {
				cur.handed = true
				pl.pending = pl.next(cur.index)
				if pl.cfg.Crossfade == 0 {
					pl.gapLeft = pl.seconds(pl.cfg.Gap)
				}
			}
		}
	} else if cur != nil && !cur.ended && cur.length > 0 {
		// the song playing reaching its end: ask for the next one (early, to cross fade)
		cross := pl.seconds(pl.cfg.Crossfade)
		if !cur.handed && cur.elapsed >= cur.length-cross {
			cur.handed = true
			pl.pending = pl.next(cur.index)
			if cross == 0 {
				pl.gapLeft = pl.seconds(pl.cfg.Gap)
			}
		}
		if cur.elapsed >= cur.length {
			pl.end(cur)
		}
	}

	// the next song: when the gap is over and it is built
	if pl.pending >= 0 {
		if pl.gapLeft > 0 {
			pl.gapLeft -= buffer
		} else if b := pl.take(pl.pending); b != nil {
			pl.start(b)
		}
	}

	// every song sounding, at its gain
	kept := pl.voices[:0]
	for _, v := range pl.voices {
		pl.sound(v, out)
		if !v.finished {
			kept = append(kept, v)
		}
	}
	clear(pl.voices[len(kept):])
	pl.voices = kept
	pl.done.Store(pl.pending < 0 && len(pl.voices) == 0)

	if cur := pl.current; cur != nil && !cur.ended {
		pl.playing.Store(int32(cur.index))
		p := 0.0
		if cur.length > 0 {
			p = math.Min(1, float64(cur.elapsed)/float64(cur.length))
		}
		pl.prog.Store(math.Float64bits(p))
	} else {
		pl.playing.Store(-1)
	}
}

// firstOn returns the first song that is on, -1 for none
func (pl *Playlist) firstOn() int {
	for i := range pl.songs {
		if !pl.off[i].Load() {
			return i
		}
	}
	return -1
}

// start starts a song built: it fades in over the cross fade (at once without), the song before
// fades out over it
func (pl *Playlist) start(b *built) {
	pl.queued.CompareAndSwap(int32(b.index), -1)
	pl.pending = -1
	for _, other := range pl.voices { // the same song again (offline it is the same patch): from its start
		if other.sub == b.sub {
			other.finished = true
		}
	}
	kept := pl.voices[:0]
	for _, other := range pl.voices {
		if !other.finished {
			kept = append(kept, other)
		}
	}
	pl.voices = kept
	if pl.current != nil && pl.current.sub == b.sub {
		pl.current = nil
	}
	v := &voice{built: b, length: pl.seconds(b.Length), gain: 1, target: 1}
	if prev := pl.current; prev != nil && pl.cfg.Crossfade > 0 && !prev.fading {
		v.gain = 0
		pl.fade(v, 1, pl.cfg.Crossfade)
		pl.fade(prev, 0, pl.cfg.Crossfade)
		prev.fading = true
	}
	pl.voices = append(pl.voices, v)
	pl.current = v
	pl.playing.Store(int32(b.index))
	if tr := b.Transport; tr != nil { // the section controls steer it
		tr.SetRepeat(pl.sectionRepeat.Load()).SetJump(arrange.JumpAt(pl.sectionJump.Load()))
	}
	pl.sections.Store(b.Transport)
	if b.Panel != nil {
		if vs, ok := pl.values.Load(pl.songs[b.index].Name); ok { // a preset loaded since it was built
			b.Panel.Restore(vs.(map[string]float64))
		}
		pl.panel.Store(&songNow{index: b.index, panel: b.Panel})
	} else {
		pl.panel.Store(nil)
	}
}

// end ends a song: its clock holds, what sounds rings out
func (pl *Playlist) end(v *voice) {
	if !v.ended {
		v.ended = true
		v.sub.HoldClock(true)
		if v.Transport != nil {
			pl.sections.CompareAndSwap(v.Transport, nil)
		}
		if v.Panel != nil {
			if now := pl.panel.Load(); now != nil && now.panel == v.Panel {
				pl.panel.CompareAndSwap(now, nil)
			}
			pl.bindings.Store(pl.songs[v.index].Name, v.Panel.Bindings())
			pl.values.Store(pl.songs[v.index].Name, v.Panel.Snapshot())
		}
	}
}

// sound runs a song for a buffer and adds it to the output at its gain; a song that rang out (or
// faded away) is finished
func (pl *Playlist) sound(v *voice, out psalm.Channels) {
	v.sub.ResetRun()
	v.sub.Run()
	in := v.sub.Output(0).Channels
	loud := false
	for i := range out[0] {
		if v.gain != v.target {
			v.gain += v.step
			if (v.step > 0 && v.gain > v.target) || (v.step < 0 && v.gain < v.target) || v.step == 0 {
				v.gain = v.target
			}
		}
		g := float32(v.gain)
		for c := range out {
			x := in[c%len(in)][i]
			if x > 1e-5 || x < -1e-5 {
				loud = true
			}
			out[c][i] += x * g
		}
	}
	buffer := int64(len(out[0]))
	v.elapsed += buffer
	if !v.ended {
		return
	}
	v.ringing += buffer
	if loud {
		v.quiet = 0
	} else {
		v.quiet += buffer
	}
	switch {
	case v.fading && v.gain == 0 && v.target == 0:
		v.finished = true
	case v.quiet >= pl.seconds(0.5): // rung out
		v.finished = true
	case !v.fading && v.ringing >= pl.seconds(pl.cfg.Tail):
		pl.fade(v, 0, 0.5)
		v.fading = true
	}
}

// Restart goes back to the start of the playlist (psalm.Restarter, from Psalm.Restart): the first
// song that is on plays from its start at once, the song playing fades out quickly (a song asked
// for next is forgotten). Live the first song is kept built for it, so it starts right away
func (pl *Playlist) Restart() {
	first := pl.firstOn()
	if first < 0 {
		return
	}
	pl.queued.Store(-1)
	pl.gapLeft = 0
	cur := pl.current
	if cur != nil && !cur.ended && cur.index == first && !cur.fading {
		// the first song playing: from its start, in place
		cur.sub.Restart()
		cur.elapsed, cur.handed = 0, false
		pl.pending = -1
		for _, v := range pl.voices { // a next song cross fading in already: gone
			if v != cur {
				pl.fade(v, 0, restartFade)
				v.fading = true
				pl.end(v)
			}
		}
		return
	}
	for _, v := range pl.voices {
		if !v.fading {
			pl.end(v)
			pl.fade(v, 0, restartFade)
			v.fading, v.handed = true, true
		}
	}
	pl.current = nil
	pl.pending = first
}

// restartFade is how fast the songs sounding fade away on a restart: quick, without a click
const restartFade = 0.08

// Queue asks for a song to play next (when the song playing ends, or now with Skip); -1 takes
// the request back
func (pl *Playlist) Queue(i int) *Playlist {
	if i >= -1 && i < len(pl.songs) {
		pl.queued.Store(int32(i))
		pl.poke()
	}
	return pl
}

// QueueSong asks for a song by name, it panics on a name the playlist doesn't have
func (pl *Playlist) QueueSong(name string) *Playlist {
	for i, s := range pl.songs {
		if s.Name == name {
			return pl.Queue(i)
		}
	}
	panic(fmt.Sprintf("playlist: no song %q", name))
}

// Skip ends the song playing: it fades out (Config.FadeOut) and the next starts at once
func (pl *Playlist) Skip() { pl.skip.Store(true) }

// SetEnabled switches a song on or off: a song that is off is passed over (it plays when asked for)
func (pl *Playlist) SetEnabled(i int, on bool) *Playlist {
	if i >= 0 && i < len(pl.songs) {
		pl.off[i].Store(!on)
		pl.poke()
	}
	return pl
}

// Playing returns the song playing and how far it is (0-1, 0 for a song without a length), -1 for
// none
func (pl *Playlist) Playing() (int, float64) {
	i := int(pl.playing.Load())
	if i < 0 {
		return -1, 0
	}
	return i, math.Float64frombits(pl.prog.Load())
}

// Done tells whether the playlist ended: the last song played and rang out (never when it loops)
func (pl *Playlist) Done() bool { return pl.done.Load() }

// Duration returns the seconds the songs that are on play one after the other, offline (where
// every song is built): their lengths and the gaps, less the cross fades; the last song's tail is
// not in it. Live it is 0
func (pl *Playlist) Duration() float64 {
	if pl.live {
		return 0
	}
	total, n := 0.0, 0
	for i, b := range pl.all {
		if !pl.off[i].Load() {
			total += b.Length
			n++
		}
	}
	if n > 1 {
		if pl.cfg.Crossfade > 0 {
			total -= float64(n-1) * pl.cfg.Crossfade
		} else {
			total += float64(n-1) * pl.cfg.Gap
		}
	}
	return total
}

// Names returns the names of the songs
func (pl *Playlist) Names() []string {
	names := make([]string, len(pl.songs))
	for i, s := range pl.songs {
		names[i] = s.Name
	}
	return names
}

// Close stops building songs in the background; it returns once a song being built is done
func (pl *Playlist) Close() {
	pl.once.Do(func() {}) // never start the builder after this
	pl.closeOnce.Do(func() { close(pl.wake) })
	if pl.stopped != nil {
		<-pl.stopped
	}
}

// the playlist as a cue list on a panel (panel.Cuer)

func (pl *Playlist) CueNames() []string { return pl.Names() }

func (pl *Playlist) CueState() panel.CueState {
	i, p := pl.Playing()
	st := panel.CueState{Playing: i, Progress: p, Queued: int(pl.queued.Load()), Enabled: make([]bool, len(pl.songs))}
	for k := range pl.off {
		st.Enabled[k] = !pl.off[k].Load()
	}
	return st
}

func (pl *Playlist) QueueCue(i int)           { pl.Queue(i) }
func (pl *Playlist) EnableCue(i int, on bool) { pl.SetEnabled(i, on) }

// Controls puts the playlist on a panel: a tile per song (tap one to play it next, its dot to
// switch it off; the song playing shows how far it is; every tile is a button "<name>/<song>" a
// MIDI pad can play) and a Next button ("<name> next") skipping to the next song
func (pl *Playlist) Controls(ctl *panel.Panel, name string) *Playlist {
	ctl.Cues(name, pl)
	ctl.Button(name+" next", pl.Skip, panel.Label("Next"))
	return pl
}

// sections shows the sections of the song playing as a cue list on a panel (panel.Cuer)
type sections struct{ pl *Playlist }

func (c sections) CueNames() []string {
	if tr := c.pl.sections.Load(); tr != nil {
		return tr.Sections()
	}
	return nil
}

func (c sections) CueState() panel.CueState {
	st := panel.CueState{Playing: -1, Queued: -1}
	tr := c.pl.sections.Load()
	if tr == nil {
		return st
	}
	playing, progress := tr.Playing()
	queued := tr.Queued()
	for i, name := range tr.Sections() {
		if name == playing {
			st.Playing, st.Progress = i, progress
		}
		if name == queued {
			st.Queued = i
		}
		st.Enabled = append(st.Enabled, tr.Enabled(name))
	}
	return st
}

func (c sections) QueueCue(i int) {
	if tr := c.pl.sections.Load(); tr != nil {
		if names := tr.Sections(); i < 0 {
			tr.Queue("")
		} else if i < len(names) {
			tr.Queue(names[i])
		}
	}
}

func (c sections) EnableCue(i int, on bool) {
	if tr := c.pl.sections.Load(); tr != nil {
		if names := tr.Sections(); i >= 0 && i < len(names) {
			tr.SetEnabled(names[i], on)
		}
	}
}

// SetSectionRepeat makes the section playing repeat, in the song playing and in the songs after it
func (pl *Playlist) SetSectionRepeat(on bool) *Playlist {
	pl.sectionRepeat.Store(on)
	if tr := pl.sections.Load(); tr != nil {
		tr.SetRepeat(on)
	}
	return pl
}

// SetSectionJump sets when a section asked for starts, in the song playing and the songs after it
func (pl *Playlist) SetSectionJump(at arrange.JumpAt) *Playlist {
	pl.sectionJump.Store(int32(at))
	if tr := pl.sections.Load(); tr != nil {
		tr.SetJump(at)
	}
	return pl
}

// Sections returns the transport of the song playing (a song with a transport), nil for none
func (pl *Playlist) Sections() *arrange.Transport { return pl.sections.Load() }

// sectionSlots is how many section tiles have a button of their own for MIDI
const sectionSlots = 16

// SectionControls puts the sections of the song playing on a panel (songs built with a
// Transport): a tile per section, changing with the song (tap one to play it next, its dot to
// switch it off), Repeat and Jump at, like a transport's Controls (arrange). The tiles' buttons are
// named by position ("<name>/1", "<name>/2", ...) for MIDI pads; which sections are on is not in
// presets (every song has its own), Repeat and Jump at are, and count for every song
func (pl *Playlist) SectionControls(ctl *panel.Panel, name string) *Playlist {
	ctl.Cues(name, sections{pl}, panel.CueSlots(sectionSlots))
	ctl.Toggle(name+" repeat", panel.Func(func(v float64) { pl.SetSectionRepeat(v >= 0.5) }), panel.Label("Repeat"))
	ctl.Select(name+" jump", []string{"section end", "next bar"}, func(i int) { pl.SetSectionJump(arrange.JumpAt(i)) },
		panel.Label("Jump at"))
	return pl
}

// songPanel mounts the panel of the song playing (panel.Mounter)
type songPanel struct{ pl *Playlist }

func (m songPanel) Mounted() *panel.Panel {
	if now := m.pl.panel.Load(); now != nil {
		return now.panel
	}
	return nil
}

// songNow is the song playing with a panel
type songNow struct {
	index int
	panel *panel.Panel
}

// SnapshotMount gives the values of the controls of every song, as "<song>/<control>": of the song
// playing as they are now, of the others as they were when they played (or were loaded); songs that
// didn't play yet are left out (panel.MountKeeper: they are in the presets of the playlist's panel)
func (m songPanel) SnapshotMount() map[string]float64 {
	values := map[string]float64{}
	now := m.pl.panel.Load()
	m.pl.values.Range(func(k, v any) bool {
		for key, x := range v.(map[string]float64) {
			values[k.(string)+"/"+key] = x
		}
		return true
	})
	if now != nil {
		name := m.pl.songs[now.index].Name
		for key := range values { // the song playing: as it is now
			if strings.HasPrefix(key, name+"/") {
				delete(values, key)
			}
		}
		for key, x := range now.panel.Snapshot() {
			values[name+"/"+key] = x
		}
	}
	return values
}

// RestoreMount sets the controls of the songs: the song playing at once, the others when they are
// built (panel.MountKeeper)
func (m songPanel) RestoreMount(values map[string]float64) {
	per := map[string]map[string]float64{}
	for _, s := range m.pl.songs {
		prefix := s.Name + "/"
		for key, x := range values {
			if rest, ok := strings.CutPrefix(key, prefix); ok {
				if per[s.Name] == nil {
					per[s.Name] = map[string]float64{}
				}
				per[s.Name][rest] = x
			}
		}
	}
	now := m.pl.panel.Load()
	for name, vs := range per {
		m.pl.values.Store(name, vs)
		if now != nil && m.pl.songs[now.index].Name == name {
			now.panel.Restore(vs)
		}
	}
}

// SongPanel returns the panel of the song playing (a song built with a Panel), nil for none
func (pl *Playlist) SongPanel() *panel.Panel { return songPanel{pl}.Mounted() }

// SongControls shows the controls of the song playing on a panel (songs built with a Panel): its
// panel, as on its own page, changing with the song; the heading is the title of the song's panel.
// The control changes and notes of MIDI reach it, MIDI learn binds its controls (kept for the song)
func (pl *Playlist) SongControls(ctl *panel.Panel, name string) *Playlist {
	ctl.Mount(name, songPanel{pl})
	return pl
}
