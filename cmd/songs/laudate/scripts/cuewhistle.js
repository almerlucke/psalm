// CUE WHISTLE: says when the psalm I whistled (whistle_psalm150.wav) begins. It plays nothing itself: at the start of Dance plus START it fires one trigger, to the script of the
// sampler that plays the recording once, to the end, as it is. START is in seconds from the start of Dance: more is later, less is earlier (a negative one begins in the
// last bar of Greatness). Change it in the box of this script on the page to tweak the start; a tenth of a second is fine (a trigger can be late by a part of a step).
const START = 0
clock("1/16")
function trigger(ev) {
  if (!shared.known()) return
  const t = START / ev.seconds, n = Math.floor(t)
  if (shared.bar() * 16 + ev.step % 16 === shared.starts.Dance * 16 + n) fire(t - n)
}
