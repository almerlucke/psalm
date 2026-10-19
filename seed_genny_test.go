package psalm_test

import (
	"slices"
	"testing"

	"github.com/almerlucke/genny/float/rand"
	"github.com/almerlucke/psalm"
)

// psalm.Seed seeds the generators of genny too
func TestSeedSeedsGenny(t *testing.T) {
	draw := func() []float64 {
		psalm.Seed(5)
		r := rand.New(0, 1)
		return []float64{r.Generate(), r.Generate(), r.Generate()}
	}
	if a, b := draw(), draw(); !slices.Equal(a, b) {
		t.Fatalf("%v != %v", a, b)
	}
}
