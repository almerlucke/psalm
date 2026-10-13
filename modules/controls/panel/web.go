package panel

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"
)

//go:embed page.html
var page []byte

// updateInterval is how often the page gets the values and meters
const updateInterval = 50 * time.Millisecond

// Serve serves the panel as a web page on an address and blocks until Close; run it on its own
// goroutine before playing the patch, and Close the panel when the program stops:
//
//	go ctl.Serve("localhost:8087") // this computer only
//	go ctl.Serve(":8087")          // also from a phone or tablet on the same network
//	defer ctl.Close()
//
// Every page open on the panel shows the same values: a fader moved on one moves on the others
func (pn *Panel) Serve(addr string) error {
	ctx, cancel := context.WithCancel(context.Background())
	srv := &http.Server{Addr: addr, Handler: pn.Handler(), BaseContext: func(net.Listener) context.Context { return ctx }}
	pn.srvMu.Lock()
	if pn.closed {
		pn.srvMu.Unlock()
		cancel()
		return nil
	}
	pn.servers = append(pn.servers, server{srv, cancel})
	pn.srvMu.Unlock()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			return fmt.Errorf("panel: %s is in use: is an earlier run of this program (or another one) still running? %w", addr, err)
		}
		return err
	}
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Close stops serving: the pages lose their connection, the address is free again
func (pn *Panel) Close() error {
	pn.srvMu.Lock()
	pn.closed = true
	servers := pn.servers
	pn.servers = nil
	pn.srvMu.Unlock()

	var first error
	for _, s := range servers {
		s.cancel() // ends the event streams, which would keep Shutdown waiting
		ctx, done := context.WithTimeout(context.Background(), closeTimeout)
		if err := s.Shutdown(ctx); err != nil && first == nil {
			first = err
		}
		done()
	}
	return first
}

// closeTimeout is how long Close waits for requests to finish
const closeTimeout = 2 * time.Second

// server is a server of the panel, and what ends its event streams
type server struct {
	*http.Server
	cancel context.CancelFunc
}

// Handler returns the web page and its API as an http.Handler, to serve it yourself:
//
//	GET  /              the page
//	GET  /api/panel     the controls: their kinds, ranges and values (JSON)
//	POST /api/set       {"name": "Cutoff", "value": 2400} sets a control
//	POST /api/press     {"name": "Bells"} presses a button
//	GET  /api/events    the values and the meters as they change (server sent events)
//	GET  /api/snapshot  the values of all controls (JSON); POST one back to restore it
//	POST /api/learn     {"name": "Cutoff"} arms a control for MIDI learn, {"name": ""} stops
//	POST /api/unbind    {"name": "Cutoff"} takes its controller from a control
//	POST /api/grid      {"name": "Drums", "pattern": 1, "set": [[pattern, row, step, field, value], ...]}
//	                    sets cells of a grid and/or asks for a pattern
//	POST /api/cues      {"name": "Song", "queue": 2} asks for a cue next (-1 takes it back),
//	                    {"name": "Song", "cue": 2, "on": false} switches one off
//	POST /api/audio     {"running": false} stops the sound of the patch, true starts it, {"restart": true}
//	                    sets it back to the start (see SetPlayer)
//	POST /api/message   {"name": "Notes", "data": {...}} sends anything to a custom control's OnMessage
//	GET  /theme.css     the stylesheet of a theme (?name=, the panel's own without; see Theme)
//	GET  /plugins/<name>/plugin.js, plugin.css, <file>   a plugin's script, styles and files (see Plugin)
//	GET  /api/presets   {"enabled": true, "names": [...]}: the saved setups (see Presets)
//	POST /api/presets   {"action": "save" (or "load", "delete"), "name": "Groove"}
func (pn *Panel) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	})
	mux.HandleFunc("GET /api/panel", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, pn.describe())
	})
	mux.HandleFunc("POST /api/set", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name  string  `json:"name"`
			Value float64 `json:"value"`
		}
		if par := pn.decode(w, r, &req, func() string { return req.Name }); par != nil {
			par.Set(req.Value)
			writeJSON(w, map[string]float64{"value": par.Value()})
		}
	})
	mux.HandleFunc("POST /api/press", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if par := pn.decode(w, r, &req, func() string { return req.Name }); par != nil {
			par.Press()
			w.WriteHeader(http.StatusNoContent)
		}
	})
	mux.HandleFunc("GET /api/snapshot", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("download") { // a file to keep, named after the panel
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", pn.title+".json"))
		}
		writeJSON(w, pn.Snapshot())
	})
	mux.HandleFunc("GET /api/presets", func(w http.ResponseWriter, r *http.Request) {
		names, err := pn.PresetNames()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"enabled": pn.presetsEnabled(), "names": names})
	})
	mux.HandleFunc("POST /api/presets", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Action string `json:"action"`
			Name   string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var err error
		switch req.Action {
		case "save":
			err = pn.SavePreset(req.Name)
		case "load":
			err = pn.LoadPreset(req.Name)
		case "delete":
			err = pn.DeletePreset(req.Name)
		default:
			err = fmt.Errorf("no action %q: save, load or delete", req.Action)
		}
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, os.ErrNotExist) {
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}
		names, _ := pn.PresetNames()
		writeJSON(w, map[string]any{"enabled": true, "names": names})
	})
	mux.HandleFunc("POST /api/snapshot", func(w http.ResponseWriter, r *http.Request) {
		var values map[string]float64
		if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		pn.Restore(values)
		writeJSON(w, pn.Snapshot())
	})
	mux.HandleFunc("POST /api/learn", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			pn.StopLearning()
			w.WriteHeader(http.StatusNoContent)
			return
		}
		par := pn.Param(req.Name)
		if par == nil || par.Kind == MeterKind {
			http.Error(w, fmt.Sprintf("no control named %q to learn", req.Name), http.StatusNotFound)
			return
		}
		par.Learn()
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /api/unbind", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if par := pn.decode(w, r, &req, func() string { return req.Name }); par != nil {
			par.Unbind()
			w.WriteHeader(http.StatusNoContent)
		}
	})
	mux.HandleFunc("POST /api/grid", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name    string       `json:"name"`
			Pattern *int         `json:"pattern"`
			Set     [][5]float64 `json:"set"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		g := pn.GridByName(req.Name)
		if g == nil {
			http.Error(w, fmt.Sprintf("no grid named %q", req.Name), http.StatusNotFound)
			return
		}
		for _, c := range req.Set {
			g.Set(int(c[0]), int(c[1]), int(c[2]), int(c[3]), c[4])
		}
		if req.Pattern != nil {
			g.SetPattern(*req.Pattern)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /api/cues", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name  string `json:"name"`
			Queue *int   `json:"queue"`
			Cue   *int   `json:"cue"`
			On    *bool  `json:"on"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cs := pn.CuesByName(req.Name)
		if cs == nil {
			http.Error(w, fmt.Sprintf("no cue list named %q", req.Name), http.StatusNotFound)
			return
		}
		if req.Queue != nil {
			cs.Queue(*req.Queue)
		}
		if req.Cue != nil && req.On != nil {
			cs.Enable(*req.Cue, *req.On)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /api/audio", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Running *bool `json:"running"`
			Restart bool  `json:"restart"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		pl := pn.Player()
		if pl == nil {
			http.Error(w, "the panel has no player to start and stop", http.StatusNotFound)
			return
		}
		if req.Restart {
			r, ok := pl.(restarter)
			if !ok {
				http.Error(w, "the player can't go back to the start", http.StatusNotFound)
				return
			}
			r.Restart()
		}
		if req.Running != nil {
			start := pl.AudioStop
			if *req.Running {
				start = pl.AudioStart
			}
			if err := start(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		writeJSON(w, map[string]bool{"running": pl.Running()})
	})
	mux.HandleFunc("/mount/{name}/", pn.serveMount)
	mux.HandleFunc("GET /theme.css", pn.serveTheme)
	mux.HandleFunc("GET /plugins/{plugin}/{file...}", pn.servePlugin)
	mux.HandleFunc("POST /api/message", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string          `json:"name"`
			Data json.RawMessage `json:"data"`
		}
		if par := pn.decode(w, r, &req, func() string { return req.Name }); par != nil {
			if par.custom == nil {
				http.Error(w, fmt.Sprintf("%q is not a control of a plugin", req.Name), http.StatusBadRequest)
				return
			}
			par.custom.receive(req.Data)
			writeJSON(w, map[string]bool{"ok": true})
		}
	})
	mux.HandleFunc("GET /api/events", pn.events)
	return mux
}

// decode reads a request naming a control and returns the control, or answers with an error
func (pn *Panel) decode(w http.ResponseWriter, r *http.Request, req any, name func() string) *Param {
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}
	par := pn.Param(name())
	if par == nil {
		http.Error(w, fmt.Sprintf("no control named %q", name()), http.StatusNotFound)
	}
	return par
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// description is the panel as the page sees it
type description struct {
	Title    string       `json:"title"`
	Presets  bool         `json:"presets"` // whether the page can save and load setups
	Player   bool         `json:"player"`  // whether the page can start and stop the sound
	Running  bool         `json:"running"` // whether the sound plays
	Restart  bool         `json:"restart"` // whether the page can set the sound back to its start
	Sections []section    `json:"sections"`
	Plugins  []pluginDesc `json:"plugins"`
	Themes   []string     `json:"themes"`
	Theme    string       `json:"theme"` // the panel's own
}

type section struct {
	Name     string    `json:"name"`
	Controls []control `json:"controls"`
}

type control struct {
	Kind    Kind            `json:"kind"`
	Name    string          `json:"name"`
	Min     float64         `json:"min"`
	Max     float64         `json:"max"`
	Curve   Curve           `json:"curve"`
	Step    float64         `json:"step"`
	Unit    string          `json:"unit,omitempty"`
	Label   string          `json:"label,omitempty"`
	Default float64         `json:"default"`
	Options []string        `json:"options,omitempty"`
	Value   float64         `json:"value"`
	CC      *Binding        `json:"cc,omitempty"`
	Grid    *gridDesc       `json:"grid,omitempty"`
	Cues    *cuesDesc       `json:"cues,omitempty"`
	Mount   *mountDesc      `json:"mount,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`  // of a custom control
	State   json.RawMessage `json:"state,omitempty"` // of a custom control
}

// cuesDesc is a cue list as the page sees it
type cuesDesc struct {
	Names   []string `json:"names"`
	Slots   int      `json:"slots,omitempty"`   // tiles named by position: the cues change
	Toggles bool     `json:"toggles,omitempty"` // a tap switches a cue on or off
	State   CueState `json:"state"`
}

// gridDesc is a grid as the page sees it: the cells in the order pattern, row, step, field
type gridDesc struct {
	Rows     []string  `json:"rows"`
	Steps    int       `json:"steps"`
	Patterns int       `json:"patterns"`
	Fields   []Field   `json:"fields"`
	Cells    []float64 `json:"cells"`
	Pattern  int       `json:"pattern"`
	Playing  int       `json:"playing"`
	Lengths  []int     `json:"lengths"`
}

func (pn *Panel) describe() description {
	pn.mu.Lock()
	sections := append([]string(nil), pn.sections...)
	pn.mu.Unlock()
	d := description{Title: pn.title, Presets: pn.presetsEnabled(), Plugins: []pluginDesc{}}
	for _, p := range pn.pluginList() {
		d.Plugins = append(d.Plugins, pluginDesc{Name: p.Name, Script: p.JS != "", Style: p.CSS != ""})
	}
	all, theme := pn.themes()
	d.Themes, d.Theme = themeNames(all), theme
	if pl := pn.Player(); pl != nil {
		d.Player, d.Running = true, pl.Running()
		_, d.Restart = pl.(restarter)
	}
	index := map[string]int{}
	for _, name := range sections {
		index[name] = len(d.Sections)
		d.Sections = append(d.Sections, section{Name: name})
	}
	for _, it := range pn.items() {
		if mt := it.mount; mt != nil {
			i := index[mt.Section]
			d.Sections[i].Controls = append(d.Sections[i].Controls, control{Kind: MountKind, Name: mt.Name, Mount: describeMount(mt.Mounted())})
			continue
		}
		if cs := it.cues; cs != nil {
			i := index[cs.Section]
			d.Sections[i].Controls = append(d.Sections[i].Controls, control{Kind: CuesKind, Name: cs.Name, Cues: &cuesDesc{Names: cs.Names, Slots: cs.Slots, Toggles: cs.Toggles, State: cs.State()}})
			continue
		}
		if g := it.grid; g != nil {
			i := index[g.Section]
			d.Sections[i].Controls = append(d.Sections[i].Controls, control{Kind: GridKind, Name: g.Name, Grid: &gridDesc{
				Rows: g.Rows, Steps: g.Steps, Patterns: g.Patterns, Fields: g.Fields,
				Cells: g.values(), Pattern: g.Pattern(), Playing: g.Playing(), Lengths: g.Lengths(),
			}})
			continue
		}
		par := it.param
		i := index[par.Section]
		c := control{
			Kind: par.Kind, Name: par.Name, Min: par.Min, Max: par.Max, Curve: par.Curve, Step: par.Step,
			Unit: par.Unit, Label: par.Label, Default: par.Default, Options: par.Options, Value: par.Value(),
		}
		if b, ok := par.Binding(); ok {
			c.CC = &b
		}
		if par.custom != nil {
			c.Data, c.State = par.custom.data, par.custom.stateNow()
		}
		d.Sections[i].Controls = append(d.Sections[i].Controls, c)
	}
	return d
}

// update is what the page gets every interval: the values that changed and the meters
type update struct {
	Values map[string]float64         `json:"values,omitempty"`
	Meters map[string][2]float64      `json:"meters,omitempty"`
	MIDI   *midiUpdate                `json:"midi,omitempty"` // when the bindings or learning changed
	Grids  map[string]*gridUpdate     `json:"grids,omitempty"`
	Cues   map[string]CueState        `json:"cues,omitempty"`
	Audio  *audioUpdate               `json:"audio,omitempty"` // when the sound starts or stops
	Mounts map[string]*mountDesc      `json:"mounts,omitempty"`
	States map[string]json.RawMessage `json:"states,omitempty"` // of custom controls, when they changed
}

type audioUpdate struct {
	Running bool `json:"running"`
}

// gridUpdate is what changed of a grid: its cells (all of them, when any changed), the patterns,
// the playheads
type gridUpdate struct {
	Cells     []float64 `json:"cells,omitempty"`
	Pattern   int       `json:"pattern"`
	Playing   int       `json:"playing"`
	Playheads []int     `json:"playheads"`
	Lengths   []int     `json:"lengths"`
}

// gridSent is what a page was sent of a grid last
type gridSent struct {
	version          int64
	pattern, playing int
	playheads        string
}

type midiUpdate struct {
	Bindings map[string]Binding `json:"bindings"`
	Learning string             `json:"learning"`
}

// events streams the values and the meters to a page while it is open
func (pn *Panel) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sent := map[string]float64{}
	midiSent := int64(-1)
	gridsSent := map[string]*gridSent{}
	cuesSent := map[string]string{}
	mountsSent := map[string]*Panel{}
	runningSent := -1
	statesSent := map[string]string{}
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()
	for {
		u := update{Values: map[string]float64{}, Meters: map[string][2]float64{}}
		pl := pn.Player()
		stopped := pl != nil && !pl.Running() // the meters stand still: show silence
		for _, par := range pn.Params() {
			switch par.Kind {
			case MeterKind:
				peak, rms := par.Levels()
				if stopped {
					peak, rms = -120, -120
				}
				u.Meters[par.Name] = [2]float64{peak, rms}
			case ButtonKind:
			default:
				if v, ok := sent[par.Name]; !ok || v != par.Value() {
					u.Values[par.Name] = par.Value()
					sent[par.Name] = par.Value()
				}
			}
		}
		for _, par := range pn.Params() {
			if par.custom == nil {
				continue
			}
			st := par.custom.stateNow()
			if st == nil || string(st) == statesSent[par.Name] {
				continue
			}
			statesSent[par.Name] = string(st)
			if u.States == nil {
				u.States = map[string]json.RawMessage{}
			}
			u.States[par.Name] = st
		}
		for _, g := range pn.Grids() {
			sent := gridsSent[g.Name]
			if sent == nil {
				sent = &gridSent{version: -1}
				gridsSent[g.Name] = sent
			}
			gu := &gridUpdate{Pattern: g.Pattern(), Playing: g.Playing(), Playheads: g.Playheads(), Lengths: g.Lengths()}
			changed := false
			if v := g.version.Load(); v != sent.version {
				sent.version = v
				gu.Cells = g.values()
				changed = true
			}
			if heads := fmt.Sprint(gu.Playheads, gu.Lengths); heads != sent.playheads || gu.Pattern != sent.pattern || gu.Playing != sent.playing {
				sent.playheads, sent.pattern, sent.playing = heads, gu.Pattern, gu.Playing
				changed = true
			}
			if changed {
				if u.Grids == nil {
					u.Grids = map[string]*gridUpdate{}
				}
				u.Grids[g.Name] = gu
			}
		}
		for _, mt := range pn.mountList() {
			child := mt.Mounted()
			if last, ok := mountsSent[mt.Name]; ok && last == child {
				continue
			}
			mountsSent[mt.Name] = child
			if u.Mounts == nil {
				u.Mounts = map[string]*mountDesc{}
			}
			u.Mounts[mt.Name] = describeMount(child)
		}
		for _, cs := range pn.cueLists() {
			st := cs.State()
			if key := fmt.Sprint(st); key != cuesSent[cs.Name] {
				cuesSent[cs.Name] = key
				if u.Cues == nil {
					u.Cues = map[string]CueState{}
				}
				u.Cues[cs.Name] = st
			}
		}
		if pl != nil {
			running := 0
			if pl.Running() {
				running = 1
			}
			if running != runningSent {
				runningSent = running
				u.Audio = &audioUpdate{Running: running == 1}
			}
		}
		if n := pn.bindings.Load(); n != midiSent {
			midiSent = n
			u.MIDI = &midiUpdate{Bindings: pn.Bindings()}
			if l := pn.Learning(); l != nil {
				u.MIDI.Learning = l.Name
			}
		}
		if len(u.Values) > 0 || len(u.Meters) > 0 || u.MIDI != nil || len(u.Grids) > 0 || len(u.Cues) > 0 || u.Audio != nil || len(u.Mounts) > 0 || len(u.States) > 0 {
			data, _ := json.Marshal(u)
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}
