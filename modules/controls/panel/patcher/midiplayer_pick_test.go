package patcher_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
)

// the player offers the MIDI files of its folder to pick from, and a pick (typed into the file text) loads it
func TestMidiPlayerPicker(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	dir := t.TempDir()
	f := &midi.File{Format: 1, Division: 480, Tracks: []midi.Track{{}}}
	f.Tracks[0].Add(0, midi.NoteOnMsg(1, 60, 100))
	f.Tracks[0].Add(480, midi.NoteOffMsg(1, 60, 0))
	if err := f.WriteFile(filepath.Join(dir, "a.mid")); err != nil {
		t.Fatal(err)
	}
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), MIDI: dir}))
	id, err := pt.Add("midiplayer", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	if err := pt.Upload(id, "file", "../my song.MID", []byte("not a midi file")); err == nil {
		t.Error("a file that is not MIDI is kept")
	}
	if err := pt.Upload(id, "file", "../my song.MID", buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	want := filepath.ToSlash(filepath.Join(dir, "my_song.mid"))
	if _, err := os.Stat(want); err != nil {
		t.Errorf("the file is not kept in the folder: %v", err)
	}
	if got := pt.Texts(id)[0].Value; got != want {
		t.Errorf("the file text is %q, want %q", got, want)
	}
	if e := pt.TextError(id, "file"); e != "" {
		t.Error(e)
	}
	if st := patcher.StateJSON(pt); !strings.Contains(st, `"choices":["`+filepath.ToSlash(filepath.Join(dir, "a.mid"))+`","`+want+`"]`) {
		t.Errorf("the file text has no choices: %s", st)
	}
}
