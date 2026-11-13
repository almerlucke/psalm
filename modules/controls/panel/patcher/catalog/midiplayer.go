package catalog

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/keyboard"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/params"
)

// midiControls is how many controller outputs a MIDI player has
const midiControls = 6

// defaultControllers are the controllers the outputs start at: the mod wheel, the expression, the volume, the pan, the cutoff (brightness) and the
// resonance
var defaultControllers = [midiControls]int{1, 11, 7, 10, 74, 71}

// playerSource is the input of a MIDI player: the player of the file now loaded, which can be swapped for another while it plays (the notes of the one
// before are let go). It keeps the settings, so a file loaded after them (a patch sets its settings before its texts) plays as they say
type playerSource struct {
	mu     sync.Mutex
	cur    *midi.Player
	old    *midi.Player // let go of in the next buffer
	play   bool
	loop   bool
	speed  float64
	follow func() float64 // the tempo followed, nil: the file's
}

func (s *playerSource) player() *midi.Player {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cur
}

// set changes the settings, on the player too
func (s *playerSource) set(f func(s *playerSource)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	was := s.play
	f(s)
	s.apply(s.play != was)
}

// apply puts the settings on the player (the lock held), the transport too when it is to follow
func (s *playerSource) apply(transport bool) {
	p := s.cur
	if p == nil {
		return
	}
	p.SetLoop(s.loop)
	p.SetSpeed(s.speed)
	p.FollowTempo(s.follow)
	switch { // the transport only where it was moved: a file that played to its end isn't started again by a change of the speed
	case !transport:
	case s.play:
		p.Play()
	default:
		p.Stop()
	}
}

// swap makes p the player
func (s *playerSource) swap(p *midi.Player) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cur != nil {
		s.cur.Stop()
		s.old = s.cur
	}
	s.cur = p
	s.apply(true)
}

func (s *playerSource) Read(events []midi.Event, frames int, sampleRate float64) []midi.Event {
	s.mu.Lock()
	old, cur := s.old, s.cur
	s.old = nil
	s.mu.Unlock()
	if old != nil {
		events = old.Read(events, frames, sampleRate)
	}
	if cur != nil {
		events = cur.Read(events, frames, sampleRate)
	}
	return events
}

// midiPlayer plays the notes of a MIDI file to the synths it is cabled to, and its controllers, the pitch bend and the pressure as modulation
type midiPlayer struct {
	*keyboard.Keyboard                        // all the channels
	chans              [16]*keyboard.Keyboard // a keyboard of each channel, reading the same file
	plays              [17][]psalm.Receiver   // what each notes output plays now
	src                *playerSource
	syncSong           atomic.Bool   // the song of the patch moves the file (read on the audio thread)
	lift               atomic.Uint64 // the velocity floor (a float64): 0 plays the velocities of the file, 1 every note at full
	modCh              atomic.Int32  // the channel the mod outputs follow (0: any)
	armed              bool          // the file is at the song and starts next buffer
	lastBuf            int64         // the buffer the song was last looked at

	// written on the audio thread, read by the mod outputs there
	cc       [128]float64
	bend     float64
	pressure float64
	which    [midiControls]atomic.Int32 // the controller of each output
}

func newMidiPlayer() *midiPlayer {
	m := &midiPlayer{src: &playerSource{speed: 1}, bend: 0.5, lastBuf: -1}
	for i, c := range defaultControllers {
		m.which[i].Store(int32(c))
	}
	ins := midi.Tee(m.src, 17)
	mk := func(in midi.Input, channel int) *keyboard.Keyboard {
		return keyboard.New(in, keyboard.Config{Channel: channel, VelocityMap: func(v float64) float64 {
			l := math.Float64frombits(m.lift.Load())
			return l + (1-l)*v
		}, Dynamics: 1, Controls: map[int]string{midi.CCModWheel: "modWheel", midi.CCSustain: "sustain"},
			HoldNotes: true, PitchBend: "pitchBend", Pressure: "pressure", PolyPressure: "pressure"})
	}
	m.Keyboard = mk(ins[0], 0)
	for i := range m.chans {
		m.chans[i] = mk(ins[1+i], i+1)
	}
	m.OnEvent(func(e midi.Event, _ psalm.CtrlTick) { // the mod outputs: of one channel (the setting), or of all
		msg := e.Message
		if ch := int(m.modCh.Load()); ch > 0 && msg.IsChannel() && msg.Channel() != ch {
			return
		}
		switch msg.Kind() {
		case midi.ControlChange:
			m.cc[msg.Controller()&127] = float64(msg.Value()) / 127
		case midi.PitchBend:
			m.bend = (msg.BendAmount() + 1) / 2
		case midi.ChannelPressure:
			m.pressure = float64(msg.Pressure()) / 127
		}
	})
	m.SetSelf(m)
	return m
}

// ReceiveMessage takes the settings: play, loop, speed, follow (the tempo of the patch); transpose
func (m *midiPlayer) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("modChannel", func(f float64) { m.modCh.Store(int32(max(0, min(f, 16)))) })
	msg.IfFltVal("seek", func(f float64) { // not a setting of the editor: a program moves the file (a render of a part of the piece)
		if p := m.src.player(); p != nil {
			p.Seek(max(0, f))
		}
	})
	msg.IfFltVal("lift", func(f float64) { m.lift.Store(math.Float64bits(max(0, min(f, 1)))) })
	msg.IfFltVal("loop", func(f float64) { m.src.set(func(s *playerSource) { s.loop = f > 0.5 }) })
	msg.IfFltVal("speed", func(f float64) { m.src.set(func(s *playerSource) { s.speed = f }) })
	msg.IfFltVal("follow", func(f float64) {
		m.src.set(func(s *playerSource) {
			s.follow = nil
			if f > 0.5 {
				s.follow = m.BPM
			}
		})
	})
	msg.IfFltVal("sync", func(f float64) {
		m.syncSong.Store(f > 0.5)
		if f > 0.5 { // in step with the song it keeps the tempo of the patch
			m.src.set(func(s *playerSource) { s.follow = m.BPM; s.play = false })
		}
	})
	msg.IfFltVal("play", func(f float64) { m.src.set(func(s *playerSource) { s.play = f > 0.5 }) })
	msg.IfIntVal("transpose", func(v int) {
		m.Keyboard.ReceiveMessage(psalm.Message{"transpose": v})
		for _, k := range m.chans {
			k.ReceiveMessage(psalm.Message{"transpose": v})
		}
	})
}

// CtrlTick follows the song when it is to (once a buffer), then reads the file
func (m *midiPlayer) CtrlTick(tick psalm.CtrlTick) {
	if m.syncSong.Load() {
		if b := tick.SampleTimestamp - int64(tick.SampleOffset); b != m.lastBuf {
			m.lastBuf = b
			m.followSong(tick)
		}
	}
	m.Keyboard.CtrlTick(tick)
}

// followSong puts the file where the song is: the bar of the file is the bar of the form (the sections in the order of the file), so a section jumped to
// plays its part of the file. It plays while the song does and stops where it ends
func (m *midiPlayer) followSong(tick psalm.CtrlTick) {
	p := m.src.player()
	if p == nil {
		return
	}
	song := patcher.SongOf(m.Env())
	pos, ok := song.At(tick, m.Config().PPQ)
	if !ok || pos.Ended {
		if p.Playing() {
			p.Stop()
		}
		return
	}
	form := song.Transport().Form()
	start := float64(pos.Section.StartBar) * form.BeatsPerBar
	in := pos.Progress * float64(pos.Section.Bars) * form.BeatsPerBar
	want := start + in
	// a seek just after a start of a section goes to the start: the notes that fall on it, which the song is already a buffer past, sound
	seekTo := want
	if in < 0.1 {
		seekTo = start
	}
	switch {
	case !p.Playing(): // it starts a buffer after the song: the scripts have tuned the synths by then (the first notes sound with their settings)
		p.SeekBeat(seekTo)
		if m.armed {
			p.Play()
		}
		m.armed = !m.armed
	case math.Abs(p.Beat()-want) > 0.06: // a jump of the song, or a loop
		p.SeekBeat(seekTo)
	}
}

// Receive takes a message
func (m *midiPlayer) Receive(v any, _ int) {
	if msg, ok := psalm.MsgVal(v); ok {
		m.ReceiveMessage(msg)
	}
}

// out is what a mod output is: 0 the pitch bend, 1 the pressure, then the controllers
func (m *midiPlayer) out(i int) func() float64 {
	switch i {
	case 0:
		return func() float64 { return m.bend }
	case 1:
		return func() float64 { return m.pressure }
	}
	k := i - 2
	return func() float64 { return m.cc[m.which[k].Load()&127] }
}

// parseControllers reads the numbers of the controllers of the outputs (0-127, up to six; the ones missing stay as they were)
func parseControllers(text string) ([]int, error) {
	var out []int
	for _, f := range strings.FieldsFunc(text, func(r rune) bool { return r == ' ' || r == ',' || r == '\t' || r == '\n' }) {
		n, err := strconv.Atoi(f)
		if err != nil || n < 0 || n > 127 {
			return nil, fmt.Errorf("%q: a controller from 0 to 127", f)
		}
		out = append(out, n)
	}
	if len(out) > midiControls {
		return nil, fmt.Errorf("at most %d controllers", midiControls)
	}
	return out, nil
}

// midiPlayerKind plays a MIDI file: its notes go to the synths it is cabled to (like a keyboard's), the controllers of the file, its pitch bend and its pressure
// are mod outputs
func midiPlayerKind(folder string) patcher.Kind {
	if folder == "" {
		folder = "assets/midi"
	}
	outs := []patcher.Port{{Name: "all", Type: patcher.Notes}}
	for ch := range 16 {
		outs = append(outs, patcher.Port{Name: strconv.Itoa(ch + 1), Type: patcher.Notes, Index: 1 + ch})
	}
	outs = append(outs, patcher.Port{Name: "bend", Type: patcher.Mod, Index: 0}, patcher.Port{Name: "pressure", Type: patcher.Mod, Index: 1})
	for i := range midiControls {
		outs = append(outs, patcher.Port{Name: "ctrl " + strconv.Itoa(i+1), Type: patcher.Mod, Index: 2 + i})
	}
	const fileHelp = "- pick a file of the folder, or type the path of a .mid file (`assets/midi/jig-1.mid`); another swaps it while it plays"
	const ctrlHelp = "- the controller numbers of the outputs ctrl 1 to ctrl 6, `1 11 7 10 74 71`: the mod wheel, expression, volume, pan, brightness, resonance"
	return patcher.Kind{Name: "midiplayer", Group: "Sources", Outs: outs,
		Make: func(string) (*patcher.Made, error) {
			m := newMidiPlayer()
			var (
				mu     sync.Mutex
				status = "no file"
				file   string
			)
			load := func(path string) error {
				path = strings.TrimSpace(path)
				if path == "" {
					return nil
				}
				f, err := midi.ReadFile(path)
				if err != nil {
					mu.Lock()
					status = "error: " + err.Error()
					mu.Unlock()
					return err
				}
				pl := midi.NewPlayer(f)
				m.src.swap(pl)
				mu.Lock()
				file = path
				status = fmt.Sprintf("%s: %d tracks, %d:%02d", path, len(f.Tracks), int(pl.Duration())/60, int(math.Round(pl.Duration()))%60)
				mu.Unlock()
				return nil
			}
			return &patcher.Made{Module: m,
				Texts: []patcher.Text{
					{Key: "file", Label: "file", Help: fileHelp, Choose: midiFiles(folder)},
					{Key: "controllers", Label: "controllers", Value: "1 11 7 10 74 71", Help: ctrlHelp},
				},
				Upload: func(key, name string, data []byte) (string, error) {
					if key != "file" {
						return "", errors.New("no such text")
					}
					return saveMIDI(folder, name, data)
				},
				SetText: func(key, value string) error {
					switch key {
					case "file":
						return load(value)
					case "controllers":
						cs, err := parseControllers(value)
						if err != nil {
							return err
						}
						for i, c := range cs {
							m.which[i].Store(int32(c))
						}
						return nil
					}
					return errors.New("no such text")
				},
				Describe: func(string) params.Set {
					return params.Set{Name: "midiplayer", Params: []params.Param{
						params.S("Player", "play", false),
						params.S("Player", "loop", false),
						params.K("Player", "speed", 0.25, 2, 1, params.Step(0.01)),
						params.S("Player", "follow", false, params.Label("follow the tempo")),
						params.S("Player", "sync", false, params.Label("follow the song")),
						params.K("Player", "modChannel", 0, 16, 0, params.Step(1), params.Label("mod outputs of channel (0: all)")),
						params.K("Player", "lift", 0, 1, 0, params.Label("velocity floor")),
						params.K("Player", "transpose", -24, 24, 0, params.Step(1), params.Unit("st")),
					}}
				},
				Actions: []patcher.Action{
					{Key: "restart", Label: "Rewind", Title: "Back to the start of the file", Do: func() error {
						if p := m.src.player(); p != nil {
							p.Seek(0)
						}
						return nil
					}},
				},
				Status: func() string {
					mu.Lock()
					defer mu.Unlock()
					s := status
					if p := m.src.player(); p != nil && file != "" {
						s += fmt.Sprintf("  %d:%02d", int(p.Position())/60, int(p.Position())%60)
						if p.Playing() {
							s += " ▶"
						}
					}
					return s
				},
				Extra: func() (k []psalm.Modular) {
					for _, c := range m.chans {
						k = append(k, c)
					}
					return
				}(),
				NotesOut: func(out int, targets []psalm.Receiver) {
					k := m.Keyboard
					if out > 0 {
						k = m.chans[out-1]
					}
					if slices.Equal(m.plays[out], targets) {
						return
					}
					m.plays[out] = targets
					k.AllNotesOff(psalm.CtrlTick{})
					k.SetTargets(targets...)
				},
				Mod: m.out}, nil
		}}
}

// midiFiles lists the MIDI files in a folder and below it (as paths to type), the list kept a few seconds: the page asks for it with every update
func midiFiles(folder string) func() []string {
	var (
		mu   sync.Mutex
		at   time.Time
		list []string
	)
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		if time.Since(at) < 3*time.Second {
			return list
		}
		at, list = time.Now(), []string{}
		_ = filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {
			if err != nil || len(list) >= 500 {
				return filepath.SkipDir
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".mid", ".midi", ".rmi":
				if !d.IsDir() {
					list = append(list, filepath.ToSlash(path))
				}
			}
			return nil
		})
		slices.Sort(list)
		return list
	}
}

// saveMIDI keeps a MIDI file the page sent in the folder, under its own name (a file of the same name with other notes gets a number), and returns the path
func saveMIDI(folder, name string, data []byte) (string, error) {
	if _, err := midi.ParseFile(data); err != nil {
		return "", fmt.Errorf("%s: %v", name, err)
	}
	base := strings.Map(func(r rune) rune {
		if r == ' ' || r > 127 || strings.ContainsRune(`/\:*?"<>|`, r) || r < 32 {
			return '_'
		}
		return r
	}, filepath.Base(strings.ReplaceAll(name, "\\", "/")))
	ext := strings.ToLower(filepath.Ext(base))
	if ext != ".mid" && ext != ".midi" && ext != ".rmi" {
		base += ".mid"
		ext = ".mid"
	}
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	if stem == "" || stem == "." {
		stem = "upload"
	}
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return "", err
	}
	for i := 1; ; i++ {
		path := filepath.Join(folder, stem+ext)
		if i > 1 {
			path = filepath.Join(folder, fmt.Sprintf("%s-%d%s", stem, i, ext))
		}
		have, err := os.ReadFile(path)
		if err == nil && bytes.Equal(have, data) { // the same file again
			return filepath.ToSlash(path), nil
		}
		if err == nil {
			continue
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return "", err
		}
		return filepath.ToSlash(path), nil
	}
}
