// "be fruitful, and multiply" (Genesis 1:28): the children laugh in bursts, five in twelve, through the last eight bars of the head out and the first four of the tag
clock("1/16")
const calls = shared.calls("children", { hits: { 6: 5, 7: 3 }, turn: 3, on: shared.layers.children, vel: 0.55, ref: -30 })
function trigger(ev) { calls(song, note, ev) }
