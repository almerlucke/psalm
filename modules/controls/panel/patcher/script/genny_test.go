package script

import (
	"strings"
	"testing"
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
