package patcher_test

import (
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
	"github.com/almerlucke/psalm/modules/playlist"
)

// patches kept, played as the songs of a playlist (the setlist example): a patch with a song ends
// with it, then the next plays
func TestPatchesAsSongs(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	dir := t.TempDir()
	// two patches: a song of one bar (2 s), a clock without a song
	{
		p := psalm.New(nil)
		pt := patcher.New(p, panel.New(p, "Make"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}), patcher.Keep(dir))
		song, _ := pt.Add("song", "", 0, 0)
		_ = pt.SetText(song, "sections", "A 1")
		if err := pt.Save("one bar"); err != nil {
			t.Fatal(err)
		}
		pt.Clear()
		_, _ = pt.Add("clock", "", 0, 0)
		if err := pt.Save("no song"); err != nil {
			t.Fatal(err)
		}
	}
	p := psalm.New(nil)
	build := func(name string) playlist.Song {
		return playlist.Song{Name: name, Build: func(sp *psalm.Psalm) playlist.Built {
			ctl := panel.New(sp, name)
			pt := patcher.New(sp, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}), patcher.Keep(dir))
			if err := pt.Load(name); err != nil {
				t.Error(err)
			}
			b := playlist.Built{Out: pt.Out(), Panel: ctl}
			if tr := patcher.SongOf(sp.Env()).Transport(); tr != nil {
				b.Transport, b.Length = tr, tr.Form().Beats()*60/pt.Tempo().Value()
			}
			return b
		}}
	}
	pl := playlist.New(p, playlist.Config{Gap: 0.5}, build("one bar"), build("no song"))
	defer pl.Close()
	p.Out(pl)
	run := func(seconds float64) {
		for range int(seconds * 44100 / 256) {
			p.ResetRun()
			p.Run()
		}
	}
	run(1)
	if i, _ := pl.Playing(); i != 0 {
		t.Fatalf("playing %d at first, want the song of one bar", i)
	}
	run(3) // its bar, the gap
	if i, _ := pl.Playing(); i != 1 {
		t.Errorf("playing %d after the bar, want the next patch", i)
	}
}
