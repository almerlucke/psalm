package playlist

import (
	"math"
	"slices"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/panel"
)

// tone sounds a level while the clock of its song goes, rings 5 buffers after it holds, and keeps
// its ticks
type tone struct {
	*psalm.Module
	level float32
	song  *psalm.Psalm
	ticks []int64
	held  int
}

func (t *tone) CtrlTick(tick psalm.CtrlTick) { t.ticks = append(t.ticks, tick.TickTimestamp) }
func (t *tone) Synthesize() {
	v := float32(0)
	if !t.song.ClockHeld() {
		t.held = 0
	} else {
		t.held++
	}
	if t.held <= 5 {
		v = t.level
	}
	for _, ch := range t.Output(0).Channels {
		for i := range ch {
			ch[i] = v
		}
	}
}

// song is a test song: a tone at a level, playing seconds, at a tempo (0 is the config's)
func song(name string, level float32, seconds, bpm float64, tones map[string]*tone) Song {
	return Song{Name: name, Build: func(p *psalm.Psalm) Built {
		if bpm > 0 {
			p.Env().SetBPM(bpm)
		}
		t := &tone{Module: psalm.NewModule(nil, []int{2}), level: level, song: p}
		t.SetSelf(t)
		t.AddTo(p)
		tones[name] = t
		return Built{Out: t, Length: seconds}
	}}
}

// rig is a host with a playlist of A, B and C (levels 0.1, 0.2, 0.4)
type rig struct {
	p     *psalm.Psalm
	pl    *Playlist
	tones map[string]*tone
}

func newRig(t *testing.T, cfg Config, p *psalm.Psalm) *rig {
	t.Helper()
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 96, SampleRate: 1000, BufferSize: 10})
	t.Cleanup(func() { psalm.PopConfig() })
	if p == nil {
		p = psalm.New(nil)
	}
	r := &rig{p: p, tones: map[string]*tone{}}
	r.pl = New(p, cfg,
		song("A", 0.1, 1, 0, r.tones),
		song("B", 0.2, 1, 60, r.tones),
		song("C", 0.4, 1, 0, r.tones))
	p.Out(r.pl)
	return r
}

// levels runs buffers and returns the level of each
func (r *rig) levels(n int) []float64 {
	var ls []float64
	for range n {
		r.p.ResetRun()
		r.p.Run()
		ls = append(ls, math.Round(float64(r.pl.Output(0).Channels[0][9])*1000)/1000)
	}
	return ls
}

// songs returns the songs heard over buffers, by their level (a level per buffer, repeats left out)
func names(ls []float64) []string {
	byLevel := map[float64]string{0: "-", 0.1: "A", 0.2: "B", 0.4: "C"}
	var got []string
	for _, l := range ls {
		n, ok := byLevel[l]
		if !ok {
			continue // two songs at once: a fade, or one ringing out
		}
		if len(got) == 0 || got[len(got)-1] != n {
			got = append(got, n)
		}
	}
	return got
}

func TestInOrder(t *testing.T) {
	r := newRig(t, Config{}, nil)
	if d := r.pl.Duration(); d != 3 {
		t.Fatalf("offline the duration should be known: %v", d)
	}
	got := names(r.levels(400)) // 4 s
	if !slices.Equal(got, []string{"A", "B", "C", "-"}) || !r.pl.Done() {
		t.Fatalf("the songs should play in order, then the playlist ends: %v", got)
	}
	// every song has its own clock, from 0 and at its tempo: B at 60 is half as many ticks
	a, b := r.tones["A"], r.tones["B"]
	if a.ticks[0] != 1 || b.ticks[0] != 1 {
		t.Fatalf("every song should count its ticks from the start: %d %d", a.ticks[0], b.ticks[0])
	}
	if na, nb := len(a.ticks), len(b.ticks); math.Abs(float64(na)/float64(nb)-2) > 0.05 {
		t.Fatalf("B at half the tempo should tick half as often: %d against %d", nb, na)
	}
}

func TestGapAndOff(t *testing.T) {
	r := newRig(t, Config{Gap: 0.5}, nil)
	r.pl.SetEnabled(1, false)
	ls := r.levels(300)
	if got := names(ls); !slices.Equal(got, []string{"A", "-", "C", "-"}) {
		t.Fatalf("a song that is off is passed over, with the gap between: %v", got)
	}
	if d := r.pl.Duration(); d != 2.5 {
		t.Fatalf("the duration should leave B out and count the gap: %v", d)
	}
}

func TestQueueAndSkip(t *testing.T) {
	r := newRig(t, Config{FadeOut: 0.2}, nil)
	r.levels(30)
	r.pl.Queue(2)
	if st := r.pl.CueState(); st.Playing != 0 || st.Queued != 2 || st.Progress < 0.25 || st.Progress > 0.35 {
		t.Fatalf("the tiles should show A playing, C next: %+v", st)
	}
	r.pl.Skip()
	ls := r.levels(5)
	if ls[0] <= 0.4 || ls[4] <= 0.4 {
		t.Fatalf("a skip should start C at once with A fading over it: %v", ls)
	}
	ls = r.levels(200)
	if got := names(ls); got[0] != "C" || slices.Contains(got, "B") {
		t.Fatalf("after the fade C alone, the queue taken (no B): %v", got)
	}
}

func TestCrossfade(t *testing.T) {
	r := newRig(t, Config{Crossfade: 0.4}, nil)
	ls := r.levels(100) // A, and B fading in from 0.6 s
	if ls[55] != 0.1 || math.Abs(ls[79]-0.15) > 0.01 || ls[99] < 0.19 {
		t.Fatalf("B should fade in over A's last 0.4 seconds, A out: %v %v %v", ls[55], ls[79], ls[99])
	}
	if d := r.pl.Duration(); math.Abs(d-2.2) > 1e-9 {
		t.Fatalf("the cross fades should shorten the duration: %v", d)
	}
}

func TestLoopAndRestart(t *testing.T) {
	r := newRig(t, Config{Loop: true}, nil)
	if got := names(r.levels(400)); !slices.Equal(got, []string{"A", "B", "C", "A"}) || r.pl.Done() {
		t.Fatalf("a playlist that loops starts again: %v", got)
	}
	r.levels(20) // into B
	a := r.tones["A"]
	r.pl.Queue(2)
	r.p.Restart()
	a.ticks = nil
	ls := r.levels(30)
	if len(a.ticks) == 0 || a.ticks[0] != 1 {
		t.Fatal("a restart should play the first song from its start")
	}
	if i, p := r.pl.Playing(); i != 0 || p > 0.35 || r.pl.CueState().Queued != -1 {
		t.Fatalf("after a restart A plays, the song asked for is forgotten: %d %v", i, p)
	}
	if got := names(ls); got[len(got)-1] != "A" || slices.Contains(got, "C") {
		t.Fatalf("B should fade quickly into A: %v", got)
	}
	r.p.Restart() // A playing: from its start, in place
	a.ticks = nil
	r.levels(1)
	if i, p := r.pl.Playing(); i != 0 || p > 0.02 || a.ticks[0] != 1 {
		t.Fatal("a restart in the first song should start it again")
	}
}

func TestControls(t *testing.T) {
	r := newRig(t, Config{}, nil)
	ctl := panel.New(r.p, "Test")
	r.pl.Controls(ctl, "Songs")
	cs := ctl.CuesByName("Songs")
	if cs == nil || !slices.Equal(cs.Names, []string{"A", "B", "C"}) || ctl.Param("Songs next") == nil || ctl.Param("Songs/B") == nil {
		t.Fatal("the controls should add the tiles and the next button")
	}
	cs.Queue(2)
	cs.Enable(1, false)
	if st := r.pl.CueState(); st.Queued != 2 || st.Enabled[1] {
		t.Fatal("the tiles should queue and switch songs")
	}
}

// engine is a sound card that plays when the test runs the patch
type engine struct{}

func (engine) GetSampleRate() float64                { return 1000 }
func (engine) GetBufferSize() int                    { return 10 }
func (engine) SetCallback(func(in, out [][]float32)) {}
func (engine) Init() error                           { return nil }
func (engine) Start() error                          { return nil }
func (engine) Stop() error                           { return nil }
func (engine) Close() error                          { return nil }

func TestLiveBuildsAhead(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 96, SampleRate: 1000, BufferSize: 10})
	p := psalm.New(engine{})
	psalm.PopConfig()
	r := newRig(t, Config{}, p)
	defer r.pl.Close()
	if len(r.tones) != 1 {
		t.Fatalf("live only the first song is built at the start, got %d", len(r.tones))
	}
	var ls []float64
	deadline := time.Now().Add(5 * time.Second)
	for len(names(ls)) < 4 && time.Now().Before(deadline) {
		ls = append(ls, r.levels(10)...)
		time.Sleep(time.Millisecond) // the builder works in the background
	}
	if got := names(ls); len(got) < 3 || !slices.Equal(got[:3], []string{"A", "B", "C"}) {
		t.Fatalf("live the songs should be built ahead and play in order: %v", got)
	}
	if b := r.tones["B"]; b.ticks[0] != 1 {
		t.Fatal("a song built ahead should start at its first tick")
	}

	// a restart: the first song is kept built, it starts at once
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		r.pl.buildMu.Lock()
		ok := r.pl.first != nil
		r.pl.buildMu.Unlock()
		if ok {
			break
		}
		time.Sleep(time.Millisecond)
	}
	r.p.Restart()
	r.levels(2)
	if i, _ := r.pl.Playing(); i != 0 {
		t.Fatalf("live a restart should start the first song at once, playing %d", i)
	}
}

// sectioned is a test song of three sections of a bar (2 seconds at 120): a tone, ended by its
// transport
func sectioned(name string, level float32, tones map[string]*tone) Song {
	return Song{Name: name, Build: func(p *psalm.Psalm) Built {
		t := &tone{Module: psalm.NewModule(nil, []int{2}), level: level, song: p}
		t.SetSelf(t)
		t.AddTo(p)
		tones[name] = t
		form := arrange.NewForm(arrange.Section{Name: "One", Bars: 1}, arrange.Section{Name: "Two", Bars: 1}, arrange.Section{Name: "Three", Bars: 1})
		a := arrange.NewArrangement(form)
		a.AddTo(p)
		return Built{Out: t, Length: a.Duration(), Transport: form.Transport()}
	}}
}

func TestSectionsOfTheSongPlaying(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 96, SampleRate: 1000, BufferSize: 10})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	tones := map[string]*tone{}
	pl := New(p, Config{}, sectioned("A", 0.1, tones), sectioned("B", 0.2, tones))
	p.Out(pl)
	ctl := panel.New(p, "Test")
	pl.SectionControls(ctl, "Sections")
	cs := ctl.CuesByName("Sections")
	run := func(n int) []float64 {
		var ls []float64
		for range n {
			p.ResetRun()
			p.Run()
			ls = append(ls, math.Round(float64(pl.Output(0).Channels[0][9])*1000)/1000)
		}
		return ls
	}
	run(10)
	if st := cs.State(); !slices.Equal(st.Names, []string{"One", "Two", "Three"}) || st.Playing != 0 {
		t.Fatalf("the tiles should show the sections of the song playing: %+v", st)
	}
	cs.Enable(1, false) // A without its second section: it ends after 4 seconds, not 6
	ls := run(450)
	if ls[390] != 0.1 || ls[420] != 0.2 {
		t.Fatalf("A should end when its transport ended (at 4 s), B start: %v %v", ls[390], ls[420])
	}
	if st := cs.State(); st.Playing != 0 || st.Enabled[1] != true {
		t.Fatalf("the tiles should show B's sections now, as written: %+v", st)
	}
	ctl.Param("Sections repeat").Set(1)
	ctl.CtrlTick(psalm.CtrlTick{TickTimestamp: 1})
	if !pl.Sections().Repeat() {
		t.Fatal("repeat should reach the transport of the song playing")
	}
	ls = run(800) // B's first section again and again: still B after 8 seconds
	if ls[799] != 0.2 || pl.Sections().Ended() {
		t.Fatalf("a repeated section should keep the song going: %v", ls[799])
	}
	pl.SetSectionRepeat(false)
	run(900) // the rest of the section and the two after it, and ringing out
	if !pl.Done() || pl.Sections() != nil || len(cs.State().Names) != 0 {
		t.Fatal("after the last song ended there are no sections to show")
	}
}

func TestSongPanels(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 96, SampleRate: 1000, BufferSize: 10})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	panels := map[string]*panel.Panel{}
	withPanel := func(s Song) Song {
		build := s.Build
		s.Build = func(sub *psalm.Psalm) Built {
			b := build(sub)
			ctl := panel.New(sub, s.Name)
			ctl.Fader("Level", 0, 1, panel.Linear, nil)
			panels[s.Name] = ctl
			b.Panel = ctl
			return b
		}
		return s
	}
	tones := map[string]*tone{}
	pl := New(p, Config{}, withPanel(song("A", 0.1, 1, 0, tones)), withPanel(song("B", 0.2, 1, 0, tones)))
	p.Out(pl)
	host := panel.New(p, "Host")
	pl.SongControls(host, "Song")
	run := func(n int) {
		for range n {
			p.ResetRun()
			p.Run()
		}
	}
	run(10)
	if pl.SongPanel() != panels["A"] || host.CuesByName("Song") != nil {
		t.Fatal("the panel of A should be mounted while A plays")
	}
	panels["A"].Param("Level").BindCC(0, 21) // as MIDI learn would
	run(100)
	if pl.SongPanel() != panels["B"] {
		t.Fatal("the panel of B should be mounted while B plays")
	}
	bs, ok := pl.bindings.Load("A")
	if !ok || bs.(map[string]panel.Binding)["Level"].Controller != 21 {
		t.Fatal("what was bound in A should be kept for A")
	}
	run(200)
	if pl.SongPanel() != nil {
		t.Fatal("after the last song nothing is mounted")
	}
	again := pl.build(0) // A built again (live, when it plays again)
	if b, ok := again.Panel.Param("Level").Binding(); !ok || b.Controller != 21 {
		t.Fatal("a song built again should get back what was bound for it")
	}
}

func TestSongValuesInPresets(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 96, SampleRate: 1000, BufferSize: 10})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	panels := map[string]*panel.Panel{}
	withPanel := func(s Song) Song {
		build := s.Build
		s.Build = func(sub *psalm.Psalm) Built {
			b := build(sub)
			ctl := panel.New(sub, s.Name)
			ctl.Fader("Level", 0, 1, panel.Linear, nil, panel.Default(0.5))
			panels[s.Name] = ctl
			b.Panel = ctl
			return b
		}
		return s
	}
	tones := map[string]*tone{}
	pl := New(p, Config{Loop: true}, withPanel(song("A", 0.1, 1, 0, tones)), withPanel(song("B", 0.2, 1, 0, tones)))
	p.Out(pl)
	host := panel.New(p, "Host")
	host.Fader("Volume", 0, 1, panel.Linear, nil)
	pl.SongControls(host, "Song")
	run := func(n int) {
		for range n {
			p.ResetRun()
			p.Run()
		}
	}
	run(10)
	panels["A"].Param("Level").Set(0.8)
	run(100) // into B
	panels["B"].Param("Level").Set(0.3)
	snap := host.Snapshot()
	if snap["Song/A/Level"] != 0.8 || snap["Song/B/Level"] != 0.3 {
		t.Fatalf("the presets should hold the controls of every song that played: %v", snap)
	}

	// a preset loaded: B (playing) at once, A when it plays again
	snap["Song/A/Level"], snap["Song/B/Level"] = 0.1, 0.9
	host.Restore(snap)
	if panels["B"].Param("Level").Value() != 0.9 {
		t.Fatal("the song playing should get its values from the preset at once")
	}
	run(100) // A again (loop): offline the same panel
	if pl.SongPanel() != panels["A"] || panels["A"].Param("Level").Value() != 0.1 {
		t.Fatalf("a song starting should get its values from the preset: %v", panels["A"].Param("Level").Value())
	}
	again := pl.build(1) // B built again (live)
	if again.Panel.Param("Level").Value() != 0.9 {
		t.Fatal("a song built again should keep its values")
	}
}
