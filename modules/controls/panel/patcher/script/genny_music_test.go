package script

import "testing"

func TestGennyMusic(t *testing.T) {
	for _, c := range []struct{ name, code, want string }{
		{"euclidPattern", `return genny.euclidPattern(8, 3, 0)`, `[true,false,false,true,false,false,true,false]`},
		{"rests all", `return genny.rests([1, 2], 0).take(2)`, `[null,null]`},
		{"rests none", `return genny.rests([1, 2], 1).take(2)`, `[1,2]`},
		{"gated", `return genny.gated([5, 6, 7], [true, false, true]).take(3)`, `[5,null,7]`},
		{"swing", `return genny.swing(0.2).take(4).map(x => +x.toFixed(2))`, `[0,0.2,0,0.2]`},
		{"accent", `return genny.accent([2, 2, 3]).take(7).length`, `7`},
		{"divider", `return genny.divider(3, 0).take(6)`, `[true,false,false,true,false,false]`},
		{"poly", `return genny.poly([2, 3]).take(2)`, `[[true,true],[false,false]]`},
		{"ratchetPattern", `return genny.ratchetPattern([1, 2]).take(2)`, `[[0],[0,0.5]]`},
		{"automatonGate", `return genny.automatonGate(30, 8).take(4).length`, `4`},
		{"pulseGate", `return genny.pulseGate([3, 2], 0).take(6)`, `[true,true,true,true,true,true]`},
		{"pulseGate thin", `return genny.pulseGate([3, 2], 9).take(6)`, `[false,false,false,false,false,false]`},
		{"split", `return genny.split(1, 1, 1, 2)`, `[1]`},
		{"section", `return genny.section([{ gen: [1], steps: 2 }, { gen: [2], steps: 1 }]).take(6)`, `[1,1,2,1,1,2]`},
		{"scale degrees", `const s = genny.scale(60, "major"); return [s.degree(2), s.degree(7), s.quantize(61), s.degrees([0, 1, 2]).take(3)]`, `[64,72,60,[60,62,64]]`},
		{"scale array", `return genny.scale(60, [0, 3, 7]).degree(4)`, `75`},
		{"scale bad", `return genny.scale(60, "nope")`, ``},
		{"just scale", `return genny.justScale(60, [1, 1.5]).degree(1) > 60`, `true`},
		{"chord", `return [genny.chord(60), genny.chord(60, "minor7"), genny.chord(60, [0, 5])]`, `[[60,64,67],[60,63,67,70],[60,65]]`},
		{"diatonic", `return genny.diatonic(genny.scale(60, "major"), 1, 3)`, `[62,65,69]`},
		{"invert open transpose", `return [genny.invert([60, 64, 67], 1), genny.transpose([60, 64], 2)]`, `[[64,67,72],[62,66]]`},
		{"voiceLeader", `const v = genny.voiceLeader(48, 72); return v.next([60, 64, 67]).length`, `3`},
		{"progression", `return genny.progression("pop", 0).take(1)`, `[0]`},
		{"progression table", `return genny.progression({ 0: [[1, 1]], 1: [[0, 1]] }, 0).take(4)`, `[0,1,0,1]`},
		{"row", `const r = genny.row(12); return [r.length, [...r].sort((a, b) => a - b).join()]`, `[12,"0,1,2,3,4,5,6,7,8,9,10,11"]`},
		{"row forms", `return [genny.retrograde([1, 2, 3]), genny.transposeRow([1, 2], 5)]`, `[[3,2,1],[6,7]]`},
		{"serial", `return genny.serial([0, 4, 7], [["prime", 0], ["retro", 0]]).take(6)`, `[0,4,7,7,4,0]`},
		{"matrix", `return genny.matrix([0, 1, 2]).length`, `3`},
		{"mutate none", `return genny.mutate([1, 2, 3], {}).take(7)`, `[1,2,3,1,2,3,1]`},
		{"lsystem", `return genny.lsystem("A", { A: "AB", B: "A" }, 3, { A: 60, B: 67 }).take(5)`, `[60,67,60,60,67]`},
		{"lsystem no symbols", `return genny.lsystem("A", {}, 1)`, ``},
		{"numbers", `return [genny.fibonacci().take(6), genny.collatz(6, false).take(4), genny.thueMorse().take(4), genny.primes().take(5)]`, `[[0,1,1,2,3,5],[6,3,10,5],[0,1,1,0],[2,3,5,7,11]]`},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := eval(t, c.code)
			if c.want == "" {
				if err == nil {
					t.Fatalf("no error, got %s", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}

func TestGennySignals(t *testing.T) {
	for _, c := range []struct{ name, code, want string }{
		{"lfo", `const g = genny.lfo("square", 4); return g.take(4).every(x => x === 1 || x === -1)`, `true`},
		{"lfo unipolar", `return genny.lfo("sine", 8, { unipolar: true }).take(16).every(x => x >= 0 && x <= 1)`, `true`},
		{"lfo bad shape", `return genny.lfo("zigzag")`, ``},
		{"slew", `return genny.slew([0, 10], 1).take(3)`, `[0,1,0]`},
		{"noise", `return genny.noise(0.1, 2).take(50).every(x => x >= -1 && x <= 1)`, `true`},
		{"lissajous", `const g = genny.lissajous(1, 1, 0, 4); return g.take(3).map(p => p.length)`, `[2,2,2]`},
		{"lorenz", `const a = genny.lorenz().take(3); return [a.length, a[0].length, a[0] !== a[1]]`, `[3,3,true]`},
		{"rossler duffing", `return [genny.rossler().next().length, genny.duffing().next().length, genny.standardMap(1).next().length, genny.pendulum().next().length]`, `[3,2,2,2]`},
		{"gaussian clamp", `return genny.gaussian(0, 5, -1, 1).take(50).every(x => x >= -1 && x <= 1)`, `true`},
		{"distributions", `return [genny.exponential(1, 0, 2), genny.beta(2, 2, 10, 20), genny.cauchy(0, 1, -1, 1), genny.weibull(0, 1, 2, 0, 3), genny.pink(), genny.lehmer(3)].map(g => g.take(20).every(Number.isFinite))`, `[true,true,true,true,true,true]`},
		{"pick", `return genny.pick(["a", "b", "c"], "ratio", 0, 0, 1).take(3)`, `["c","c","c"]`},
		{"pick bad", `return genny.pick(["a"], "wavy")`, ``},
		{"shifting", `return genny.shifting(["a", "b"], 2, [1, 0], [0, 1]).take(4)`, `["a","a","b","b"]`},
		{"tendency", `return genny.tendency([{ n: 8, from: [0, 1], to: [5, 6] }]).take(8).every(x => x >= 0 && x <= 6)`, `true`},
		{"tendency narrow", `return genny.tendency([{ n: 4, from: [3, 3] }], { loop: true }).take(5)`, `[3,3,3,3,3]`},
		{"tendency none", `return genny.tendency([])`, ``},
		{"tendencyChoice", `return genny.tendencyChoice(["a", "b"], [{ n: 3, from: [0, 0] }], { loop: true }).take(3)`, `["a","a","a"]`},
		{"spray", `return genny.spray([10, 20], 0).length`, `2`},
		{"sieve", `const s = genny.sieve(3, 0, 0, 11); return [s.values(), s.complement().contains(3), s.union(genny.sieveOf(0, 11, [1])).contains(1), s.gen().take(5), s.gate(3).take(4)]`, `[[0,3,6,9],false,true,[0,3,6,9,0],[true,false,false,true]]`},
		{"funnel", `return genny.funnel([1, 6], [0, 7])`, `[0,7]`},
		{"allowIntervals", `return genny.allowIntervals(genny.sequence([0, 1, 7, 8]), [1, 6]).take(2).length`, `2`},
		{"noRepeats", `return genny.noRepeats(genny.sequence([1, 1, 2, 2, 3]), { stop: 10 }).take(3)`, `[1,2,3]`},
		{"noDuplicates", `return genny.noDuplicates(genny.sequence([1, 2, 3]), 3).take(3).length`, `3`},
		{"intervalWalk", `return genny.intervalWalk(0, 20, [1, 2]).take(6).every(x => x >= 0 && x <= 20)`, `true`},
		{"learn", `return genny.learn([1, 2, 3, 1, 2, 3], 1).take(4).every(x => x >= 1 && x <= 3)`, `true`},
		{"grammar", `return genny.grammar({ S: [[1, [60, 62]]] }, "S").take(4)`, `[60,62,60,62]`},
		{"grammar ref", `return genny.grammar({ S: [[1, [{ rule: "A" }, 1]]], A: [[1, [7]]] }, "S").take(2)`, `[7,1]`},
		{"grammar bad", `return genny.grammar({ S: [[1, [{ rule: "Z" }]]] }, "S")`, ``},
		{"jump", `return genny.jump([1, 2, 3, 4, 5, 6], 2, 2, 0).take(4).length`, `4`},
		{"stutter", `return genny.stutter([1, 2, 3], 1, 1).take(6)`, `[1,1,2,2,3,3]`},
		{"in a generator", `return genny.sequence([1, 2]).map(x => x * 2).take(2)`, `[2,4]`},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := eval(t, c.code)
			if c.want == "" {
				if err == nil {
					t.Fatalf("no error, got %s", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}
