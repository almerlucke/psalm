// clap lists the CLAP plugins on this computer and plays a chord on one, to a WAV file:
//
//	go run ./cmd/clap                                          list the plugins (the places searched are in the doc of modules/clap)
//	go run ./cmd/clap -plugin "Surge XT" -params               and the parameters of one
//	go run ./cmd/clap -plugin "Surge XT" -render chord.wav     a chord (-notes 60,64,67.02 -seconds 4); pitches may lie between the keys
//	go run ./cmd/clap -plugin "Surge XT" -preset pluck.txt -save editors-presets/clap   make a preset of a text (lines "a osc 1 type = Sine", "a amp eg attack = 5 ms")
//	go run ./cmd/clap -plugin "Surge XT" -presets              the plugin's own presets (Surge XT needs its data: see docs/clap.md); -load "Name" -render x.wav plays one
//	go run ./cmd/clap -dir ~/plugins                           search another directory
package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/clap"
)

func main() {
	dir := flag.String("dir", "", "search this directory instead of the usual ones")
	name := flag.String("plugin", "", "the plugin to use (its name or id)")
	showPresets := flag.Bool("presets", false, "list the presets the plugin has of its own")
	loadPreset := flag.String("load", "", "with -render: load this preset of the plugin's own first (its name, or the start of it)")
	showParams := flag.Bool("params", false, "list the parameters of the plugin")
	explore := flag.String("explore", "", "parameter keys (separated by a semicolon): what the plugin shows over their range")
	out := flag.String("render", "", "play a chord on the plugin and write it to this WAV file")
	notes := flag.String("notes", "60,64,67", "the pitches of the chord (MIDI, may be fractional)")
	buffer := flag.Int("buffer", 256, "the size of a buffer in samples")
	seconds := flag.Float64("seconds", 4, "how long the chord is held")
	preset := flag.String("preset", "", "text files of settings (key = what the plugin shows for it), separated by commas: applied before the chord, and saved with -save")
	save := flag.String("save", "", "with -preset: write each as a preset of the editor (a JSON file) in this folder")
	window := flag.Float64("window", 0, "show the window of the plugin (if it has one) for this many seconds")
	set := flag.String("set", "", "parameters to set before it plays: key=value, separated by a semicolon")
	flag.Parse()

	var dirs []string
	if *dir != "" {
		dirs = []string{*dir}
	}
	found := clap.Scan(dirs...)
	if *name == "" {
		for _, d := range found {
			fmt.Printf("%-28s %-12s %-14s %s\n", d.Name, d.Vendor, strings.Join(d.Features, ","), d.Path)
		}
		if len(found) == 0 {
			fmt.Println("no plugins in", clap.SearchPath())
		}
		return
	}
	var d *clap.Descriptor
	for i := range found {
		if found[i].Name == *name || found[i].ID == *name {
			d = &found[i]
			break
		}
	}
	if d == nil {
		log.Fatalf("no plugin %q", *name)
	}

	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: *buffer})
	pl, err := clap.New(*d)
	if err != nil {
		log.Fatal(err)
	}
	defer pl.Free()
	if *window > 0 {
		fmt.Println("the plugin has a window:", pl.HasWindow())
		if err := pl.ShowWindow(); err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Duration(*window * float64(time.Second)))
		fmt.Println("the window is open:", pl.WindowOpen())
		return
	}
	if *showPresets {
		for i, p := range pl.Presets() {
			fmt.Printf("%4d  %-16s %-36s %s\n", i, p.Category, p.Name, p.Creator)
		}
		fmt.Println(len(pl.Presets()), "presets")
	}
	if *loadPreset != "" {
		found := -1
		for i, p := range pl.Presets() {
			if p.Name == *loadPreset || (found < 0 && strings.HasPrefix(strings.ToLower(p.Name), strings.ToLower(*loadPreset))) {
				found = i
				if p.Name == *loadPreset {
					break
				}
			}
		}
		if found < 0 {
			log.Fatalf("no preset %q", *loadPreset)
		}
		if err := pl.LoadPreset(found); err != nil {
			log.Fatal(err)
		}
		fmt.Println("loaded", pl.Presets()[found].Name)
	}
	if *showParams {
		for _, p := range pl.Describe().Params {
			fmt.Printf("%-40q %g .. %g  (%g)\n", p.Key, p.Min, p.Max, p.Value)
		}
	}
	if *explore != "" { // what a parameter shows over its range
		for _, k := range strings.Split(*explore, ";") {
			k = strings.TrimSpace(k)
			fmt.Printf("%q:", k)
			last := ""
			for i := 0; i <= 200; i++ {
				v := float64(i) / 200
				if t := pl.Text(k, v); t != last {
					fmt.Printf("  %.3f=%s", v, t)
					last = t
				}
			}
			fmt.Println()
		}
	}
	for _, file := range strings.Split(*preset, ",") {
		if file = strings.TrimSpace(file); file == "" {
			continue
		}
		name, settings, err := readPreset(pl, file)
		if err != nil {
			log.Fatal(err)
		}
		if *save != "" {
			if err := os.MkdirAll(*save, 0o755); err != nil {
				log.Fatal(err)
			}
			data, _ := json.MarshalIndent(map[string]any{"settings": settings}, "", " ")
			if err := os.WriteFile(filepath.Join(*save, name+".json"), data, 0o644); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("saved %s (%d settings)\n", name, len(settings))
		}
		if *out != "" {
			for k, v := range settings {
				pl.ReceiveMessage(psalm.Message{k: v})
			}
		}
	}
	if *out == "" {
		return
	}
	for _, kv := range strings.Split(*set, ";") {
		if k, v, ok := strings.Cut(kv, "="); ok {
			f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err != nil {
				log.Fatal(err)
			}
			pl.ReceiveMessage(psalm.Message{strings.TrimSpace(k): f})
		}
	}
	cfg := psalm.CurrentEnv().Config()
	for i, s := range strings.Split(*notes, ",") {
		pitch, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			log.Fatal(err)
		}
		pl.ReceiveMessage(psalm.Message{"command": "trigger", "noteOn": fmt.Sprint("n", i), "amplitude": 0.7, "message": psalm.Message{"pitch": pitch}, "tick": psalm.CtrlTick{}})
	}
	var pcm []int16
	n := int(*seconds * cfg.SampleRate / float64(cfg.BufferSize))
	tail := int(2 * cfg.SampleRate / float64(cfg.BufferSize))
	for b := 0; b < n+tail; b++ {
		if b == n {
			for i := range strings.Split(*notes, ",") {
				pl.ReceiveMessage(psalm.Message{"command": "trigger", "noteOff": fmt.Sprint("n", i), "tick": psalm.CtrlTick{}})
			}
		}
		pl.ResetRun()
		pl.Synthesize()
		ch := pl.Output(0).Channels
		for i := range ch[0] {
			for c := 0; c < 2; c++ {
				pcm = append(pcm, int16(math.Max(-1, math.Min(1, float64(ch[c][i])))*32767))
			}
		}
	}
	if err := writeWAV(*out, pcm, int(cfg.SampleRate)); err != nil {
		log.Fatal(err)
	}
	fmt.Println("wrote", *out)
}

// readPreset reads a text of settings: a line "key = what the plugin shows" (the key may be the end of one), # starts a comment, "# name: Pluck" names the preset
// (else the file does)
func readPreset(pl *clap.Plugin, file string) (name string, settings map[string]float64, err error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", nil, err
	}
	name = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	settings = map[string]float64{}
	for n, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "# name:"); ok {
			name = strings.TrimSpace(rest)
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, text, ok := strings.Cut(line, "=")
		if !ok {
			return "", nil, fmt.Errorf("%s:%d: no =", file, n+1)
		}
		full, v, err := pl.Resolve(strings.TrimSpace(key), strings.TrimSpace(text))
		if err != nil {
			return "", nil, fmt.Errorf("%s:%d: %w", file, n+1, err)
		}
		settings[full] = v
	}
	return name, settings, nil
}

func writeWAV(path string, pcm []int16, rate int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data := uint32(len(pcm) * 2)
	for _, v := range []any{[]byte("RIFF"), 36 + data, []byte("WAVEfmt "), uint32(16), uint16(1), uint16(2), uint32(rate), uint32(rate * 4), uint16(4), uint16(16), []byte("data"), data, pcm} {
		if err := binary.Write(f, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	return nil
}
