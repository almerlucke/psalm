package main

import (
	"fmt"
	"math"
	"path"
	"strings"

	"github.com/almerlucke/psalm/synths/sampler"
)

// the percussion of the piece: hand-played one-shots (the world percussion and a cymbal), each strip a sampler with a few recordings of every kind of hit, each
// recording on a key of its own from 36. The scripts get the keys by kit and kind (shared.kits.hands.conga is a list of keys, played in turn), so the
// recordings are named here only
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
	{"shaker", []kind{
		{"egg", []string{worldPath + "EggShaker/fast_01.wav", worldPath + "EggShaker/fast_02.wav", worldPath + "EggShaker/fast_03.wav", worldPath + "EggShaker/fast_04.wav"}},
		{"maraca", []string{worldPath + "Maracas/01_01.wav", worldPath + "Maracas/01_02.wav", worldPath + "Maracas/01_03.wav", worldPath + "Maracas/01_04.wav"}},
	}},
	{"tambourine", []kind{
		{"hit", []string{worldPath + "Tambourine/01_01.wav", worldPath + "Tambourine/02_01.wav", worldPath + "Tambourine/03_01.wav", worldPath + "Tambourine/04_01.wav"}},
		{"jingle", []string{worldPath + "Tambourine/05.wav", worldPath + "Tambourine/06.wav", worldPath + "Tambourine/07.wav", worldPath + "Tambourine/08.wav"}},
	}},
	{"wood", []kind{
		{"clave", []string{worldPath + "Claves/01.wav", worldPath + "Claves/02.wav", worldPath + "Claves/03.wav", worldPath + "Claves/04.wav"}},
		{"castanet", []string{worldPath + "Castanets/01.wav", worldPath + "Castanets/02.wav", worldPath + "Castanets/03.wav", worldPath + "Castanets/04.wav", worldPath + "Castanets/05.wav", worldPath + "Castanets/06.wav"}},
	}},
	{"hands", []kind{
		{"bongo", []string{worldPath + "Bongos/1_01.wav", worldPath + "Bongos/1_02.wav", worldPath + "Bongos/1_03.wav", worldPath + "Bongos/1_04.wav"}},
		{"conga", []string{worldPath + "HighConga/v2_01_01.wav", worldPath + "Conga/v3_01_01.wav", worldPath + "HighConga/v3_01_02.wav", worldPath + "MutedConga/v2_02_01.wav"}},
		{"low", []string{worldPath + "LowConga/v3_01_01.wav", worldPath + "LowConga/v4_01_01.wav", worldPath + "Darbuka/doom_01_01.wav", worldPath + "Darbuka/doom_01_02.wav"}},
		{"cajon", []string{worldPath + "CajonFlamenco/101.wav", worldPath + "CajonFlamenco/102.wav", worldPath + "CajonFlamenco/103.wav"}},
		{"clap", []string{worldPath + "HandClap/01_02.wav", worldPath + "HandClap/01_03.wav", worldPath + "HandClap/01_05.wav"}},
	}},
	{"cymbal", []kind{
		{"swell", []string{"assets/sounds/various/cymbal-big2-83942.wav"}},
	}},
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
				lines = append(lines, fmt.Sprintf("%d %s whole name=%s vol=%.1f", key, f, strings.TrimSuffix(path.Base(f), path.Ext(f)), vol))
				key++
			}
		}
		return strings.Join(lines, "\n"), nil
	}
	return "", fmt.Errorf("no kit %q", name)
}

// rmsDB is the level of a sound, in dB, the channels together
func rmsDB(snd sampler.Sound) float64 {
	sum, n := 0.0, 0.0
	for c := range snd.NumChannels() {
		for _, v := range snd.Buffer(c, 0)[:snd.NumFrames()] {
			sum += float64(v) * float64(v)
			n++
		}
	}
	if n == 0 || sum == 0 {
		return -120
	}
	return 10 * math.Log10(sum/n)
}
