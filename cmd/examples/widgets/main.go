// Widgets: play and shape synths from the page, with the widgets of the panel
// (modules/controls/panel/widgets): a keyboard (or the computer's keys: A W S E D F T G Y H U J K,
// Z and X an octave down and up), an XY pad on the filter and one on the mod wheel and the
// aftertouch, the amplitude and filter envelopes to drag into shape, drum pads (the number keys
// hit them too) and a scope of what comes out. Choose the synth: every one takes the same
// messages for its envelopes, its wheel and its aftertouch. Open http://localhost:8087.
//
//	go run ./cmd/examples/widgets
//	go run ./cmd/examples/widgets -theme neon -addr :8087   (from a tablet on the same network)
package main

import (
	"flag"
	"log"
	"maps"
	"slices"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/widgets"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/drumsynth"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/pd"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/supersaw"
	"github.com/almerlucke/psalm/synths/vocal"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/sndfile"
)

// synth is one to play: its name on the page and the synth
type synth struct {
	name   string
	module psalm.Modular
}

// playing forwards what the widgets send to the synth chosen, and keeps the settings sent, so the
// next synth chosen gets them too
type playing struct {
	synths []synth
	now    int
	sent   map[string]float64
}

func (pl *playing) ReceiveMessage(msg psalm.Message) {
	if _, ok := msg["command"]; !ok {
		for k, v := range msg {
			if f, ok := v.(float64); ok {
				pl.sent[k] = f
			}
		}
	}
	pl.synths[pl.now].module.Receive(msg, 0)
}

func (pl *playing) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		pl.ReceiveMessage(msg)
	}
}

// choose plays another synth (on the audio thread): the notes of the one left let go, the settings
// moved on the page sent to the new one
func (pl *playing) choose(i int) {
	if i == pl.now {
		return
	}
	if off, ok := pl.synths[pl.now].module.(interface{ AllNotesOff() }); ok {
		off.AllNotesOff()
	}
	pl.now = i
	if len(pl.sent) > 0 {
		msg := psalm.Message{}
		for k, v := range pl.sent {
			msg[k] = v
		}
		pl.synths[i].module.Receive(msg, 0)
	}
	log.Printf("playing %s", pl.synths[i].name)
}

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the page")
	theme := flag.String("theme", "default", "the theme of the page: default, midnight, paper, amber, contrast or neon")
	flag.Parse()

	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	table := gen.Brightness(32)
	pl := &playing{sent: map[string]float64{}, synths: []synth{
		{"subtractive lead", subtractive.NewPoly(subtractive.LeadPreset(), 12).AddTo(p)},
		{"subtractive pad", subtractive.NewPoly(subtractive.PadPreset(), 12).AddTo(p)},
		{"supersaw lead", supersaw.NewPoly(supersaw.Lead(), 12).AddTo(p)},
		{"wavetable pad", wavetable.NewPoly(wavetable.PadPreset(table), 12).AddTo(p)},
		{"fm electric piano", fm.NewPoly(fm.EPianoPreset(), 12).AddTo(p)},
		{"pd brass", pd.NewPoly(pd.Brass(), 12).AddTo(p)},
		{"pulsar vowel lead", pulsar.NewPoly(pulsar.VowelLeadPreset(), 12).AddTo(p)},
		{"vocal choir", vocal.NewPoly(vocal.Choir(), 8).AddTo(p)},
		{"waveguide violin", waveguide.NewPoly(waveguide.Violin(), 8).AddTo(p)},
	}}
	kit := drumsynth.NewPoly(drumsynth.Config{Kit: drumsynth.Kit909()}, 16).AddTo(p)

	mix := mixer.New(p)
	vienna := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"))
	mix.Bus("hall", mixer.StripConfig{}, convolver.New(vienna, convolver.Config{PartitionSize: 512, Mix: 1, Gain: 0.7}).AddTo(p))
	for _, s := range pl.synths {
		mix.Channel(s.name, s.module, mixer.StripConfig{Volume: -4, Sends: map[string]float64{"hall": 0.3}})
	}
	mix.Channel("drums", kit, mixer.StripConfig{Volume: -4, Sends: map[string]float64{"hall": 0.1}})
	p.Out(mix.Master())

	ctl := panel.New(p, "Widgets")
	t, ok := panel.Themes[*theme]
	if !ok {
		log.Fatalf("theme %q: one of %v", *theme, slices.Sorted(maps.Keys(panel.Themes)))
	}
	ctl.SetTheme(t)

	ctl.Section("Play")
	var names []string
	for _, s := range pl.synths {
		names = append(names, s.name)
	}
	ctl.Select("Synth", names, pl.choose)
	widgets.Keys(ctl, "Keys", widgets.Play(pl), widgets.From(48), widgets.Octaves(3))

	// the synths keep their own settings until a widget moves (NoDefault): then the one playing, and
	// every one chosen after it, gets the widget's
	ctl.Section("Shape")
	send := func(key string) panel.Target {
		return panel.Func(func(v float64) { pl.ReceiveMessage(psalm.Message{key: v}) })
	}
	widgets.XY(ctl, "Filter",
		widgets.Axis{Name: "cutoff", Min: 150, Max: 12000, Curve: panel.Exp, To: send("cutoff"), Default: 2000, Unit: "Hz"},
		widgets.Axis{Name: "resonance", Max: 0.95, To: send("resonance"), Default: 0.2},
		panel.NoDefault())
	widgets.XY(ctl, "Hands",
		widgets.Axis{Name: "wheel", Max: 1, To: send("modWheel")},
		widgets.Axis{Name: "pressure", Max: 1, To: send("pressure")})
	widgets.Envelope(ctl, "Amp", pl, "amp", widgets.ADSR{Attack: 5, Decay: 300, Sustain: 0.7, Release: 400}, panel.NoDefault())
	widgets.Envelope(ctl, "Filter env", pl, "filter", widgets.ADSR{Attack: 5, Decay: 300, Sustain: 0.3, Release: 400}, panel.NoDefault())

	ctl.Section("Drums")
	widgets.Pads(ctl, "Pads", widgets.Hit(kit, "kick", "snare", "clap", "hat", "open", "tomLow", "tomMid", "tomHigh",
		"rim", "cowbell", "crash", "ride", "congaLow", "congaHigh", "shaker", "tambourine"))

	ctl.Section("Out")
	widgets.Scope(ctl, "Scope", mix.Master())
	ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(mix.Master(), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03))
	ctl.Fader("Hall", -40, 6, panel.Linear, panel.Message(mix.Strip("hall"), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03))
	ctl.Meter("Master", mix.Master())

	go func() {
		if err := ctl.Serve(*addr); err != nil {
			log.Fatal(err)
		}
	}()
	defer ctl.Close()
	log.Printf("open http://%s: play the keys (or A W S E D ... on the computer), the pads (1 to 9)", *addr)
	if err := p.Listen(); err != nil {
		log.Fatal(err)
	}
}
