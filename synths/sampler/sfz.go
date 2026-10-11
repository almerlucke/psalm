package sampler

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/almerlucke/sndfile"
)

// Loader loads the sound of a file
type Loader func(path string) (Sound, error)

// LoadFile loads a sound file as it is (wav, aiff, flac, ogg: what libsndfile reads)
func LoadFile(path string) (Sound, error) {
	return sndfile.NewSoundFile[float32](path)
}

// MipMaps returns a loader that also keeps band limited copies of every sound, for playing them
// up to depth times as fast without aliasing (costs depth times the memory)
func MipMaps(depth int) Loader {
	return func(path string) (Sound, error) {
		return sndfile.NewMipMapSoundFile[float32](path, depth)
	}
}

// LoadSFZ loads an SFZ instrument and its sounds. It knows the headers <control>, <global>,
// <master>, <group> and <region>, #define and #include, and the opcodes of the regions: the keys
// and velocities (lokey hikey key lovel hivel), the pitch (pitch_keycenter pitch_keytrack transpose
// tune), the level (volume amplitude pan amp_veltrack), where it plays (offset end), the loops
// (loop_mode loop_start loop_end loop_crossfade), release regions (trigger rt_decay), choke groups
// (group off_by off_mode), round robins (seq_length seq_position lorand hirand), the envelope
// (ampeg_delay ampeg_attack ampeg_hold ampeg_decay ampeg_sustain ampeg_release) and a filter
// (cutoff resonance fil_type fil_veltrack). The opcodes it doesn't know are counted in
// Instrument.Unsupported
func LoadSFZ(path string) (*Instrument, error) {
	return LoadSFZWith(path, LoadFile)
}

// MustSFZ is LoadSFZ that panics when the instrument doesn't load
func MustSFZ(path string) *Instrument {
	inst, err := LoadSFZ(path)
	if err != nil {
		panic(err)
	}
	return inst
}

// LoadSFZWith loads an SFZ instrument, its sounds with a loader
func LoadSFZWith(path string, load Loader) (*Instrument, error) {
	text, err := readSFZ(path, map[string]string{}, 0)
	if err != nil {
		return nil, err
	}
	return ParseSFZ(text, filepath.Dir(path), load)
}

// ParseSFZ reads an SFZ instrument from text; the paths of its sounds are relative to dir
func ParseSFZ(text string, dir string, load Loader) (*Instrument, error) {
	p := sfzParser{dir: dir, load: load, sounds: map[string]Sound{}, inst: &Instrument{Unsupported: map[string]int{}}}
	if err := p.parse(text); err != nil {
		return nil, err
	}
	if len(p.inst.Regions) == 0 {
		return nil, fmt.Errorf("sampler: the instrument has no regions")
	}
	return p.inst, nil
}

var (
	lineComment  = regexp.MustCompile(`//[^\n]*`)
	blockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	defineLine   = regexp.MustCompile(`(?m)^\s*#define\s+(\$\w+)\s+([^\r\n]*?)\s*$`)
	includeLine  = regexp.MustCompile(`(?m)^\s*#include\s+"([^"]+)"\s*$`)
	sfzToken     = regexp.MustCompile(`<(\w+)>|([A-Za-z0-9_]+)=`)
)

// readSFZ reads a file without its comments, with its includes in place and its defines replaced
func readSFZ(path string, defines map[string]string, depth int) (string, error) {
	if depth > 16 {
		return "", fmt.Errorf("sampler: %s: includes nest too deep", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	text := blockComment.ReplaceAllString(string(data), " ")
	text = lineComment.ReplaceAllString(text, "")
	var out strings.Builder
	for _, line := range strings.SplitAfter(text, "\n") {
		if m := defineLine.FindStringSubmatch(line); m != nil {
			defines[m[1]] = m[2]
			continue
		}
		if m := includeLine.FindStringSubmatch(line); m != nil {
			inc, err := readSFZ(filepath.Join(filepath.Dir(path), filepath.FromSlash(strings.ReplaceAll(m[1], `\`, "/"))), defines, depth+1)
			if err != nil {
				return "", err
			}
			out.WriteString(inc + "\n")
			continue
		}
		out.WriteString(replaceDefines(line, defines))
	}
	return out.String(), nil
}

// replaceDefines replaces $NAME by its value, the longest names first
func replaceDefines(line string, defines map[string]string) string {
	if !strings.Contains(line, "$") {
		return line
	}
	for {
		best := ""
		for name := range defines {
			if strings.Contains(line, name) && len(name) > len(best) {
				best = name
			}
		}
		if best == "" {
			return line
		}
		line = strings.ReplaceAll(line, best, defines[best])
	}
}

type opcodes = map[string]string

type sfzParser struct {
	dir    string
	load   Loader
	sounds map[string]Sound
	inst   *Instrument

	control, global, master, group opcodes
	region                         opcodes
	inRegion                       bool
}

func (p *sfzParser) parse(text string) error {
	p.control, p.global, p.master, p.group = opcodes{}, opcodes{}, opcodes{}, opcodes{}
	scope := p.control
	matches := sfzToken.FindAllStringSubmatchIndex(text, -1)
	for n, m := range matches {
		if !startsToken(text, m) {
			continue
		}
		if m[2] >= 0 { // a header
			if err := p.endRegion(); err != nil {
				return err
			}
			switch text[m[2]:m[3]] {
			case "control":
				scope = p.control
			case "global":
				p.global, p.master, p.group = opcodes{}, opcodes{}, opcodes{}
				scope = p.global
			case "master":
				p.master, p.group = opcodes{}, opcodes{}
				scope = p.master
			case "group":
				p.group = opcodes{}
				scope = p.group
			case "region":
				p.region, p.inRegion = opcodes{}, true
				scope = p.region
			default: // <curve>, <effect>, <midi>, <sample>: not played
				p.inst.Unsupported["<"+text[m[2]:m[3]]+">"]++
				scope = opcodes{}
			}
			continue
		}
		// the value runs to the next opcode or header
		end := len(text)
		for _, next := range matches[n+1:] {
			if startsToken(text, next) {
				end = next[0]
				break
			}
		}
		scope[text[m[4]:m[5]]] = strings.TrimSpace(text[m[1]:end])
	}
	return p.endRegion()
}

// startsToken says whether a match is a header or an opcode: an opcode starts a word, so "a=b"
// inside a value with spaces is part of the value
func startsToken(text string, m []int) bool {
	return m[2] >= 0 || m[0] == 0 || strings.ContainsRune(" \t\r\n>", rune(text[m[0]-1]))
}

// endRegion builds the region read, when there is one
func (p *sfzParser) endRegion() error {
	if !p.inRegion {
		return nil
	}
	p.inRegion = false
	ops := opcodes{}
	for _, scope := range []opcodes{p.global, p.master, p.group, p.region} {
		for k, v := range scope {
			ops[k] = v
		}
	}
	r, ok, err := p.build(ops)
	if err != nil || !ok {
		return err
	}
	if err := r.check(); err != nil {
		return err
	}
	p.inst.Regions = append(p.inst.Regions, r)
	return nil
}

// build makes a region of its opcodes; ok is false for a region that doesn't play
func (p *sfzParser) build(ops opcodes) (r Region, ok bool, err error) {
	name := ops["sample"]
	if name == "" {
		return r, false, nil
	}
	if strings.HasPrefix(name, "*") { // the built in sounds (*sine, *silence)
		p.inst.Unsupported["sample="+name]++
		return r, false, nil
	}
	path := filepath.Join(p.dir, filepath.FromSlash(strings.ReplaceAll(p.control["default_path"]+name, `\`, "/")))
	snd, found := p.sounds[path]
	if !found {
		if snd, err = p.load(path); err != nil {
			return r, false, fmt.Errorf("sampler: loading %s: %w", name, err)
		}
		p.sounds[path] = snd
	}
	r = NewRegion(snd)
	r.Name = name

	keyShift := 0
	if v, ok := p.control["note_offset"]; ok {
		keyShift += atoi(v)
	}
	if v, ok := p.control["octave_offset"]; ok {
		keyShift += 12 * atoi(v)
	}
	var errs []string
	key := func(op, v string) int {
		k, err := parseKey(v)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s=%s", op, v))
		}
		return k + keyShift
	}
	num := func(op, v string) float64 {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s=%s", op, v))
		}
		return f
	}
	frames := func(op, v string) int64 { return int64(num(op, v)) }

	// key sets the range and the root, the opcodes after it can change them
	if v, ok := ops["key"]; ok {
		k := key("key", v)
		r.LoKey, r.HiKey, r.Root = k, k, float64(k)
	}
	for op, v := range ops {
		switch op {
		case "sample", "key":
		case "lokey":
			r.LoKey = key(op, v)
		case "hikey":
			r.HiKey = key(op, v)
		case "lovel":
			r.LoVel = int(num(op, v))
		case "hivel":
			r.HiVel = int(num(op, v))
		case "pitch_keycenter":
			r.Root = float64(key(op, v))
		case "pitch_keytrack":
			r.KeyTrack = num(op, v)
		case "transpose":
			r.Transpose = num(op, v)
		case "tune", "pitch":
			r.Tune = num(op, v)
		case "volume", "gain":
			r.Volume = num(op, v)
		case "amplitude":
			r.Amp = num(op, v) / 100
		case "pan":
			r.Pan = num(op, v) / 100
		case "amp_veltrack":
			r.VelTrack = num(op, v) / 100
		case "offset":
			r.Offset = frames(op, v)
		case "end":
			if r.End = frames(op, v); r.End < 0 {
				return r, false, nil // end=-1: a region that doesn't play
			}
		case "loop_mode", "loopmode":
			switch v {
			case "no_loop":
				r.Loop = NoLoop
			case "one_shot":
				r.Loop = OneShot
			case "loop_continuous":
				r.Loop = LoopContinuous
			case "loop_sustain":
				r.Loop = LoopSustain
			default:
				errs = append(errs, op+"="+v)
			}
		case "loop_start", "loopstart":
			r.LoopStart = frames(op, v)
		case "loop_end", "loopend":
			r.LoopEnd = frames(op, v)
		case "loop_crossfade":
			r.LoopCrossfade = num(op, v)
		case "trigger":
			switch v {
			case "attack":
				r.Trigger = OnAttack
			case "release":
				r.Trigger = OnRelease
			default: // first, legato, release_key
				p.inst.Unsupported[op+"="+v]++
				return r, false, nil
			}
		case "rt_decay":
			r.RtDecay = num(op, v)
		case "group", "polyphony_group":
			r.Group = int(num(op, v))
		case "off_by", "offby":
			r.OffBy = int(num(op, v))
		case "off_mode":
			if v == "normal" {
				r.OffMode = OffNormal
			}
		case "seq_length":
			r.SeqLength = int(num(op, v))
		case "seq_position":
			r.SeqPosition = int(num(op, v))
		case "lorand":
			r.LoRand = num(op, v)
		case "hirand":
			r.HiRand = num(op, v)
		case "ampeg_delay":
			r.Env.Delay = num(op, v)
		case "ampeg_attack":
			r.Env.Attack = num(op, v)
		case "ampeg_hold":
			r.Env.Hold = num(op, v)
		case "ampeg_decay":
			r.Env.Decay = num(op, v)
		case "ampeg_sustain":
			r.Env.Sustain = num(op, v) / 100
		case "ampeg_release":
			r.Env.Release = num(op, v)
		case "cutoff":
			r.Cutoff = num(op, v)
		case "resonance":
			r.Resonance = num(op, v)
		case "fil_veltrack":
			r.FilVelTrack = num(op, v)
		case "fil_type", "filtype":
			switch {
			case strings.HasPrefix(v, "lpf"):
				r.Filter = LowPass
			case strings.HasPrefix(v, "hpf"):
				r.Filter = HighPass
			case strings.HasPrefix(v, "bpf"):
				r.Filter = BandPass
			default:
				p.inst.Unsupported[op+"="+v]++
			}
		default:
			p.inst.Unsupported[op]++
		}
	}
	if len(errs) > 0 {
		return r, false, fmt.Errorf("sampler: %s: bad %s", name, strings.Join(errs, ", "))
	}
	if r.SeqLength > 0 && r.SeqPosition == 0 {
		r.SeqPosition = 1
	}
	return r, true, nil
}

// parseKey reads a key as a number or a note name (c4 is 60, c#4 and db4 61)
func parseKey(v string) (int, error) {
	if n, err := strconv.Atoi(v); err == nil {
		return n, nil
	}
	s := strings.ToLower(v)
	if s == "" {
		return 0, fmt.Errorf("no key")
	}
	steps := map[byte]int{'c': 0, 'd': 2, 'e': 4, 'f': 5, 'g': 7, 'a': 9, 'b': 11}
	k, ok := steps[s[0]]
	if !ok {
		return 0, fmt.Errorf("bad key %q", v)
	}
	s = s[1:]
	if strings.HasPrefix(s, "#") {
		k, s = k+1, s[1:]
	} else if strings.HasPrefix(s, "b") && len(s) > 1 {
		k, s = k-1, s[1:]
	}
	oct, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("bad key %q", v)
	}
	return (oct+1)*12 + k, nil
}

func atoi(v string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(v))
	return n
}
