// TABLA: the loop of four bars, cut at its hits, 44 pieces on the keys of a sampler, put where they were: each verse is the loop once, and
// the verse remixes it. The operations are few, and a markov chain chooses them (the weights are holy numbers): a bar rotated by a few
// steps, two bars swapped, a bar turned round, a few steps repeated to the end of the bar (a stutter), one piece repeated on four steps (a
// roll), the weaker hits thinned out. The more heat a verse has the more of them. The last bar is always the loop as it was: the refrain that
// returns. In the verse of the dividing of the Red Sea the hits part and leave a gap, and come together again in a roll.
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
const base = Array(64).fill(null)
for (const p of shared.slices.tabla) {
  const s = Math.round(p.at)
  if (s >= 0 && s < 64 && base[s] === null) base[s] = p
}
const bar = (pat, b) => pat.slice(b * 16, b * 16 + 16)
const put = (pat, b, a) => { for (let i = 0; i < 16; i++) pat[b * 16 + i] = a[i] }
const pick = list => list[Math.floor(R() * list.length)]
const three = () => Math.floor(R() * 3)             // one of the first three bars

const ops = {
  rotate(pat) { const n = pick([2, 3, 4, 6]), b = three(), a = bar(pat, b); put(pat, b, a.map((_, i) => a[(i - n + 16) % 16])) },
  swap(pat) { const a = three(), b = (a + 1 + Math.floor(R() * 2)) % 3, x = bar(pat, a), y = bar(pat, b); put(pat, a, y); put(pat, b, x) },
  reverse(pat) { const b = three(); put(pat, b, bar(pat, b).reverse()) },
  stutter(pat) {
    const b = three(), at = pick([0, 4, 8]), len = pick([2, 3, 4]), a = bar(pat, b), cell = a.slice(at, at + len)
    for (let i = at + len, j = 0; i < 16; i++, j++) a[i] = cell[j % len]
    put(pat, b, a)
  },
  roll(pat) {
    const b = three(), a = bar(pat, b)
    let at = a.findIndex((x, i) => x !== null && i >= 4 && i <= 11)
    if (at < 0) at = a.findIndex(x => x !== null)
    if (at < 0) return
    const p = a[at]
    for (let i = 0; i < 4 && at + i < 16; i++) a[at + i] = p
    put(pat, b, a)
  },
  thin(pat) { const b = three(), a = bar(pat, b); put(pat, b, a.map((x, i) => i % 4 !== 0 && R() < 0.45 ? null : x)) },
}
const chain = genny.markov({
  rotate:  { swap: 12, stutter: 7, roll: 3, thin: 3 },
  swap:    { stutter: 12, reverse: 7, rotate: 3, thin: 3 },
  reverse: { roll: 12, rotate: 7, thin: 7, stutter: 3 },
  stutter: { thin: 12, rotate: 7, roll: 7, reverse: 3 },
  roll:    { rotate: 12, swap: 7, thin: 7, stutter: 3 },
  thin:    { swap: 12, roll: 7, stutter: 7, rotate: 3 },
}, "rotate")

// the sea divided: the first bar as it was, then a few hits and a gap that grows, a roll rising at the end, and the loop back
function parting(pat) {
  for (let i = 20; i < 48; i++) if (!(i >= 44)) pat[i] = null
  const a = bar(pat, 2)
  const key = (base.find(x => x && x.at > 10) || base.find(x => x)).key
  put(pat, 2, a.map((x, i) => i >= 12 ? { key: key + (i - 12) % 4, len: 2 } : null))
}

function patternFor(v) {
  const pat = base.slice()
  if (v === 12) parting(pat)
  else for (let i = 0; i < shared.heat[v]; i++) ops[chain.next()](pat)
  put(pat, 3, bar(base, 3))         // the refrain
  return pat
}

let pat = null, lastVerse = -1
function trigger(ev) {
  if (!shared.known()) return
  const v = shared.verse(), k = shared.step(ev)
  if (v !== lastVerse) { lastVerse = v; pat = patternFor(v) }
  const p = pat[k]
  if (!p) return
  const accent = k % 4 === 0 ? 0.88 : 0.74
  note(p.key, (accent + 0.08 * R()) * shared.fade(), p.len)
}
