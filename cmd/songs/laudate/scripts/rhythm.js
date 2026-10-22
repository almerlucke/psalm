// RHYTHM: the clock of the lead. It has a clock of its own, sixteenths, and no sound: what it plays is triggers (fire), to the piano script. The music goes in
// phrases of two bars, four to a group (see the score): the first phrase of a group makes a rhythm, the second says it again, the third is the answer, the fourth says
// the first again with a few steps changed. A euclidean cycle of the hits of the section in sixteen steps (5, 7 or 9) says where something may happen, and a markov
// chain says what: a single hit, a rest, a ratchet of two or three (fire(0, 2), fire(0, 3)), or a hit late by half a step (fire(0.5)). The second bar of a phrase is the
// cycle turned five steps, so the phrase has an inside.
clock("1/16", 0.58)
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
const weights = (d, rat) => {
  const r = Math.max(0.01, rat)
  return {
    hit:  { hit: 2 + 3 * d, rest: 2, r2: 1.5 * r, r3: r, late: 1 + d },
    rest: { hit: 3 + 3 * d, rest: 3 - 2 * d, r2: r, r3: 0.5 * r, late: 1 },
    r2:   { hit: 3, rest: 3 + 2 * r, late: 1, r2: 0.2 },
    r3:   { hit: 2, rest: 5, late: 1 },
    late: { hit: 3 + d, rest: 3, r2: 0.5 * r },
  }
}
let chain = null, mem = null, section = "", phrase = -1
// a phrase: thirty-two steps, each nothing or what happens in it
function make(s, turn) {
  const out = []
  for (let b = 0; b < 2; b++) {
    const grid = genny.euclid(s.hits, 16, (turn + 5 * b) % 16)
    for (let i = 0; i < 16; i++) out.push(grid.next() ? chain.next() : "rest")
  }
  return out
}
function trigger(ev) {
  if (!shared.known()) return
  const s = shared.s()
  if (!s.hits) return
  if (section !== song.section) { section = song.section; chain = genny.markov(weights(s.lead, s.rat), "hit"); mem = null }
  const p = shared.phrase(), q = shared.q(), k = ev.step % 32
  if (p !== phrase || !mem) {
    phrase = p
    if (q === 0 || !mem) { mem = make(s, (p * 3) % 16); shared.lead = { on: !!s.lp[q], kind: "new" } }
    else if (q === 3) {                                             // the first again, a little changed
      mem = mem.map(t => (t !== "rest" && R() < 0.25) ? chain.next() : t)
      shared.lead = { on: !!s.lp[q], kind: "var" }
    } else shared.lead = { on: !!s.lp[q], kind: "again" }
  }
  if (!shared.lead.on) return
  switch (mem[k]) {
    case "hit": fire(); break
    case "r2": fire(0, 2); break
    case "r3": fire(0, 3); break
    case "late": fire(0.5); break
  }
}
