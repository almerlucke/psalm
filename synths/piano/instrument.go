package piano

import (
	"math"
	"math/cmplx"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/effects/convolver"
)

// Piano is the whole instrument: the voices, the strings of all 88 keys ringing along with what is
// played when their dampers are up (the sustain pedal, a held key, the top keys that have no
// dampers), and the soundboard every string sounds through. It takes the messages of a poly synth
// (triggers with a duration or noteOn and noteOff); the pedals are "sustain" (0 up, 1 down, half
// pedaling in between), "sostenuto" (above 0.5 is down: the keys held when it goes down stay
// undamped until it comes up) and "soft" (above 0.5 is down). A message without a command reaches
// every voice.
//
// The voices are handed out like the keys of a piano: a key that still sounds is struck again on
// its own strings (no new voice, nothing cut off); a new key takes a free voice, or, when all sound,
// the quietest one (released keys first), faded out over a buffer
type Piano struct {
	*psalm.Module
	cfg       Config
	voices    []*Voice
	order     []int64 // when each voice was struck
	strikes   int64
	mixed     psalm.Channels
	fadeOut   psalm.Channels
	fading    bool
	bank      []*sympathetic
	board     *convolver.Convolver // the soundboard
	ir        [][]float64          // its response
	listener  *listener            // where the piano is heard from
	power     []float64            // its power response, for the voicing
	hz        float64
	dry       [2][]float64 // the strings, delayed as long as the soundboard takes
	dryPos    int
	sustain   float64   // the sustain pedal, 0 up to 1 down
	caught    [128]bool // the keys the sostenuto pedal holds undamped
	sostenuto bool
	pedal     struct { // the noise of the sustain pedal: the dampers lifting and landing
		noise, lp, hp float64
		thump         complex128
	}
	idle     int       // samples everything has been silent: the strings, the soundboard
	held     [128]bool // keys whose dampers the hand lifts
	sounding [128]bool // keys struck, their strings are the voices
}

// sympathetic is the string of a key that is not played, ringing along
type sympathetic struct {
	str
	pitch        float64
	gainL, gainR float64
	quiet        int // samples it has been silent with its damper down
}

// idleSamples is how long the instrument has to be silent to rest (the soundboard has died away)
const idleSamples = 44100

// the noise of the sustain pedal, at Mechanics 1 and Level 1
const (
	pedalSwish = 0.0015
	pedalThump = 0.0015
)

// the coupling of the bridge to the strings that ring along, at Resonance 1
const resonanceCoupling = 0.0005

// NewPoly creates a piano with a number of voices: a voice per key that sounds at once (with the
// pedal down notes ring for many seconds, give it enough)
func NewPoly(cfg Config, numVoices int) *Piano {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	voices := make([]*Voice, numVoices)
	for i := range voices {
		voices[i] = New(cfg)
	}
	p := &Piano{
		Module: psalm.NewModule([]int{psalm.Mono}, []int{cfg.NumChannels}),
		cfg:    cfg,
		voices: voices,
		order:  make([]int64, numVoices),
	}
	for _, v := range voices {
		v.sostenuto = &p.caught
	}
	p.mixed = psalm.NewChannels(cfg.NumChannels, p.Config().BufferSize)
	p.fadeOut = psalm.NewChannels(cfg.NumChannels, p.Config().BufferSize)
	ir := cfg.SoundboardIR
	if len(ir) == 0 {
		ir = boardResponse(p.Config().SampleRate)
	}
	p.board = convolver.New(ir, convolver.Config{PartitionSize: boardPartition, Mix: 1, Gain: 1})
	p.ir = ir
	p.listener = newListener(cfg.Position, p.Config().SampleRate)
	p.power, p.hz = boardPower(ir, p.Config().SampleRate)
	p.voice()
	p.dry = [2][]float64{make([]float64, boardPartition), make([]float64, boardPartition)}
	p.buildBank()
	p.SetSelf(p)
	return p
}

// buildBank sets up the strings of all keys for ringing along: one string each, the aftersound
// polarization (the one that keeps its energy)
func (p *Piano) buildBank() {
	cfg := &p.cfg
	sr := p.Config().SampleRate
	disp := dispersion(sr, math.Max(0, cfg.Stiffness))
	p.bank = p.bank[:0]
	for key := 21; key <= 108; key++ {
		pitch := float64(key)
		d := stringDesign{
			sr:         sr,
			f0:         440 * math.Exp2((stretched(pitch, cfg)-69)/12),
			stages:     disp[key].stages,
			ad:         disp[key].ad,
			t60:        ringTime(pitch) * math.Max(0.05, cfg.Decay) * afterFactor,
			damping:    highDamping * math.Max(0, cfg.Damping),
			damperTime: math.Max(0.02, cfg.DamperTime) * (1 + math.Max(0, (48-pitch)/24)),
		}
		s := &sympathetic{pitch: pitch}
		s.reset(designString(d))
		pan := keyPan(pitch, cfg)
		s.gainL, s.gainR = math.Cos(pan*math.Pi/2), math.Sin(pan*math.Pi/2)
		s.damper = s.damped
		p.bank = append(p.bank, s)
	}
}

// voice sets the gains of the keys that even out the soundboard (see voicing)
func (p *Piano) voice() {
	gains := voicing(p.power, p.hz, p.Config().SampleRate, p.cfg)
	for _, v := range p.voices {
		v.voicing = gains
	}
}

// AllNotesOff releases all held notes
func (p *Piano) AllNotesOff() {
	for _, v := range p.voices {
		v.NoteOff()
		v.SetID("")
	}
}

func (p *Piano) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		p.ReceiveMessage(msg)
		return
	}
	for _, v := range p.voices {
		v.Receive(value, index)
	}
}

func (p *Piano) ReceiveMessage(msg psalm.Message) {
	command, ok := msg.StrVal("command")
	if ok && command == "trigger" {
		tick := msg.MustTickVal("tick")
		if id, ok := msg.StrVal("noteOff"); ok {
			for _, v := range p.voices {
				if v.ID() == id {
					v.NoteOff()
					v.SetID("")
				}
			}
			return
		}
		note := msg.MustMsg("message")
		amplitude := msg.MustFltVal("amplitude")
		if id, ok := msg.StrVal("noteOn"); ok {
			v, restrike := p.allocate(note)
			v.SetID(id)
			v.strike(amplitude, note, tick, -1, restrike)
			return
		}
		p.Note(msg.MustFltVal("duration"), amplitude, note, tick, p.Config())
		return
	}
	if ok && command != "voice" {
		return
	}

	// a setting for every voice: the instrument follows it
	for _, v := range p.voices {
		v.ReceiveMessage(msg)
	}
	p.cfg = p.voices[0].cfg
	msg.IfFltVal("sostenuto", func(f float64) {
		down := f > 0.5
		if down && !p.sostenuto {
			// it catches the dampers of the keys held now
			for _, v := range p.voices {
				if v.Held() {
					p.caught[v.Key()] = true
				}
			}
		}
		if !down {
			clear(p.caught[:])
		}
		p.sostenuto = down
	})
	msg.IfFltVal("sustain", func(f float64) {
		f = math.Max(0, math.Min(f, 1))
		down, was := f > 0.5, p.sustain > 0.5
		if down != was {
			// all dampers lift with a soft swish, and land with a thud
			mech := math.Max(0, math.Min(p.cfg.Mechanics, 1)) * math.Max(p.cfg.Level, 1e-9)
			if down {
				p.pedal.noise += mech * pedalSwish
			} else {
				p.pedal.noise += mech * pedalSwish * 0.5
				p.pedal.thump += complex(mech*pedalThump, 0)
			}
		}
		p.sustain = f
	})
	for _, k := range []string{"body", "hardness", "strikePosition", "stiffness", "decay", "damping"} {
		if _, ok := msg[k]; ok {
			p.voice()
			break
		}
	}
	if _, ok := msg["position"]; ok {
		p.listener = newListener(p.cfg.Position, p.Config().SampleRate)
	}
	for _, k := range []string{"stiffness", "decay", "damping", "damperTime", "width", "stretch", "position"} {
		if _, ok := msg[k]; ok {
			p.buildBank()
			return
		}
	}
}

// Note plays a note (like a trigger message with a duration)
func (p *Piano) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v, restrike := p.allocate(message)
	v.strike(amplitude, message, tick, int(duration*p.Config().SampleRate)+tick.SampleOffset, restrike)
}

// allocate returns the voice for a note: the voice of its key if that still sounds (to strike it
// again), a free voice, or the quietest (a released key before a held one), faded out
func (p *Piano) allocate(note psalm.Message) (*Voice, bool) {
	pitch := 60.0
	note.IfFltVal("pitch", func(f float64) { pitch = f })
	note.IfFltVal("frequency", func(f float64) { pitch = 69 + 12*math.Log2(f/440) })
	key := keyOf(pitch)
	p.strikes++

	pick := func(i int, restrike bool) (*Voice, bool) {
		p.order[i] = p.strikes
		return p.voices[i], restrike
	}

	// the key is still sounding: strike it again (the latest voice of the key)
	same := -1
	for i, v := range p.voices {
		if v.IsActive() && v.Key() == key && (same < 0 || p.order[i] > p.order[same]) {
			same = i
		}
	}
	if same >= 0 {
		return pick(same, true)
	}
	for i, v := range p.voices {
		if !v.IsActive() {
			return pick(i, false)
		}
	}

	// all voices sound: take the quietest, keys held down count as louder
	quietest, least := 0, math.Inf(1)
	for i, v := range p.voices {
		level := v.peak
		if v.Held() {
			level *= 4
		}
		if level < least {
			quietest, least = i, level
		}
	}
	v := p.voices[quietest]
	v.Synthesize()
	n := float32(len(p.fadeOut[0]))
	for c, ch := range v.Output(0).Channels {
		for i, x := range ch {
			p.fadeOut[c][i] += x * (1 - float32(i)/n)
		}
	}
	p.fading = true
	v.Clear()
	return pick(quietest, false)
}

func (p *Piano) Synthesize() {
	// the voices, and a voice taken while sounding fading out
	if p.fading {
		p.mixed.Copy(p.fadeOut)
		p.fadeOut.Clear()
		p.fading = false
	} else {
		p.mixed.Clear()
	}
	for _, v := range p.voices {
		if v.IsActive() {
			v.Synthesize()
			p.mixed.Accum(v.Output(0).Channels)
		}
	}

	// nothing sounds: rest (the soundboard tail and the strings ringing along have died away)
	in0 := p.mixed
	if p.idle > idleSamples && silent(in0) && p.pedal.noise < 1e-9 && cmplx.Abs(p.pedal.thump) < 1e-9 {
		p.Output(0).Clear()
		return
	}

	var (
		cfg   = &p.cfg
		in    = p.mixed
		out   = p.Output(0).Channels
		sr    = p.Config().SampleRate
		glide = 1 - math.Exp(-1/(0.01*sr))
		level = cfg.Level
	)
	if level <= 0 {
		level = 1
	}
	coupling := resonanceCoupling * math.Max(0, math.Min(cfg.Resonance, 1)) / level
	body := math.Max(0, math.Min(cfg.Body, 1))
	direct := 1 - 0.7*body
	pedalLP := 1 - math.Exp(-2*math.Pi*1500/sr)
	pedalHP := 1 - math.Exp(-2*math.Pi*150/sr)
	pedalDecay := math.Exp(-1 / (0.06 * sr))
	pedalRot := cmplx.Rect(math.Exp(-1/(0.03*sr)), 2*math.Pi*60/sr)
	boardIn := p.board.Input(0).Channels

	// the keys whose dampers are lifted by the hand, and the keys sounding (their strings ring in
	// the voices, they are not driven again here; a key held down silently does ring along)
	clear(p.held[:])
	clear(p.sounding[:])
	for _, v := range p.voices {
		key := max(0, min(int(math.Round(v.Pitch())), 127))
		if v.Held() {
			p.held[key] = true
		}
		if v.IsActive() && v.force > 1e-4 {
			p.sounding[key] = true
		}
	}

	for i := range out[0] {
		l := float64(in[0][i])
		r := l
		if len(in) > 1 {
			r = float64(in[1][i])
		}

		// the strings that ring along: what is played reaches them through the bridge
		bridge := (l + r) * 0.5 * coupling
		var symL, symR float64
		if coupling > 0 {
			for _, s := range p.bank {
				key := int(s.pitch)
				contact := 0.0
				if s.pitch <= highestDamper && !p.held[key] && !p.caught[key] {
					contact = damperContact(p.sustain, s.pitch)
				}
				if s.quiet > 4096 && (contact == 1 || bridge == 0) {
					continue // silent and nothing reaches it, nothing to do
				}
				target := dampedGain(s.damped, contact)
				x := 0.0
				if !p.sounding[key] {
					x = bridge * (1 - contact)
				}
				s.damper += (target - s.damper) * glide
				y := s.tick(x)
				if math.Abs(y) < 1e-7 {
					s.quiet++
				} else {
					s.quiet = 0
				}
				symL += y * s.gainL
				symR += y * s.gainR
			}
		}
		l += symL * level
		r += symR * level

		// the pedal
		if p.pedal.noise > 1e-9 || cmplx.Abs(p.pedal.thump) > 1e-9 {
			n := (p.Rand().Float64()*2 - 1) * p.pedal.noise
			p.pedal.lp += pedalLP * (n - p.pedal.lp)
			p.pedal.hp += pedalHP * (p.pedal.lp - p.pedal.hp)
			p.pedal.noise *= pedalDecay
			p.pedal.thump *= pedalRot
			x := p.pedal.lp - p.pedal.hp + real(p.pedal.thump)
			l += x
			r += x
		}

		// the strings reach the soundboard through the bridge, and are heard directly, delayed as
		// long as the soundboard takes to answer
		m := float32((l + r) * 0.5)
		boardIn[0][i], boardIn[1][i] = m, m
		dl, dr := p.dry[0][p.dryPos], p.dry[1][p.dryPos]
		p.dry[0][p.dryPos], p.dry[1][p.dryPos] = l, r
		p.dryPos = (p.dryPos + 1) % len(p.dry[0])
		out[0][i] = float32(dl)
		if len(out) > 1 {
			out[1][i] = float32(dr)
		}
	}

	p.board.Synthesize()
	wet := p.board.Output(0).Channels
	for i := range out[0] {
		// nearly all of a piano's sound is radiated by the board, some of the strings is kept
		// for clarity
		l := direct*float64(out[0][i]) + body*float64(wet[0][i])
		r := direct*float64(out[len(out)-1][i]) + body*float64(wet[1][i])
		l, r = p.listener.process(l, r)
		if math.Abs(l)+math.Abs(r) < 1e-7 {
			p.idle++
		} else {
			p.idle = 0
		}
		if len(out) == 1 {
			out[0][i] = float32((l + r) * 0.5)
		} else {
			out[0][i] = float32(l)
			out[1][i] = float32(r)
		}
	}
}

// SignalLatency: the strings are heard as late as the soundboard answers
func (p *Piano) SignalLatency() int { return boardPartition }

// silent returns whether all samples are zero
func silent(c psalm.Channels) bool {
	for _, ch := range c {
		for _, v := range ch {
			if v != 0 {
				return false
			}
		}
	}
	return true
}
