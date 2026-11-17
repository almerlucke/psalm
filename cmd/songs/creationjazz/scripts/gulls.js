// the gulls cry over the solo of the flute and the last bars of the tag: pieces of the recording of the sea birds cut at their cries, walked by a markov chain, four hits
// in twelve turned a little every bar, so they wander over the changes
clock("1/16")
const calls = shared.calls("gulls", { hits: { 5: 4, 7: 3 }, turn: 5, on: shared.layers.gulls, vel: 0.45, ref: -30 })
function trigger(ev) { calls(song, note, ev) }
