package panel

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

// MountKind is the kind of a mounted panel on the page
const MountKind Kind = "mount"

// Mounter gives the panel to show in a mount: it can be another one every time (the panel of the
// song a playlist plays), nil for none. It is called from the page's goroutines and the audio
// thread
type Mounter interface {
	Mounted() *Panel
}

// MountKeeper is a mounter that keeps the values of the panels it mounts in snapshots (and so in
// presets) too, also of the ones not mounted now: the controls of every song of a playlist. Keys
// are its own ("Groove/Tempo"); in a snapshot they come after the name of the mount
// ("Song/Groove/Tempo")
type MountKeeper interface {
	Mounter
	SnapshotMount() map[string]float64
	RestoreMount(values map[string]float64)
}

// Mount is a place on the panel showing another panel
type Mount struct {
	Name    string
	Section string
	m       Mounter
}

// Mount adds a place showing another panel, the one the mounter gives now: its controls, as they
// are on its own page, inside this one. When the mounter gives another panel the page shows that
// one. The control changes and notes of MIDI (MIDI) reach the panel mounted too, and MIDI learn
// on the page binds its controls. A playlist mounts the panel of the song playing
// (playlist.SongControls)
func (pn *Panel) Mount(name string, m Mounter) *Mount {
	mt := &Mount{Name: name, m: m}
	pn.mu.Lock()
	defer pn.mu.Unlock()
	if pn.taken(name) {
		panic(fmt.Sprintf("panel: a control named %q twice", name))
	}
	if pn.section == "" {
		pn.section = pn.title
		pn.sections = append(pn.sections, pn.section)
	}
	mt.Section = pn.section
	if pn.mounts == nil {
		pn.mounts = map[string]*Mount{}
	}
	pn.mounts[name] = mt
	pn.order = append(pn.order, item{mount: mt})
	return mt
}

// Mounted returns the panel the mount shows now, nil for none
func (mt *Mount) Mounted() *Panel { return mt.m.Mounted() }

// mountList returns the mounts in the order they were declared
func (pn *Panel) mountList() []*Mount {
	var ms []*Mount
	for _, it := range pn.items() {
		if it.mount != nil {
			ms = append(ms, it.mount)
		}
	}
	return ms
}

// serveMount serves the page and the API of a mounted panel under /mount/<name>/
func (pn *Panel) serveMount(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	pn.mu.Lock()
	mt := pn.mounts[name]
	pn.mu.Unlock()
	var child *Panel
	if mt != nil {
		child = mt.Mounted()
	}
	if child == nil {
		http.Error(w, fmt.Sprintf("nothing mounted at %q", name), http.StatusNotFound)
		return
	}
	http.StripPrefix("/mount/"+name, child.Handler()).ServeHTTP(w, r)
}

// mountDesc is a mount as the page sees it: the title of the panel mounted ("" for none) and its
// id (every panel has its own), so the page loads another panel only when it is another one
type mountDesc struct {
	Title string `json:"title"`
	ID    int64  `json:"id"`
}

// describeMount describes the panel mounted
func describeMount(child *Panel) *mountDesc {
	if child == nil {
		return &mountDesc{}
	}
	return &mountDesc{Title: child.title, ID: child.id}
}

// panelIDs numbers the panels
var panelIDs atomic.Int64
