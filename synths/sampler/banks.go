package sampler

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// ParseBanks makes an instrument from a text of one line for each sound, the way a patch describes the sounds of a sampler:
//
//	# the first key, the file, how to cut it, and anything of: vol=dB pan=-1..1 name=word
//	36 assets/sounds/various/tabla_loop_4bars_90bpm-86005.wav hits 9 0.08      44 pieces on the keys from 36
//	84 assets/sounds/chants/Johannes.Hymnus.wav change 1.1 0.25 vol=-3        pieces of a sung line, cut where it changes
//	36 assets/sounds/goedertierenheid.wav verses 26 20                         26 verses and their refrains, two pieces each, found by the pauses
//	0  assets/sounds/various/wooden-tubes-32138.wav even 16                    sixteen equal pieces
//	24 assets/sounds/loofdeheer.wav notes 0.5 1.9 60 2.6 4.9 56.1             the sung notes: from, to (seconds) and the pitch of each, on keys five apart
//	60 assets/sounds/various/birdfish-happy-loop-6199.wav whole 72 66          the whole sound on keys 60 to 72, in tune at 66
//	0  assets/Virtual-Playing-Orchestra3/Strings/harp-sustain.sfz sfz          an SFZ instrument as it is
//
// The modes are hits (rise dB, least gap seconds: OnsetCuts), change (sensitivity, least gap seconds: ChangeCuts), pauses (depth dB, least pause seconds, longest piece seconds: PauseCuts), verses (the number of verses, depth dB: VerseCuts, two pieces for each: the verse and its refrain), even (the number of
// pieces: EvenCuts), notes (from, to and pitch of each note: Notes, five keys apart) and whole (the highest key, the key it is in tune at; default the first key for both, so it plays on one key). Pieces
// are made by Slices, one key each from the first key up. Empty lines and lines that begin with # are left out. load reads the sounds. The
// second result describes what every line made, to show where the keys are.
func ParseBanks(text string, load Loader) (*Instrument, []string, error) {
	var regions []Region
	var report []string
	for n, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fail := func(format string, a ...any) error {
			return fmt.Errorf("line %d: %s", n+1, fmt.Sprintf(format, a...))
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			return nil, nil, fail("a key, a file and a mode: 36 loop.wav hits 9 0.08")
		}
		first, err := strconv.Atoi(fields[0])
		if err != nil || first < 0 || first > 127 {
			return nil, nil, fail("%q is not a key from 0 to 127", fields[0])
		}
		path, mode := fields[1], fields[2]
		var args []float64
		vol, pan, name := 0.0, 0.0, strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		for _, f := range fields[3:] {
			key, val, ok := strings.Cut(f, "=")
			if !ok {
				x, err := strconv.ParseFloat(f, 64)
				if err != nil {
					return nil, nil, fail("%q is not a number", f)
				}
				args = append(args, x)
				continue
			}
			switch key {
			case "name":
				name = val
			case "vol", "pan":
				x, err := strconv.ParseFloat(val, 64)
				if err != nil {
					return nil, nil, fail("%q is not a number", val)
				}
				if key == "vol" {
					vol = x
				} else {
					pan = x
				}
			default:
				return nil, nil, fail("%q: vol=, pan= or name=", key)
			}
		}
		arg := func(i int, def float64) float64 {
			if i < len(args) {
				return args[i]
			}
			return def
		}
		var made []Region
		if mode == "sfz" {
			inst, err := LoadSFZWith(path, load)
			if err != nil {
				return nil, nil, fail("%v", err)
			}
			made = inst.Regions
		} else {
			snd, err := load(path)
			if err != nil {
				return nil, nil, fail("%v", err)
			}
			switch mode {
			case "hits":
				made = Slices(snd, first, OnsetCuts(snd, arg(0, 9), arg(1, 0.08)))
			case "change":
				made = Slices(snd, first, ChangeCuts(snd, arg(0, 1.1), arg(1, 0.25)))
			case "pauses":
				made = Slices(snd, first, PauseCuts(snd, arg(0, 30), arg(1, 0.3), arg(2, 0)))
			case "verses":
				made = Slices(snd, first, VerseCuts(snd, int(arg(0, 1)), arg(1, 20)))
			case "even":
				made = Slices(snd, first, EvenCuts(snd, int(arg(0, 8))))
			case "notes": // from, to and pitch of every note, in seconds and MIDI pitch
				var notes [][3]float64
				for i := 0; i+2 < len(args); i += 3 {
					notes = append(notes, [3]float64{args[i], args[i+1], args[i+2]})
				}
				made = Notes(snd, first, 5, notes)
			case "whole":
				hi := min(127, int(arg(0, float64(first))))
				made = []Region{NewRegion(snd).Keys(first, max(first, hi), arg(1, float64(first)))}
				made[0].Loop = OneShot
			default:
				return nil, nil, fail("%q: hits, change, pauses, verses, even, notes, whole or sfz", mode)
			}
			for i := range made {
				made[i].Name = name
			}
		}
		for i := range made {
			if vol != 0 {
				made[i].Volume += vol
			}
			if pan != 0 {
				made[i].Pan = pan
			}
		}
		regions = append(regions, made...)
		if len(made) > 0 {
			lo, hi := made[0].LoKey, made[0].HiKey
			for _, r := range made {
				lo, hi = min(lo, r.LoKey), max(hi, r.HiKey)
			}
			report = append(report, fmt.Sprintf("%s: %d on keys %d-%d", name, len(made), lo, hi))
		}
	}
	inst, err := NewInstrument(regions...)
	if err != nil {
		return nil, nil, err
	}
	return inst, report, nil
}
