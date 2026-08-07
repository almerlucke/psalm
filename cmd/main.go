package main

import (
	"log/slog"
	"os"

	"github.com/almerlucke/genny/float/rand"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/gen"
	"github.com/almerlucke/psalm/modules/controls/logger"
	"github.com/almerlucke/psalm/modules/controls/timer"
)

func main() {
	l := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	p := psalm.New()

	tmr := timer.New(p.PPQ(), true).AddTo(p).SetID("timer")
	p2 := psalm.NewPatch([]int{1}, []int{1})
	p2.AddTo(p)
	p2.ValueIn(tmr)
	g := gen.New[float64](rand.New(0.0, 10.0)).ValueIn(p2.PatchInput(0))
	g.AddTo(p2)
	p2.ValueOut(g)

	lgr := logger.New(l).ValueIn(p2)

	l.Info("starting", "logger", lgr)

	_ = p.Listen()
}
