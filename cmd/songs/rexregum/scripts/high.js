// HIGH: the three higher voices of the choir, on the wavetable synth's evolvingPad. The program tunes it (patch.go: the position of the table sways on a
// slow wave, five detuned copies a note); this script opens the filter section by section from a dark, closed sound in the Dawn to an
// open one in the Throne, so the piece grows brighter as it climbs.
const cutoff = [1700, 2100, 2700, 3500, 4600, 6200, 8200]
function section(name, i) {
  tune({ cutoff: cutoff[i] })
}
function trigger(ev) {
  mod(BASE * shared.fadeIn(ev, 5) / 2) // in over the first five bars: the level of the strip
}
