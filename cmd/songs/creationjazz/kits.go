package main

import (
	"encoding/json"
	"fmt"
	"math"
	"path"
	"strings"

	"github.com/almerlucke/psalm/synths/sampler"
)

// the percussion of the piece: hand-played one-shots on a sampler, a few recordings of every kind of hit, each on a key of its own from 36. The scripts get the keys
// by kit and kind (shared.kits.congas.low is a list of keys, played in turn), so the recordings are named here only
const (
	worldPath   = "assets/sounds/world_percussion/"
	kitFirstKey = 36
	kitLevel    = -18.0 // the level (dB, rms of the whole recording) every recording is brought to, the mixer sets the level of the strip
)

type kind struct {
	name  string
	files []string
}

type kit struct {
	name  string
	kinds []kind
}

var kits = []kit{
	{"congas", []kind{
		{"low", []string{worldPath + "LowConga/v3_01_01.wav", worldPath + "LowConga/v3_02_01.wav", worldPath + "LowConga/v4_01_01.wav"}},
		{"open", []string{worldPath + "Conga/v2_01_01.wav", worldPath + "Conga/v2_02_01.wav", worldPath + "Conga/v3_01_01.wav"}},
		{"high", []string{worldPath + "HighConga/v2_01_01.wav", worldPath + "HighConga/v2_02_01.wav", worldPath + "HighConga/v3_01_01.wav"}},
		{"muted", []string{worldPath + "MutedConga/v2_02_01.wav", worldPath + "MutedConga/v2_02_02.wav", worldPath + "MutedLowConga/v3_01_01.wav"}},
	}},
	{"bongos", []kind{
		{"high", []string{worldPath + "Bongos/1_01.wav", worldPath + "Bongos/1_03.wav", worldPath + "Bongos/1_05.wav", worldPath + "Bongos/1_07.wav"}},
		{"low", []string{worldPath + "Bongos/2_01.wav", worldPath + "Bongos/2_03.wav", worldPath + "Bongos/2_05.wav", worldPath + "Bongos/2_07.wav"}},
	}},
	{"claves", []kind{
		{"clave", []string{worldPath + "Claves/01.wav", worldPath + "Claves/02.wav", worldPath + "Claves/03.wav", worldPath + "Claves/04.wav"}},
		{"castanet", []string{worldPath + "Castanets/01.wav", worldPath + "Castanets/02.wav", worldPath + "Castanets/03.wav"}},
	}},
	{"shakers", []kind{
		{"maraca", []string{worldPath + "Maracas/01_01.wav", worldPath + "Maracas/01_02.wav", worldPath + "Maracas/01_03.wav", worldPath + "Maracas/01_04.wav"}},
		{"egg", []string{worldPath + "EggShaker/fast_01.wav", worldPath + "EggShaker/fast_02.wav", worldPath + "EggShaker/fast_03.wav", worldPath + "EggShaker/fast_04.wav"}},
		{"tambourine", []string{worldPath + "Tambourine/01_01.wav", worldPath + "Tambourine/02_01.wav", worldPath + "Tambourine/03_01.wav"}},
		{"swell", []string{"assets/sounds/various/cymbal-big2-83942.wav"}},
	}},
	{"darbuka", []kind{
		{"doum", []string{worldPath + "Darbuka/doom_01_01.wav", worldPath + "Darbuka/doom_01_03.wav", worldPath + "Darbuka/doom_01_05.wav"}},
		{"tak", []string{worldPath + "Darbuka/tak_02_01.wav", worldPath + "Darbuka/tak_02_03.wav", worldPath + "Darbuka/tak_02_05.wav"}},
		{"pa", []string{worldPath + "Darbuka/pa_03_01.wav", worldPath + "Darbuka/pa_03_03.wav"}},
	}},
}

// the recordings of creation are played as percussion too: long recordings cut at their hits (a piece on each key from 36), played on the grid by scripts over
// euclidean rhythms, a markov chain walking the pieces (the calls of the beasts, the gulls, the birds of the sea, the laughter of the children)
type cut struct {
	name, path string
	rise, gap  float64 // how a hit is found (sampler hits)
	from, to   float64 // the seconds of the recording to use, 0 0 for all
	vol        float64 // dB
	lowest     float64 // the pieces quieter than this (dB) are left out
}

var cuts = []cut{
	{"animals", animalsPath, 9, 0.4, 0, 70, 8, -48},
	{"sea and birds", seaBirdsPath, 6, 0.4, 0, 0, 8, -50},
	{"gulls", gullsPath, 9, 0.4, 0, 70, 8, -48},
	{"children", childrenPath, 6, 0.2, 0, 0, 6, -50},
}

func cutByName(name string) (cut, bool) {
	for _, c := range cuts {
		if c.name == name {
			return c, true
		}
	}
	return cut{}, false
}

// piece is a piece of a recording: the key, how long it lasts (seconds) and how loud it is (dB)
type piece struct {
	Key int     `json:"key"`
	Sec float64 `json:"sec"`
	DB  float64 `json:"db"`
}

// pieces cuts a recording as the sampler does and says of each piece the key, the seconds and the level
func (c cut) pieces() ([]piece, error) {
	snd, err := sampler.LoadFile(c.path)
	if err != nil {
		return nil, err
	}
	sr := snd.SampleRate()
	cuts := sampler.Window(sampler.OnsetCuts(snd, c.rise, c.gap), sr, c.from, c.to, snd.NumFrames())
	var out []piece
	for i := 0; i+1 < len(cuts) && kitFirstKey+i <= 127; i++ {
		out = append(out, piece{Key: kitFirstKey + i, Sec: math.Round(float64(cuts[i+1]-cuts[i])/sr*100) / 100, DB: math.Round(rmsRange(snd, cuts[i], cuts[i+1]))})
	}
	return out, nil
}

// piecesJS gives the scripts the pieces of the cut recordings (shared.pieces.animals is a list of {key, sec, db}, without the pieces too quiet to use)
func piecesJS() (string, error) {
	all := map[string][]piece{}
	for _, c := range cuts {
		ps, err := c.pieces()
		if err != nil {
			return "", err
		}
		for _, p := range ps {
			if p.DB > c.lowest {
				all[c.name] = append(all[c.name], p)
			}
		}
	}
	b, err := json.Marshal(all)
	return "shared.pieces = " + string(b) + "\n", err
}

// kitKeys is what the scripts know of the kits: the keys of every kind
func kitKeys() map[string]map[string][]int {
	out := map[string]map[string][]int{}
	for _, k := range kits {
		out[k.name] = map[string][]int{}
		key := kitFirstKey
		for _, kd := range k.kinds {
			for range kd.files {
				out[k.name][kd.name] = append(out[k.name][kd.name], key)
				key++
			}
		}
	}
	return out
}

// kitLines is the sounds of the sampler of a kit: a line for every recording on its key, brought to the same level
func kitLines(name string) (string, error) {
	if c, ok := cutByName(name); ok {
		return fmt.Sprintf("%d %q hits %g %g %g %g name=%s vol=%g", kitFirstKey, c.path, c.rise, c.gap, c.from, c.to, strings.ReplaceAll(c.name, " ", "-"), c.vol), nil
	}
	for _, k := range kits {
		if k.name != name {
			continue
		}
		var lines []string
		key := kitFirstKey
		for _, kd := range k.kinds {
			for _, f := range kd.files {
				snd, err := sampler.LoadFile(f)
				if err != nil {
					return "", err
				}
				vol := max(-20, min(20, kitLevel-rmsDB(snd)))
				lines = append(lines, fmt.Sprintf("%d %q whole name=%s vol=%.1f", key, f, strings.ReplaceAll(strings.TrimSuffix(path.Base(f), path.Ext(f)), " ", "-"), vol))
				key++
			}
		}
		return strings.Join(lines, "\n"), nil
	}
	return "", fmt.Errorf("no kit %q", name)
}

// rmsDB is the level of a sound, in dB, the channels together
func rmsDB(snd sampler.Sound) float64 { return rmsRange(snd, 0, snd.NumFrames()) }

// rmsRange is the level of a stretch of a sound (frames from to)
func rmsRange(snd sampler.Sound, from, to int64) float64 {
	sum, n := 0.0, 0.0
	for c := range snd.NumChannels() {
		for _, v := range snd.Buffer(c, 0)[from:to] {
			sum += float64(v) * float64(v)
			n++
		}
	}
	if n == 0 || sum == 0 {
		return -120
	}
	return 10 * math.Log10(sum/n)
}
