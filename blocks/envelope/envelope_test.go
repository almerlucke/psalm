package envelope

import (
	"testing"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
)

func TestReceive(t *testing.T) {
	s := adsr.NewSetting(1, 5, 0.7, 200, 1, 300)
	Receive(psalm.Message{"ampAttack": 20.0, "ampDecay": 0.0, "ampSustain": 0.5, "ampRelease": 900.0, "filterAttack": 1.0}, "amp", s)
	if s.AttackDuration != 20 || s.DecayDuration != shortest || s.DecayLevel != 0.5 || s.ReleaseDuration != 900 {
		t.Errorf("setting %+v", s)
	}
	var a, r float64
	Seconds(psalm.Message{"ampAttack": 250.0, "ampRelease": 2000.0}, "amp", &a, nil, &r)
	if a != 0.25 || r != 2 {
		t.Errorf("seconds %g %g", a, r)
	}
}
