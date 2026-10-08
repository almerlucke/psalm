package panel

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

//go:embed page.html
var page []byte

// updateInterval is how often the page gets the values and meters
const updateInterval = 50 * time.Millisecond

// Serve serves the panel as a web page on an address and blocks; run it on its own goroutine
// before playing the patch:
//
//	go ctl.Serve("localhost:8080") // this computer only
//	go ctl.Serve(":8080")          // also from a phone or tablet on the same network
//
// Every page open on the panel shows the same values: a fader moved on one moves on the others
func (pn *Panel) Serve(addr string) error {
	return http.ListenAndServe(addr, pn.Handler())
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
		writeJSON(w, pn.Snapshot())
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
	Title    string    `json:"title"`
	Sections []section `json:"sections"`
}

type section struct {
	Name     string    `json:"name"`
	Controls []control `json:"controls"`
}

type control struct {
	Kind    Kind     `json:"kind"`
	Name    string   `json:"name"`
	Min     float64  `json:"min"`
	Max     float64  `json:"max"`
	Curve   Curve    `json:"curve"`
	Step    float64  `json:"step"`
	Unit    string   `json:"unit,omitempty"`
	Default float64  `json:"default"`
	Options []string `json:"options,omitempty"`
	Value   float64  `json:"value"`
	CC      *Binding `json:"cc,omitempty"`
}

func (pn *Panel) describe() description {
	pn.mu.Lock()
	sections := append([]string(nil), pn.sections...)
	pn.mu.Unlock()
	d := description{Title: pn.title}
	index := map[string]int{}
	for _, name := range sections {
		index[name] = len(d.Sections)
		d.Sections = append(d.Sections, section{Name: name})
	}
	for _, par := range pn.Params() {
		i := index[par.Section]
		c := control{
			Kind: par.Kind, Name: par.Name, Min: par.Min, Max: par.Max, Curve: par.Curve, Step: par.Step,
			Unit: par.Unit, Default: par.Default, Options: par.Options, Value: par.Value(),
		}
		if ch, ctl, ok := par.CC(); ok {
			c.CC = &Binding{ch, ctl}
		}
		d.Sections[i].Controls = append(d.Sections[i].Controls, c)
	}
	return d
}

// update is what the page gets every interval: the values that changed and the meters
type update struct {
	Values map[string]float64    `json:"values,omitempty"`
	Meters map[string][2]float64 `json:"meters,omitempty"`
	MIDI   *midiUpdate           `json:"midi,omitempty"` // when the bindings or learning changed
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
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()
	for {
		u := update{Values: map[string]float64{}, Meters: map[string][2]float64{}}
		for _, par := range pn.Params() {
			switch par.Kind {
			case MeterKind:
				peak, rms := par.Levels()
				u.Meters[par.Name] = [2]float64{peak, rms}
			case ButtonKind:
			default:
				if v, ok := sent[par.Name]; !ok || v != par.Value() {
					u.Values[par.Name] = par.Value()
					sent[par.Name] = par.Value()
				}
			}
		}
		if n := pn.bindings.Load(); n != midiSent {
			midiSent = n
			u.MIDI = &midiUpdate{Bindings: pn.Bindings()}
			if l := pn.Learning(); l != nil {
				u.MIDI.Learning = l.Name
			}
		}
		if len(u.Values) > 0 || len(u.Meters) > 0 || u.MIDI != nil {
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
