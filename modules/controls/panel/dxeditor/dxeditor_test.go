package dxeditor

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/synths/dx"
)

func TestEditor(t *testing.T) {
	p := psalm.New(nil)
	start := dx.Presets["bellPad"]()
	synth := dx.NewPoly(start, 2, 2)
	synth.AddTo(p)
	ctl := panel.New(p, "Test")
	other := dx.Presets["woodBass"]()
	other.Name = "WOOD"
	e := Add(ctl, "DX", synth, start, Bank([]dx.Patch{start, other}))
	srv := httptest.NewServer(ctl.Handler())
	defer srv.Close()
	var ts int64
	tick := func() {
		for range 2 { // a buffer: the values reach the editor, the next applies them
			ts += int64(synth.Config().BufferSize)
			ctl.CtrlTick(psalm.CtrlTick{SampleTimestamp: ts})
		}
	}
	set := func(name string, v string) {
		res, err := http.Post(srv.URL+"/api/set", "application/json", strings.NewReader(`{"name": "`+name+`", "value": `+v+`}`))
		if err != nil || res.StatusCode != 200 {
			t.Fatalf("set %s: %v %v", name, err, res.Status)
		}
	}
	voice := func() dx.Patch {
		var got dx.Patch
		synth.CallVoices(func(v *dx.Voice) { got = v.Patch() })
		return got
	}
	tick()
	if v := voice(); v.Algorithm != start.Algorithm || v.Ops[2].Coarse != start.Ops[2].Coarse {
		t.Errorf("the synth should play the patch it started with")
	}

	// a setting of an operator, the algorithm
	set("DX op3Coarse", "7")
	set("DX", "17")
	tick()
	if v := voice(); v.Ops[2].Coarse != 7 || v.Algorithm != 16 || v.Routing != nil {
		t.Errorf("edited: coarse %d, algorithm %d", v.Ops[2].Coarse, v.Algorithm+1)
	}

	// an algorithm of your own: 6 into 1, 5 into 1, carriers 1 and 2, feedback on 5
	for _, c := range []struct{ name, v string }{{"DX carrier1", "1"}, {"DX carrier2", "1"}, {"DX carrier3", "0"},
		{"DX carrier4", "0"}, {"DX carrier5", "0"}, {"DX carrier6", "0"}, {"DX fbFrom", "5"}, {"DX fbTo", "5"}} {
		set(c.name, c.v)
	}
	for from := 2; from <= 6; from++ {
		for to := 1; to < from; to++ {
			on := "0"
			if to == 1 && (from == 6 || from == 5) {
				on = "1"
			}
			set("DX mod"+string(rune('0'+from))+string(rune('0'+to)), on)
		}
	}
	set("DX custom", "1")
	tick()
	r := voice().Routing
	if r == nil || len(r.Carriers) != 2 || len(r.Mods) != 2 || r.FeedbackFrom != 4 {
		t.Fatalf("my own algorithm: %+v", r)
	}
	set("DX custom", "0")
	tick()
	if voice().Routing != nil {
		t.Errorf("custom off: the algorithm again")
	}

	// loading a patch of the bank moves every setting to it
	set("DX program", "1")
	tick()
	tick()
	if v := voice(); v.Ops[0].Coarse != other.Ops[0].Coarse || v.Algorithm != other.Algorithm || v.Ops[1].EGRates != other.Ops[1].EGRates {
		t.Errorf("loaded: %+v", v.Ops[0])
	}
	if e.Setting("op1Coarse").Value() != float64(other.Ops[0].Coarse) || e.patchName() != "WOOD" {
		t.Errorf("the controls didn't follow the patch loaded")
	}
}

func TestPatchesOfYourOwn(t *testing.T) {
	p := psalm.New(nil)
	start := dx.Presets["bellPad"]()
	synth := dx.NewPoly(start, 2, 2)
	synth.AddTo(p)
	ctl := panel.New(p, "Test")
	e := Add(ctl, "DX", synth, start)
	if err := e.Keep(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(ctl.Handler())
	defer srv.Close()
	var ts int64
	tick := func() {
		for range 2 {
			ts += int64(synth.Config().BufferSize)
			ctl.CtrlTick(psalm.CtrlTick{SampleTimestamp: ts})
		}
	}
	msg := func(body string) {
		res, err := http.Post(srv.URL+"/api/message", "application/json", strings.NewReader(`{"name": "DX", "data": {"presets": `+body+`}}`))
		if err != nil || res.StatusCode != 200 {
			t.Fatal(err)
		}
		tick()
	}
	tick()
	e.Setting("op3Coarse").Set(9)
	e.Set(20)
	e.Setting("custom").Set(1) // an algorithm of my own, kept too
	tick()
	msg(`{"save": "mine"}`)
	for i := 0; len(e.Presets().Mine()) == 0 && i < 200; i++ {
		time.Sleep(5 * time.Millisecond)
	}
	e.Setting("op3Coarse").Set(1)
	e.Set(3)
	e.Setting("custom").Set(0)
	tick()
	msg(`{"load": "mine", "mine": true}`)
	tick()
	var v dx.Patch
	synth.CallVoices(func(x *dx.Voice) { v = x.Patch() })
	if v.Ops[2].Coarse != 9 || v.Algorithm != 19 || v.Routing == nil || e.patchName() != "mine" {
		t.Errorf("loaded back: coarse %d, algorithm %d, routing %v, name %q", v.Ops[2].Coarse, v.Algorithm+1, v.Routing != nil, e.patchName())
	}
}

// the filter after the operators is set like the other settings
func TestEditorFilter(t *testing.T) {
	p := psalm.New(nil)
	start := dx.Presets["bellPad"]()
	synth := dx.NewPoly(start, 1, 2)
	synth.AddTo(p)
	ctl := panel.New(p, "Test")
	e := Add(ctl, "DX", synth, start)
	for _, k := range []string{"filterType", "filterCutoff", "filterResonance", "filterEnvAmount", "filterKeyTrack", "filterAttack", "filterDecay", "filterSustain", "filterRelease"} {
		if e.Setting(k) == nil {
			t.Errorf("no %s", k)
		}
	}
	e.Setting("filterType").Set(1)
	e.Setting("filterCutoff").Set(40)
	var ts int64
	for range 2 {
		ts += int64(synth.Config().BufferSize)
		ctl.CtrlTick(psalm.CtrlTick{SampleTimestamp: ts})
	}
	var got dx.Patch
	synth.CallVoices(func(v *dx.Voice) { got = v.Patch() })
	if !got.FilterSet || got.Filter.Type != 1 || got.Filter.Cutoff != 40 {
		t.Errorf("the voice plays filter %+v (set %v)", got.Filter, got.FilterSet)
	}
}

// a patch of exact operators loaded: the synth plays its exact values, not only the 0-99 ones
func TestEditorLoadsExactOperators(t *testing.T) {
	p := psalm.New(nil)
	start := dx.Presets["bellPad"]()
	synth := dx.NewPoly(start, 1, 2)
	synth.AddTo(p)
	ctl := panel.New(p, "Test")
	e := Add(ctl, "DX", synth, start)
	flute := dx.Presets["flute"]()
	e.Load(flute)
	var ts int64
	for range 4 {
		ts += int64(synth.Config().BufferSize)
		ctl.CtrlTick(psalm.CtrlTick{SampleTimestamp: ts})
	}
	var got dx.Patch
	synth.CallVoices(func(v *dx.Voice) { got = v.Patch() })
	if !got.Ops[0].FreeOn || got.Ops[0].Free != flute.Ops[0].Free || got.Level != flute.Level || got.VibratoDepth != flute.VibratoDepth {
		t.Errorf("the synth plays %+v (level %v), want the flute's exact values", got.Ops[0], got.Level)
	}
}

func sysexEditor(t *testing.T) (*Editor, string, []dx.Patch) {
	dir := t.TempDir()
	var few []dx.Patch
	for i, n := range []string{"bellPad", "woodBass", "bellPad"} {
		p := dx.Presets[n]()
		p.Name = fmt.Sprintf("V%d-%s", i, strings.ToUpper(n[:4]))
		few = append(few, p)
	}
	if err := os.WriteFile(filepath.Join(dir, "few.syx"), dx.EncodeBank(few), 0o644); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644)
	start := dx.Presets["bellPad"]()
	synth := dx.NewPoly(start, 2, 2)
	ctl := panel.New(psalm.New(nil), "Test")
	e := Add(ctl, "DX", synth, start, Bank([]dx.Patch{start}), SysExFolder(dir))
	return e, dir, few
}

func TestSysExFolder(t *testing.T) {
	e, dir, few := sysexEditor(t)
	if f := e.SysExFiles(); len(f) != 1 || f[0] != "few.syx" {
		t.Fatalf("files %v", f)
	}
	if err := e.LoadSysEx("few"); err != nil {
		t.Fatal(err)
	}
	st := e.sysexState()
	if st.File != "few.syx" || len(st.Bank) != 32 || st.Bank[1] != few[1].Name {
		t.Errorf("bank %q %d", st.File, len(st.Bank))
	}
	for _, bad := range []string{"../few", "a/b", `a\b`, "", ".hidden", "x..y", "a\x00b", ".syx"} {
		if _, err := e.SaveSysEx(bad, false, true); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
	if err := e.LoadSysEx("../few"); err == nil {
		t.Error("a path should be rejected")
	}
	if err := e.LoadSysEx("missing"); err == nil {
		t.Error("a missing file should be rejected")
	}

	// the patch saved as a single dump, read back; not replaced without saying so
	e.Load(few[0])
	e.parts["op3Coarse"].Set(9)
	file, err := e.SaveSysEx("mine.SYX", false, false)
	if err != nil || file != "mine.syx" {
		t.Fatalf("save %q %v", file, err)
	}
	back, err := dx.LoadSysEx(filepath.Join(dir, "mine.syx"))
	if err != nil || len(back) != 1 || back[0].Ops[2].Coarse != 9 || back[0].Name != few[0].Name {
		t.Fatalf("read back %v %v", back, err)
	}
	if _, err := e.SaveSysEx("mine", false, false); !errors.Is(err, os.ErrExist) {
		t.Errorf("replaced without saying so: %v", err)
	}
	if _, err := e.SaveSysEx("mine", true, true); err != nil {
		t.Fatal(err)
	}
	if back, err = dx.LoadSysEx(filepath.Join(dir, "mine.syx")); err != nil || len(back) != 32 {
		t.Errorf("bank read back %d %v", len(back), err)
	}

	// through the messages of the page
	e.sysexMessage([]byte(`{"save": "../x"}`))
	if st := e.sysexState(); !st.Error {
		t.Error("the bad name should be an error on the page")
	}
	e.sysexMessage([]byte(`{"load": ""}`))
	if st := e.sysexState(); st.File != "" || len(st.Bank) != 1 || len(st.Files) != 2 {
		t.Errorf("built in again: %+v", st)
	}
}

func TestSysExOff(t *testing.T) {
	start := dx.Presets["bellPad"]()
	ctl := panel.New(psalm.New(nil), "Test")
	e := Add(ctl, "DX", dx.NewPoly(start, 2, 2), start, Bank([]dx.Patch{start}))
	if e.sysexState() != nil || e.LoadSysEx("a") == nil {
		t.Error("off without a folder")
	}
	if _, err := e.SaveSysEx("a", false, false); err == nil {
		t.Error("no folder to save into")
	}
}
