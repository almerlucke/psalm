package shaper

// ChainHelp is what to know to write a chain (blocks/shapechain), shown above its box
const ChainHelp = "a phase 0..1 goes in, a wave comes out; stages joined by `|`: `bipolar` is a saw, `sin` a sine, `mult 3 | sin` three cycles\n" +
	"`add` `mult` `mod` `ripple` `linear a b` `bipolar` `unipolar` `abs` `sin` `tri` `mirror lo hi` `pulse w` `quantize q` `clip lo hi` `softclip a` `cheby 1:1 3:0.3` `table sine` `pow p` `tanh d`\n" +
	"`par(chain, chain)` runs chains side by side and adds them: `par(sin, mult 3 | sin | mult 0.4)`\n" +
	"a number can be a macro `m1`..`m4` (0..1, the knobs of the sound): `mult m1[1,5]` is 1 to 5; `#` starts a comment\n" +
	"the morph goes from chain A through B, C, D (an empty chain ends the list)\n" +
	"Ctrl+Enter (or Run) takes it; one that can't be read is shown in red, the sound keeps the one before"
