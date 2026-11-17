// the sea and the birds answer the steel pan in its solo: pieces of a recording of the sea with birds, cut at its hits, three in twelve turned every bar
clock("1/16")
const calls = shared.calls("sea and birds", { hits: { 3: 3 }, turn: 7, on: shared.layers.sea, vel: 0.5, ref: -30 })
function trigger(ev) { calls(song, note, ev) }
