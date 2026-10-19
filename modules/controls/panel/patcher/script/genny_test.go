package script

import (
	"strings"
	"testing"

	"github.com/almerlucke/genny"
)

// eval runs JavaScript with genny, what its last expression is as JSON
func eval(t *testing.T, code string) (string, error) {
	t.Helper()
	r := NewRuntime()
	_ = r.vm.Set("genny", r.genny)
	v, err := r.vm.RunString("JSON.stringify((() => {" + code + "})())")
	if err != nil {
		return "", cleanError(err)
	}
	return v.String(), nil
}

func TestGenny(t *testing.T) {
	for _, c := range []struct{ name, code, want string }{
		{"sequence", `return genny.sequence([1, 2, 3]).take(5)`, `[1,2,3,1,2]`},
		{"sequence once", `const g = genny.sequence([1, 2], false); const a = g.take(2); return [a, g.done(), (g.reset(), g.next())]`, `[[1,2],true,1]`},
		{"bucket", `return genny.bucket([1, 2, 3]).take(3).sort()`, `[1,2,3]`},
		{"bucket random", `return genny.bucket([7], "random").take(3)`, `[7,7,7]`},
		{"arpeggio", `return genny.arpeggio([1, 2, 3], "up", "exclusive").take(6)`, `[1,2,3,2,1,2]`},
		{"arpeggio reverse", `return genny.arpeggio([1, 2, 3], "up", "none", true).take(3)`, `[3,2,1]`},
		{"repeat", `const g = genny.repeat([1, 2], 3); return [g.take(5), g.done()]`, `[[1,2,1,1,1],true]`},
		{"repeat over", `return genny.repeat(genny.sequence([1, 2], false), 2).take(5)`, `[1,2,1,2,2]`},
		{"and", `return genny.and(genny.sequence([1, 2], false), genny.sequence([9], false)).take(5)`, `[1,2,9,1,2]`},
		{"or", `return genny.or("random", genny.constant(4), [4]).take(3)`, `[4,4,4]`},
		{"combine", `return genny.combine([1, 2], genny.constant("x")).take(2)`, `[[1,"x"],[2,"x"]]`},
		{"continuous", `return genny.continuous(genny.sequence([1, 2], false)).take(3)`, `[1,2,1]`},
		{"from function", `let n = 0; return genny.from(() => n++).take(3)`, `[0,1,2]`},
		{"from function*", `const g = genny.from(function* () { yield 1; yield 2 }); const a = g.take(3); g.reset(); return [a, g.done(), g.next()]`, `[[1,2,2],false,1]`},
		{"function* done", `const g = genny.from(function* () { yield 1; yield 2 }); g.next(); const d = g.done(); g.next(); return [d, g.done()]`, `[false,true]`},
		{"from iterator", `return genny.from([5, 6][Symbol.iterator]()).take(3)`, `[5,6,6]`},
		{"in and", `function* two() { yield "a"; yield "b" } return genny.and(two, genny.sequence([1], false)).take(4)`, `["a","b",1,"a"]`},
		{"rand", `return genny.rand(2, 3).take(100).every(v => v >= 2 && v < 3)`, `true`},
		{"rwalk", `return genny.rwalk(5, 0, 10, 0.5).take(200).every(v => v >= 0 && v <= 10)`, `true`},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := eval(t, c.code)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("%s: %s, want %s", c.code, got, c.want)
			}
		})
	}
}

// what is wrong is an error of the script, one it can catch
func TestGennyErrors(t *testing.T) {
	for _, c := range []struct{ code, want string }{
		{`return genny.bucket([1], "sideways")`, `bucket: no "sideways"`},
		{`return genny.and()`, "at least one"},
		{`return genny.from(() => { throw new Error("inside") }).next()`, "inside"},
		{`try { genny.arpeggio([1], "down") } catch (e) { return "caught" }`, ""},
	} {
		got, err := eval(t, c.code)
		if c.want == "" {
			if err != nil || got != `"caught"` {
				t.Errorf("%s: %s %v, want it caught", c.code, got, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want an error of %q", c.code, err, c.want)
		}
	}
}

func TestGennyMore(t *testing.T) {
	for _, c := range []struct{ name, code, want string }{
		{"rand default", `return genny.rand().take(50).every(v => v >= 0 && v < 1)`, `true`},
		{"choice", `return genny.choice(["a", "b", "c"], [0, 1, 0]).take(4)`, `["b","b","b","b"]`},
		{"choice even", `return new Set(genny.choice([1, 2]).take(200)).size`, `2`},
		{"markov", `return genny.markov({C3: {E3: 1}, E3: {G3: 1}, G3: {C3: 1}}).take(5)`, `["C3","E3","G3","C3","E3"]`},
		{"markov start", `return genny.markov({a: {b: 1}, b: {a: 1}}, "b").take(3)`, `["b","a","b"]`},
		{"walk", `return genny.walk([5], [1, 2, 3, 4, 5]).take(100).every(v => v >= 1 && v <= 5)`, `true`},
		{"walk 2d", `return genny.walk([2, 3], [1, 2, 3, 4, 5, 6]).take(100).every(v => v >= 1 && v <= 6)`, `true`},
		{"logistic", `return genny.logistic(3.9).take(200).every(v => v >= 0 && v <= 1)`, `true`},
		{"henon", `return genny.henon().take(200).every(v => Math.abs(v) < 2)`, `true`},
		{"ramp", `const g = genny.ramp(5, 0, 8); return [g.take(5), g.done()]`, `[[0,2,4,6,8],true]`},
		{"ramp loop", `return genny.ramp(3, 0, 1, 1, true).take(5)`, `[0,0.5,1,1,1]`},
		{"euclid 3 of 8", `return genny.euclid(3, 8).take(8).map(x => x ? "x" : ".").join("")`, `"x..x..x."`},
		{"euclid rotate", `return genny.euclid(3, 8, 1).take(8).filter(x => x).length`, `3`},
		{"euclid loops", `return genny.euclid(1, 2).take(4)`, `[true,false,true,false]`},
		{"flatten", `return genny.flatten(genny.sequence([[1, 2], [3]])).take(5)`, `[1,2,3,1,2]`},
		{"map", `return genny.sequence([1, 2]).map(v => v * 10).take(3)`, `[10,20,10]`},
		{"map function", `return genny.map([1, 2], v => v + 1).take(2)`, `[2,3]`},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := eval(t, c.code)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("%s: %s, want %s", c.code, got, c.want)
			}
		})
	}
	for _, c := range []struct{ code, want string }{
		{`genny.choice([1, 2], [1])`, "weights"},
		{`genny.markov({a: {z: 1}})`, "not a state"},
		{`genny.markov({a: {}})`, "needs"},
		{`genny.walk([1], [1])`, "2 values"},
		{`genny.walk([2], [1])`, "grid"},
		{`genny.ramp(1, 0, 1)`, "2 steps"},
		{`genny.euclid(9, 8)`, "euclid"},
		{`genny.map([1], 3)`, "function"},
	} {
		if _, err := eval(t, "return "+c.code); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want an error of %q", c.code, err, c.want)
		}
	}
}

// the same seed gives the same draws
func TestGennySeeded(t *testing.T) {
	code := `return [genny.choice([1,2,3,4]).take(8), genny.rand().take(3), genny.markov({a:{a:1,b:1},b:{a:1,b:1}}).take(8)]`
	seed := func() string {
		genny.Seed(3)
		v, err := eval(t, code)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	if a, b := seed(), seed(); a != b {
		t.Fatalf("%s != %s", a, b)
	}
}

func TestShapers(t *testing.T) {
	for _, c := range []struct{ name, code, want string }{
		{"add", `return genny.shapers.add(12).shape(48)`, `60`},
		{"note name", `return genny.shapers.add(12).shape("C3")`, `60`},
		{"chord", `return genny.shapers.mult(2).shape([1, 2, 3])`, `[2,4,6]`},
		{"on a generator", `return genny.sequence([1, 2]).shape(genny.shapers.mult(10)).take(3)`, `[10,20,10]`},
		{"genny.shape", `return genny.shape([1, 2], genny.shapers.add(1)).take(2)`, `[2,3]`},
		{"function as shaper", `return genny.sequence([1, 2]).shape(x => x * x).take(2)`, `[1,4]`},
		{"quantize", `return genny.shapers.quantize(12).shape([50, 59, 70])`, `[48,60,72]`},
		{"mirror", `return genny.shapers.mirror(0, 10).shape([12, -3, 5])`, `[8,3,5]`},
		{"linear", `return genny.shapers.linear(2, -1).shape(0.5)`, `0`},
		{"bipolar", `return genny.shapers.bipolar().shape([0, 1])`, `[-1,1]`},
		{"pulse", `return genny.shapers.pulse(0.5).shape([0.2, 0.7])`, `[1,0]`},
		{"clip hard", `return genny.shapers.clip("hard", -1, 1).shape([-5, 0.5, 5])`, `[-1,0.5,1]`},
		{"clip soft", `return genny.shapers.clip("soft", 2).shape(0) === 0`, `true`},
		{"chebyshev", `return genny.shapers.chebyshev({2: 1}).shape(0.5)`, `-0.5`},
		{"series", `return genny.shapers.series(genny.shapers.add(1), genny.shapers.mult(3)).shape(1)`, `6`},
		{"parallel", `return genny.shapers.parallel(genny.shapers.add(1), genny.shapers.mult(3)).shape(1)`, `5`},
		{"mixer", `return genny.shapers.mixer([genny.shapers.add(1), genny.shapers.mult(3)], [1, 0.5]).shape(1)`, `3.5`},
		{"interp", `return genny.shapers.interp(0.5, genny.shapers.add(0), genny.shapers.add(10)).shape(1)`, `6`},
		{"route", `return genny.shapers.route(1, genny.shapers.add(0), genny.shapers.add(10)).shape(1)`, `11`},
		{"table of values", `return genny.shapers.table([0, 10]).shape([0, 0.5, 1, 7])`, `[0,5,10,10]`},
		{"table of a wave", `return genny.shapers.table("sawtooth", 4).shape([0, 0.25, 0.5, 0.75]).map(x => Math.round(x * 100) / 100)`, `[-1,-0.5,0,0.5]`},
		{"chain", `return genny.shapers.chain("mult 3 | add 1").shape([1, 2])`, `[4,7]`},
		{"chain macros", `return genny.shapers.chain("mult m1[0,10] | add m2", [1, 0.5]).shape(2)`, `20.5`},
		{"chain in a series", `return genny.sequence([0.25]).shape(genny.shapers.series(genny.shapers.chain("sin"), genny.shapers.add(1))).take(1)`, `[2]`},
		{"emulations", `return ["supersaw", "hardsync", "softsync", "trimod", "voyager", "pwmod"].every(n => { const s = genny.shapers[n](); return [0, 0.3, 0.9].every(x => Number.isFinite(s.shape(x))) })`, `true`},
		{"shapes follow the generator", `const g = genny.sequence([1, 2], false).shape(genny.shapers.add(1)); const a = g.take(2); return [a, g.done()]`, `[[2,3],true]`},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := eval(t, c.code)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("%s: %s, want %s", c.code, got, c.want)
			}
		})
	}
	for _, c := range []struct{ code, want string }{
		{`genny.shapers.mod(0)`, "not 0"},
		{`genny.shapers.quantize(0)`, "not 0"},
		{`genny.shapers.mirror(2, 1)`, "bottom below top"},
		{`genny.shapers.clip("wild")`, "hard or soft"},
		{`genny.shapers.route(5, genny.shapers.add(1))`, "index"},
		{`genny.shapers.series()`, "needs shapers"},
		{`genny.shapers.chain("wobble")`, "not a stage"},
		{`genny.shapers.chain("mod 0")`, "not 0"},
		{`genny.shapers.table("zigzag")`, "no \"zigzag\""},
		{`genny.shapers.chebyshev({x: 1})`, "harmonic"},
		{`genny.sequence([1]).shape(3)`, "a shaper"},
		{`genny.sequence(["wat"]).shape(genny.shapers.add(1)).next()`, "wat"},
		{`genny.sequence([0]).shape(x => 1 / x * 0 / 0).next()`, "NaN"},
	} {
		if _, err := eval(t, "return "+c.code); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want an error of %q", c.code, err, c.want)
		}
	}
}
