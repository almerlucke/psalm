// clap lists the CLAP plugins on this computer and plays a chord on one, to a WAV file:
//
//	go run ./cmd/clap                                          list the plugins (the places searched are in the doc of modules/clap)
//	go run ./cmd/clap -plugin "Surge XT" -params               and the parameters of one
//	go run ./cmd/clap -plugin "Surge XT" -render chord.wav     a chord (-notes 60,64,67.02 -seconds 4); pitches may lie between the keys
//	go run ./cmd/clap -dir ~/plugins                           search another directory
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/clap"
)

func main() {
	dir := flag.String("dir", "", "search this directory instead of the usual ones")
	name := flag.String("plugin", "", "the plugin to use (its name or id)")
	showParams := flag.Bool("params", false, "list the parameters of the plugin")
	out := flag.String("render", "", "play a chord on the plugin and write it to this WAV file")
	notes := flag.String("notes", "60,64,67", "the pitches of the chord (MIDI, may be fractional)")
	buffer := flag.Int("buffer", 256, "the size of a buffer in samples")
	seconds := flag.Float64("seconds", 4, "how long the chord is held")
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
	if *showParams {
		for _, p := range pl.Describe().Params {
			fmt.Printf("%-40q %g .. %g  (%g)\n", p.Key, p.Min, p.Max, p.Value)
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
