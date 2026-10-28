package main

import (
	"encoding/json"
	"fmt"
	"math"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/almerlucke/psalm/synths/sampler"
)

// the percussion of the piece: hand-played recordings. The eggs of the shaker and the swell of a cymbal are one-shots, each on a key of its own from 36; the low hand
// drums are a long recording cut at its hits, a piece on each key from 36, walked by the script (a markov chain) on a euclidean cycle that runs against the bar. The
// glass is struck glass, a recording for each pitch, played at the pitches of the chord
const (
	worldPath   = "assets/sounds/world_percussion/"
	kitFirstKey = 36
	kitLevel    = -18.0 // the level (dB, rms of the whole recording) every one-shot is brought to, the mixer sets the level of the strip
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
	{"shaker", []kind{
		{"egg", []string{worldPath + "EggShaker/fast_01.wav", worldPath + "EggShaker/fast_02.wav", worldPath + "EggShaker/fast_03.wav", worldPath + "EggShaker/fast_04.wav", worldPath + "EggShaker/fast_05.wav"}},
	}},
	{"cymbal", []kind{
		{"swell", []string{"assets/sounds/various/cymbal-big2-83942.wav"}},
	}},
}

type cut struct {
	name, path string
	rise, gap  float64 // how a hit is found (sampler hits)
	from, to   float64 // the seconds of the recording to use, 0 0 for all
	vol        float64 // dB
	lowest     float64 // the pieces quieter than this (dB) are left out
}

var cuts = []cut{
	{"hands", "assets/sounds/various/66-bpm-uniq-percussions-264333.wav", 9, 0.08, 0, 24, 10, -40},
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

// piecesJS gives the scripts the pieces of the cut recordings (shared.pieces.hands is a list of {key, sec, db}, without the pieces too quiet to use)
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
		return fmt.Sprintf("%d %q hits %g %g %g %g name=%s vol=%g", kitFirstKey, c.path, c.rise, c.gap, c.from, c.to, c.name, c.vol), nil
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
				vol := math.Max(-20, math.Min(20, kitLevel-rmsDB(snd)))
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

var pitchName = regexp.MustCompile(`^([A-G])(#?)(\d)_`)

// pitched makes a line of the sampler for every pitch of a folder of recordings named by their note (C#5_03.wav): one recording for each pitch, on its key and the
// keys above it up to the next recording (a little higher)
func pitched(dir, name string) (string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.wav"))
	if err != nil {
		return "", err
	}
	slices.Sort(files)
	semis := map[string]int{"C": 0, "D": 2, "E": 4, "F": 5, "G": 7, "A": 9, "B": 11}
	byPitch := map[int]string{}
	var pitches []int
	for _, f := range files {
		m := pitchName.FindStringSubmatch(filepath.Base(f))
		if m == nil {
			continue
		}
		oct, _ := strconv.Atoi(m[3])
		p := 12*(oct+1) + semis[m[1]]
		if m[2] == "#" {
			p++
		}
		if _, ok := byPitch[p]; ok || p > 125 {
			continue
		}
		byPitch[p] = f
		pitches = append(pitches, p)
	}
	if len(pitches) == 0 {
		return "", fmt.Errorf("%s: no recordings named by their note", dir)
	}
	slices.Sort(pitches)
	var lines []string
	for i, p := range pitches {
		hi := p + 2
		if i+1 < len(pitches) {
			hi = pitches[i+1] - 1
		}
		lines = append(lines, fmt.Sprintf("%d %q whole %d %d name=%s", p, byPitch[p], hi, p, name))
	}
	return strings.Join(lines, "\n"), nil
}
