// the beasts of the earth call on the beat of the percussion: pieces of a recording of animals cut at their calls, walked by a markov chain (mostly on to the next, now and
// then two on, one back or the same again) on five hits in twelve in the break, three in the head out, where the percussion is at its fullest
clock("1/16")
const calls = shared.calls("animals", { hits: { 4: 5, 6: 3 }, turn: 0, on: shared.layers.animals, vel: 0.5, ref: -30 })
function trigger(ev) { calls(song, note, ev) }
