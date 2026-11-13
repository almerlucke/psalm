// The editor of the chains of the shaper synth: each chain a rack of cards (one for every stage, knobs for their numbers, each number a
// constant or a macro with a range), plots (the wave, the curve of the stage you pick, the harmonics, the four chains along the morph),
// envelopes to drag with buttons for fades. The text of the chain is the truth: the rack and the text box are two views of one string.
psalm.register("shapeview", (p, api) => {
	const cat = p.data || {stages: [], keys: ["chainA", "chainB", "chainC", "chainD"]};
	const info = Object.fromEntries(cat.stages.map(s => [s.name, s]));
	const KEYS = cat.keys;
	let st = {chains: [], used: 0, used2: 0, values: {}};
	let ci = 0;                     // the chain being edited: 0 to 3 are the chains of line 1, 4 to 7 those of line 2
	let lineNo = 0;                   // the line being edited
	const sel = Array(8).fill(0);   // the stage picked in each chain
	const pending = Array(8).fill(0); // until when the page's own text is the one to show (ms)
	let rack = Array.from({length: 8}, () => []);    // the items of each chain: {text, off}
	let rackSig = Array(8).fill("");
	const colors = {};
	const restyle = () => { for (const n of ["ink", "accent", "muted", "line", "track"]) colors[n] = api.css(n); colors.danger = "#e5484d"; };
	restyle();

	const h = (tag, cls, text) => { const e = document.createElement(tag); if (cls) e.className = cls; if (text !== undefined) e.textContent = text; return e; };
	const send = msg => api.message(p.name, msg);
	const fmt = v => String(Math.round(v * 10000) / 10000);
	const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
	const later = (() => { const t = {}; return (k, ms, f) => { clearTimeout(t[k]); t[k] = setTimeout(f, ms); }; })();

	// ---------- reading and writing chains ----------
	const splitTop = s => { // cut at top level pipes
		const out = []; let depth = 0, cur = "";
		for (const ch of s) {
			if (ch === "(") depth++;
			if (ch === ")") depth = Math.max(0, depth - 1);
			if (ch === "|" && depth === 0) { out.push(cur); cur = ""; } else cur += ch;
		}
		out.push(cur);
		return out.map(x => x.replace(/\s+/g, " ").trim()).filter(Boolean);
	};
	const rackOf = text => { // bypassed stages are lines "#off | stage"
		const items = []; let buf = "";
		const flush = () => { for (const s of splitTop(buf)) items.push({text: s, off: false}); buf = ""; };
		for (const line of String(text).split("\n")) {
			const t = line.trim();
			if (t.startsWith("#off")) { flush(); const rest = t.slice(4).replace(/^\s*\|?\s*/, "").trim(); if (rest) items.push({text: rest, off: true}); continue; }
			buf += line.replace(/#.*$/, "") + "\n";
		}
		flush();
		return items;
	};
	const textOf = items => {
		if (items.length && items[0].off) items[0].off = false;
		return items.map((it, i) => i === 0 ? it.text : (it.off ? "#off | " : "| ") + it.text).join("\n");
	};
	const tokens = s => s.match(/m[1-4]\[[^\]]*\]|[^\s,]+/g) || [];
	const argOf = tok => {
		const m = /^m([1-4])(?:\[\s*(-?[\d.e+-]+)\s*,\s*(-?[\d.e+-]+)\s*\])?$/.exec(tok);
		if (m) return {macro: +m[1], lo: m[2] === undefined ? 0 : +m[2], hi: m[3] === undefined ? 1 : +m[3]};
		const v = parseFloat(tok);
		return {macro: 0, v: isNaN(v) ? 0 : v};
	};
	const tokOf = a => a.macro ? ("m" + a.macro + (a.lo === 0 && a.hi === 1 ? "" : "[" + fmt(a.lo) + "," + fmt(a.hi) + "]")) : fmt(a.v);
	const stageOf = text => { // {name, args: [..]} for a stage with numbers; null when it has none or is not plain
		const tk = tokens(text), name = tk[0], inf = info[name];
		if (!inf || inf.kind || !inf.params) return {name, inf, args: null, rest: tk.slice(1).join(" ")};
		const args = inf.params.map((pr, i) => tk[i + 1] !== undefined ? argOf(tk[i + 1]) : {macro: 0, v: pr.def});
		return {name, inf, args};
	};
	const stageText = (name, args) => name + (args.length ? " " + args.map(tokOf).join(" ") : "");

	const setChain = (i, items, structural) => {
		rack[i] = items; const text = textOf(items);
		pending[i] = performance.now() + 700;
		if (st.chains[i]) st.chains[i].text = text;
		rackSig[i] = items.map(x => (x.off ? "-" : "") + x.text).join("|");
		if (structural) buildRack();
		if (i === ci) textbox.value = text;
		later("text" + i, 80, () => send({text: {key: KEYS[i], value: text}}));
	};
	const setChainText = (i, text) => { rack[i] = rackOf(text); setChain(i, rack[i], i === ci); };

	// ---------- drawing ----------
	const prep = c => {
		const dpr = window.devicePixelRatio || 1, w = c.clientWidth, hh = c.clientHeight;
		if (!w || !hh) return null;
		if (c.width !== Math.round(w * dpr) || c.height !== Math.round(hh * dpr)) { c.width = Math.round(w * dpr); c.height = Math.round(hh * dpr); }
		const g = c.getContext("2d"); g.setTransform(dpr, 0, 0, dpr, 0, 0); g.clearRect(0, 0, w, hh);
		return {g, w, h: hh};
	};
	const line = (g, ys, w, hh, lo, hi, color, width, loop) => {
		if (!ys || !ys.length) return;
		g.beginPath(); g.strokeStyle = color; g.lineWidth = width; g.lineJoin = "round";
		const n = ys.length, pad = 3;
		for (let i = 0; i < n + (loop ? 1 : 0); i++) {
			const y = ys[i % n], x = pad + (w - 2 * pad) * (i / (loop ? n : n - 1));
			const yy = hh - pad - (hh - 2 * pad) * ((clamp(y, lo, hi) - lo) / (hi - lo));
			i ? g.lineTo(x, yy) : g.moveTo(x, yy);
		}
		g.stroke();
	};
	const grid = (g, w, hh, zero) => {
		g.strokeStyle = colors.line; g.lineWidth = 1; g.globalAlpha = 0.7;
		g.beginPath(); if (zero) { g.moveTo(0, hh / 2); g.lineTo(w, hh / 2); } g.moveTo(w / 2, 0); g.lineTo(w / 2, hh); g.stroke(); g.globalAlpha = 1;
	};
	const range = ys => { let a = Math.min(...ys), b = Math.max(...ys); if (b - a < 1e-6) { a -= 1; b += 1; } const m = Math.max(1, Math.abs(a), Math.abs(b)); return a >= 0 ? [Math.min(0, a), Math.max(1, b)] : [-m, m]; };
	const drawWave = (c, main, ghost, extra) => {
		const d = prep(c); if (!d) return; grid(d.g, d.w, d.h, true);
		const all = (main || []).concat(ghost || []); const [lo, hi] = all.length ? range(all) : [-1, 1];
		if (ghost) line(d.g, ghost, d.w, d.h, lo, hi, colors.muted, 1.5, true);
		if (extra) line(d.g, extra, d.w, d.h, lo, hi, colors.muted, 1, true);
		line(d.g, main, d.w, d.h, lo, hi, colors.accent, 2, true);
	};
	const drawCurve = (c, ys) => {
		const d = prep(c); if (!d) return; grid(d.g, d.w, d.h, true);
		if (!ys) return; const [lo, hi] = range(ys);
		d.g.globalAlpha = .5; d.g.strokeStyle = colors.muted; d.g.beginPath(); // y = x
		d.g.moveTo(3, d.h - 3 - (d.h - 6) * ((-1 - lo) / (hi - lo))); d.g.lineTo(d.w - 3, d.h - 3 - (d.h - 6) * ((1 - lo) / (hi - lo))); d.g.stroke(); d.g.globalAlpha = 1;
		line(d.g, ys, d.w, d.h, lo, hi, colors.accent, 2, false);
	};
	const drawSpectrum = (c, sp, harm) => {
		const d = prep(c); if (!d) return; if (!sp || !sp.length) return;
		const n = sp.length, bw = (d.w - 4) / n;
		sp.forEach((v, i) => {
			const db = v > 1e-6 ? 20 * Math.log10(v) : -90, f = clamp((db + 60) / 60, 0, 1);
			d.g.fillStyle = harm && i + 1 > harm && harm < 1000 ? colors.muted : colors.accent;
			d.g.fillRect(2 + i * bw, d.h - 2 - f * (d.h - 6), Math.max(1, bw - 1), Math.max(1, f * (d.h - 6)));
		});
	};

	// ---------- the root ----------
	const root = h("div", "shapeview");
	const bar = h("div", "sv-bar"), tabs = h("div", "sv-tabs");
	const lineEls = [0, 1].map(n => { const b = h("button", "sv-tab sv-line", "Line " + (n + 1)); b.title = n ? "line 2: its own chains, or line 1's when it has none" : "line 1"; b.addEventListener("click", () => { lineNo = n; pick(n * 4 + ci % 4); }); tabs.append(b); return b; });
	tabs.append(h("span", "sv-gap"));
	const tabEls = [0, 1, 2, 3].map(i => { const b = h("button", "sv-tab", "ABCD"[i]); b.title = "edit chain " + "ABCD"[i]; b.addEventListener("click", () => pick(lineNo * 4 + i)); tabs.append(b); return b; });
	const playBtn = h("button", "sv-btn", "▶ Play"); playBtn.title = "play a note on the synth, to hear the chains as they are";
	const octave = h("select", "sv-select"); [36, 48, 60, 72].forEach(n => { const o = h("option", "", "C" + (n / 12 - 1)); o.value = n; if (n === 48) o.selected = true; octave.append(o); });
	const own = h("div", "sv-own");
	const ownMsg = h("span", "", "Line 2 reads the chains of line 1. "), ownBtn = h("button", "sv-btn", "Give it chains of its own");
	ownBtn.title = "copy the chains of line 1 to line 2, to change them there";
	ownBtn.addEventListener("click", () => { for (let j = 0; j < 4; j++) { const t = st.chains[j] ? st.chains[j].text : ""; setChainText(4 + j, t); } pick(4 + ci % 4); });
	const ownOff = h("button", "sv-btn", "Same as line 1 again"); ownOff.title = "line 2 reads the chains of line 1 again (its own are cleared)";
	ownOff.addEventListener("click", () => { for (let j = 0; j < 4; j++) setChainText(4 + j, ""); });
	own.append(ownMsg, ownBtn, ownOff);
	playBtn.addEventListener("click", () => send({play: +octave.value + 9}));
	const startSel = h("select", "sv-select"); startSel.title = "replace this chain by a start";
	const starts = [["Sine", "sin"], ["Saw", "bipolar"], ["Triangle", "tri | bipolar"], ["Square", "pulse 0.5 | bipolar"], ["Soft saw", "bipolar | tanh 1.5"], ["Folded sine", "sin | fold 3"],
		["FM pair", "par(thru, mult 2 | sin | mult 0.3) | sin"], ["Additive", "par(sin, mult 2 | sin | mult 0.5, mult 3 | sin | mult 0.33)"], ["Hard sync", "mult 2.5 | mod 1 | bipolar"], ["CZ resonant", "czres 0.6 4"]];
	startSel.append(h("option", "", "Start from…")); starts.forEach(([n, t]) => { const o = h("option", "", n); o.value = t; startSel.append(o); });
	startSel.addEventListener("change", () => { if (startSel.value) setChainText(ci, startSel.value); startSel.selectedIndex = 0; });
	const freeze = h("button", "sv-btn", "❄ Freeze"); freeze.title = "turn the wave of this chain into a table of points: sculpt, then keep it";
	freeze.addEventListener("click", () => {
		const w = st.chains[ci] && st.chains[ci].wave; if (!w) return;
		const pts = []; for (let i = 0; i < 128; i++) pts.push(fmt(w[i * 2])); pts.push(fmt(w[0]));
		setChainText(ci, "table " + pts.join(" "));
	});
	const vary = h("button", "sv-btn", "≋ Vary in B–D"); vary.title = "copy this chain to B, C and D, the first number of the picked stage moving a third, two thirds and all the way to the end of its range: a morph made from one chain";
	vary.addEventListener("click", () => {
		const items = rack[ci]; const it = items[sel[ci]]; if (!it) return;
		const s = stageOf(it.text); if (!s.args || !s.args.length) return;
		const pr = s.inf.params[0], cur = s.args[0].macro ? (pr.lo + pr.hi) / 2 : s.args[0].v;
		const end = cur - pr.lo > pr.hi - cur ? pr.lo : pr.hi;
		const base = ci % 4; let k = 0;
		for (let jj = 0; jj < 4; jj++) {
			if (jj === base) continue; k++; const j = lineNo * 4 + jj;
			const copy = items.map(x => ({...x}));
			const a = s.args.map(x => ({...x})); a[0] = {macro: 0, v: cur + (end - cur) * (k / 3)};
			copy[sel[ci]].text = stageText(s.name, a);
			rack[j] = copy; setChain(j, copy, false);
		}
		buildRack();
	});
	bar.append(tabs, playBtn, octave, startSel, freeze, vary);

	const plots = h("div", "sv-plots");
	const cWave = h("canvas", "sv-canvas wave"), cCurve = h("canvas", "sv-canvas curve"), cSpec = h("canvas", "sv-canvas spec");
	const capWave = h("div", "sv-cap"); const wavePlot = h("div", "sv-plot"); wavePlot.append(cWave, capWave);
	const capCurve = h("div", "sv-cap"); const curvePlot = h("div", "sv-plot"); curvePlot.append(cCurve, capCurve);
	const capSpec = h("div", "sv-cap"); const specPlot = h("div", "sv-plot"); specPlot.append(cSpec, capSpec);
	plots.append(wavePlot, curvePlot, specPlot);

	const rackEl = h("div", "sv-rack"), addRow = h("div", "sv-add");
	const addSel = h("select", "sv-select"); addSel.append(h("option", "", "+ Add a stage…"));
	for (const g of [...new Set(cat.stages.map(s => s.group))]) {
		const og = h("optgroup"); og.label = g;
		for (const s of cat.stages.filter(s => s.group === g)) { const o = h("option", "", s.name + " – " + s.help); o.value = s.name; og.append(o); }
		addSel.append(og);
	}
	addSel.addEventListener("change", () => {
		const inf = info[addSel.value]; addSel.selectedIndex = 0; if (!inf) return;
		const text = inf.example || stageText(inf.name, (inf.params || []).map(pr => ({macro: 0, v: pr.def})));
		const items = rack[ci].map(x => ({...x})); const at = Math.min(items.length, sel[ci] + 1 || items.length);
		items.splice(items.length ? at : 0, 0, {text, off: false}); sel[ci] = items.length ? Math.min(at, items.length - 1) : 0;
		setChain(ci, items, true);
	});
	addRow.append(addSel);

	const textbox = h("textarea", "sv-text"); textbox.rows = 3; textbox.spellcheck = false;
	textbox.placeholder = "the chain as text: bipolar | fold m1[1,6]";
	const err = h("div", "sv-err");
	textbox.addEventListener("input", () => { pending[ci] = performance.now() + 1500; later("typing", 350, () => { rack[ci] = rackOf(textbox.value); rackSig[ci] = rack[ci].map(x => (x.off ? "-" : "") + x.text).join("|"); send({text: {key: KEYS[ci], value: textbox.value}}); buildRack(); }); });

	// the morph and macros
	const morphBox = h("div", "sv-morph"), thumbs = [0, 1, 2, 3].map(i => { const c = h("canvas", "sv-canvas thumb"); c.title = "chain " + "ABCD"[i]; c.addEventListener("click", () => pick(lineNo * 4 + i)); return c; });
	const thumbRow = h("div", "sv-thumbs"); thumbs.forEach(c => thumbRow.append(c));
	const nowC = h("canvas", "sv-canvas thumb now"); nowC.title = "the wave now: the chains blended at the morph";
	const sliders = {};
	const slider = (key, label, lo, hi, step, hint) => {
		const row = h("label", "sv-slider"); const nm = h("span", "nm", label); if (hint) row.title = hint;
		const r = document.createElement("input"); r.type = "range"; r.min = lo; r.max = hi; r.step = step;
		const v = h("span", "val"); let drag = false;
		r.addEventListener("input", () => { drag = true; v.textContent = fmt(+r.value); send({set: {[key]: +r.value}}); });
		r.addEventListener("change", () => { drag = false; });
		row.append(nm, r, v); sliders[key] = {r, v, row, set(x) { if (!drag && document.activeElement !== r) { r.value = x; v.textContent = fmt(x); } }};
		return row;
	};
	const morphRow = slider("morph", "morph", 0, 1, 0.005, "0 is chain A, 1 the last chain; between them a blend");
	const macroRows = [1, 2, 3, 4].map(n => slider("macro" + n, "m" + n, 0, 1, 0.005, "macro " + n + ": a number in the chains that the synth moves while it plays"));
	const amtRows = [slider("morphAmount", "morph env", -1, 1, 0.01, "how far the morph envelope pulls the morph at its peak"), slider("shapeAmount", "m1 env", -1, 1, 0.01, "how far the m1 envelope pulls macro 1 at its peak")];
	const stereoRows = [slider("stereoMorph", "stereo morph", -1, 1, 0.01, "the left channel reads the morph that much under and the right that much over: a centred sound gets wide")]
		.concat([1, 2, 3, 4].map(n => slider("stereoMacro" + n, "stereo m" + n, -1, 1, 0.01, "macro " + n + " is lower on the left and higher on the right by this much (a fold deeper on one side)")));
	const nowRow = h("div", "sv-nowrow"); nowRow.append(nowC, h("span", "sv-cap", "now"));
	morphBox.append(thumbRow, nowRow, morphRow, h("div", "sv-sub", "macros (the cards that use them show m1 to m4)"), ...macroRows, ...amtRows, h("div", "sv-sub", "stereo: the left and the right read other places (automatic oversampling)"), ...stereoRows);

	// ---------- envelopes ----------
	const ENV = [["amp", "amp", "volume over the note"], ["morphEnv", "morph", "moves the morph over the note (see morph env)"], ["shapeEnv", "m1", "moves macro 1 over the note (see m1 env)"], ["pitchEnv", "pitch", "bends the pitch over the note (see pitch env amount)"]];
	const fades = {
		amp: [["Pluck", 3, 250, 0.1, 250], ["Soft", 40, 400, 0.8, 400], ["Fade in", 1500, 500, 0.9, 600], ["Swell", 2500, 1000, 0.8, 2500], ["Fade out", 20, 300, 0.85, 3000]],
		other: [["Quick", 5, 300, 0.2, 200], ["Slow rise", 2500, 3000, 0.5, 1500], ["Fall", 2, 800, 0, 300], ["Hold", 5, 100, 1, 300]],
	};
	const lane = [0, 0.25, 0.5, 0.65, 1], TMAX = 10000;
	const tx = d => Math.log(1 + clamp(d, 0, TMAX) / 8) / Math.log(1 + TMAX / 8), dx = u => 8 * (Math.exp(clamp(u, 0, 1) * Math.log(1 + TMAX / 8)) - 1);
	const envBox = h("div", "sv-envs"), envs = ENV.map(([prefix, label, hint]) => {
		const box = h("div", "sv-env"), cap = h("div", "sv-cap", label); cap.title = hint;
		const c = h("canvas", "sv-canvas envc"), btns = h("div", "sv-fades");
		const key = s => (prefix === "amp" ? "amp" : prefix) + s;
		const get = () => { const v = st.values || {}; return {a: v[key("Attack")] ?? 5, d: v[key("Decay")] ?? 200, s: v[key("Sustain")] ?? 0.8, r: v[key("Release")] ?? 300}; };
		const put = o => { const set = {}; for (const [k, s] of [["a", "Attack"], ["d", "Decay"], ["s", "Sustain"], ["r", "Release"]]) if (o[k] !== undefined) set[key(s)] = o[k]; send({set}); Object.assign(st.values, Object.fromEntries(Object.entries(set))); };
		for (const [n, a, d, s, r] of fades[prefix === "amp" ? "amp" : "other"]) {
			const b = h("button", "sv-mini", n); b.addEventListener("click", () => put({a, d, s, r})); btns.append(b);
		}
		const geom = () => {
			const d = prep(c); if (!d) return null; const e = get(), W = d.w, H = d.h, top = 6, bot = H - 6, ys = v => bot - (bot - top) * v;
			const pts = [[0, 0], [W * (lane[0] + (lane[1] - lane[0]) * tx(e.a)), 1], [W * (lane[1] + (lane[2] - lane[1]) * tx(e.d)), e.s], [W * lane[3], e.s], [W * (lane[3] + (lane[4] - lane[3]) * tx(e.r)), 0]];
			return {d, e, W, H, ys, pts: pts.map(([x, v]) => [x, ys(v)])};
		};
		const draw = () => {
			const g0 = geom(); if (!g0) return; const {d, pts} = g0, g = d.g;
			g.strokeStyle = colors.line; g.lineWidth = 1; g.globalAlpha = .6; g.beginPath(); for (const l of lane.slice(1, 4)) { g.moveTo(l * d.w, 0); g.lineTo(l * d.w, d.h); } g.stroke(); g.globalAlpha = 1;
			g.beginPath(); g.moveTo(pts[0][0], pts[0][1]); pts.slice(1).forEach(q => g.lineTo(q[0], q[1])); g.lineTo(pts[4][0], d.h - 6);
			g.globalAlpha = .18; g.fillStyle = colors.accent; g.fill(); g.globalAlpha = 1;
			g.beginPath(); pts.forEach((q, i) => i ? g.lineTo(q[0], q[1]) : g.moveTo(q[0], q[1])); g.strokeStyle = colors.accent; g.lineWidth = 2; g.stroke();
			for (const i of [1, 2, 4]) { g.beginPath(); g.arc(pts[i][0], pts[i][1], 5, 0, 7); g.fillStyle = colors.accent; g.fill(); }
			g.fillStyle = colors.muted; g.font = "11px sans-serif";
			const e = g0.e; g.fillText("A " + Math.round(e.a) + "ms", 4, 12); g.fillText("D " + Math.round(e.d), lane[1] * d.w + 4, 12); g.fillText("S " + fmt(e.s), lane[2] * d.w + 4, 12); g.fillText("R " + Math.round(e.r), lane[3] * d.w + 4, 12);
		};
		let grab = -1;
		c.addEventListener("pointerdown", ev => {
			const g0 = geom(); if (!g0) return; const r = c.getBoundingClientRect(), x = ev.clientX - r.left, y = ev.clientY - r.top;
			let best = -1, bd = 22; for (const i of [1, 2, 4]) { const dd = Math.hypot(g0.pts[i][0] - x, g0.pts[i][1] - y); if (dd < bd) { bd = dd; best = i; } }
			if (best < 0) return; grab = best; c.setPointerCapture(ev.pointerId); ev.preventDefault();
		});
		c.addEventListener("pointermove", ev => {
			if (grab < 0) return; const g0 = geom(); if (!g0) return; const r = c.getBoundingClientRect(), x = (ev.clientX - r.left) / r.width, y = ev.clientY - r.top;
			const v = clamp((g0.d.h - 6 - y) / (g0.d.h - 12), 0, 1), lu = (i, from, to) => dx((x - lane[from]) / (lane[to] - lane[from]));
			if (grab === 1) put({a: Math.max(1, lu(0, 0, 1))});
			if (grab === 2) put({d: Math.max(1, lu(0, 1, 2)), s: v});
			if (grab === 4) put({r: Math.max(1, lu(0, 3, 4))});
		});
		const up = () => { grab = -1; }; c.addEventListener("pointerup", up); c.addEventListener("pointercancel", up);
		box.append(cap, c, btns); envBox.append(box); return {draw};
	});

	// ---------- the rack ----------
	const cardPlots = []; // [{wave, curve, index}] to redraw
	const buildRack = () => {
		rackEl.textContent = ""; cardPlots.length = 0; let k = -1;
		const items = rack[ci];
		items.forEach((it, i) => {
			const s = stageOf(it.text), card = h("div", "sv-card" + (i === sel[ci] ? " sel" : "") + (it.off ? " off" : ""));
			card.dataset.i = i;
			if (!it.off) k++;
			const head = h("div", "sv-head"), grip = h("span", "sv-grip", "⠿"), name = h("b", "sv-name", s.name);
			grip.draggable = true; grip.title = "drag to move this stage"; // only the grip drags the card: a slider or a number moves without picking the card up name.title = s.inf ? s.inf.help : "";
			const tools = h("span", "sv-tools");
			const mk = (t, title, f) => { const b = h("button", "sv-x", t); b.title = title; b.addEventListener("click", e => { e.stopPropagation(); f(); }); tools.append(b); return b; };
			mk(it.off ? "○" : "●", "bypass: leave this stage out", () => { if (i === 0) return; const c = rack[ci].map(x => ({...x})); c[i].off = !c[i].off; setChain(ci, c, true); });
			mk("⧉", "copy this stage after itself", () => { const c = rack[ci].map(x => ({...x})); c.splice(i + 1, 0, {...c[i]}); sel[ci] = i + 1; setChain(ci, c, true); });
			mk("✕", "take this stage away", () => { const c = rack[ci].map(x => ({...x})); c.splice(i, 1); sel[ci] = Math.max(0, Math.min(sel[ci], c.length - 1)); setChain(ci, c, true); });
			head.append(grip, name, tools);
			const mini = h("canvas", "sv-canvas mini"); card.append(head, mini);
			const body = h("div", "sv-args");
			const edit = f => { const c = rack[ci].map(x => ({...x})); c[i].text = f(); rack[ci] = c; setChain(ci, c, false); };
			if (s.args) {
				s.inf.params.forEach((pr, j) => {
					const a = s.args[j], row = h("div", "sv-arg"), nm = h("span", "nm", pr.name);
					const num = document.createElement("input"); num.type = "number"; num.step = (pr.hi - pr.lo) / 200; num.className = "num";
					const rg = document.createElement("input"); rg.type = "range"; rg.min = pr.lo; rg.max = pr.hi; rg.step = (pr.hi - pr.lo) / 400; rg.className = "rg";
					const mc = h("select", "mc"); ["·", "m1", "m2", "m3", "m4"].forEach((t, q) => { const o = h("option", "", t); o.value = q; mc.append(o); }); mc.value = a.macro; mc.title = "a constant, or a macro the synth moves";
					const lo = document.createElement("input"), hi = document.createElement("input"); for (const e of [lo, hi]) { e.type = "number"; e.className = "num lohi"; e.step = (pr.hi - pr.lo) / 200; }
					lo.title = "the number when the macro is 0"; hi.title = "the number when the macro is 1";
					const show = () => { num.value = a.macro ? "" : fmt(a.v); rg.value = a.macro ? pr.def : a.v; lo.value = fmt(a.lo ?? 0); hi.value = fmt(a.hi ?? 1); num.hidden = rg.hidden = !!a.macro; lo.hidden = hi.hidden = !a.macro; };
					const cur = () => { const q = stageOf(rack[ci][i].text); return q.args; };
					const commit = () => edit(() => { const args = cur().map(x => ({...x})); args[j] = {...a}; return stageText(s.name, args); });
					rg.addEventListener("input", () => { a.v = +rg.value; num.value = fmt(a.v); commit(); });
					num.addEventListener("change", () => { a.v = +num.value; rg.value = a.v; commit(); });
					mc.addEventListener("change", () => { a.macro = +mc.value; if (a.macro && a.lo === undefined) { a.lo = pr.lo; a.hi = pr.hi; } show(); commit(); });
					for (const e of [lo, hi]) e.addEventListener("change", () => { a.lo = +lo.value; a.hi = +hi.value; commit(); });
					show(); row.append(nm, mc, rg, num, lo, hi); body.append(row);
				});
			} else if (s.inf && (s.inf.kind === "list" || s.inf.kind === "word" || s.inf.kind === "par")) {
				const t = document.createElement("input"); t.className = "txt"; t.value = it.text.slice(s.name.length).trim(); t.spellcheck = false;
				t.title = s.inf.help; t.addEventListener("change", () => edit(() => (s.name + " " + t.value).trim()));
				body.append(t);
			} else if (!s.inf) {
				const t = document.createElement("input"); t.className = "txt"; t.value = it.text; t.spellcheck = false; t.title = "a stage the editor does not know: written as text";
				t.addEventListener("change", () => edit(() => t.value.trim() || it.text)); body.append(t);
			}
			card.append(body);
			card.addEventListener("click", () => { sel[ci] = i; [...rackEl.children].forEach((c, q) => c.classList && c.classList.contains("sv-card") && c.classList.toggle("sel", q === i)); redraw(); });
			grip.addEventListener("dragstart", e => { e.dataTransfer.setData("text/plain", String(i)); e.dataTransfer.effectAllowed = "move"; e.dataTransfer.setDragImage(card, 12, 12); card.classList.add("drag"); });
			grip.addEventListener("dragend", () => card.classList.remove("drag"));
			card.addEventListener("dragover", e => { e.preventDefault(); card.classList.add("over"); });
			card.addEventListener("dragleave", () => card.classList.remove("over"));
			card.addEventListener("drop", e => {
				e.preventDefault(); card.classList.remove("over"); const from = +e.dataTransfer.getData("text/plain"); if (isNaN(from) || from === i) return;
				const c = rack[ci].map(x => ({...x})); const [m] = c.splice(from, 1); c.splice(i, 0, m); sel[ci] = i; setChain(ci, c, true);
			});
			cardPlots.push({mini, index: it.off ? -1 : k, i});
			rackEl.append(card);
		});
		rackEl.append(addRow);
		redraw();
	};

	const showOwn = () => { // the note that line 2 reads line 1's chains, and its buttons
		const own2 = (st.used2 || 0) > 0;
		own.hidden = !(lineNo === 1 || own2); ownMsg.hidden = ownBtn.hidden = own2; ownOff.hidden = !own2;
	};
	const pick = i => {
		ci = i; lineNo = i >= 4 ? 1 : 0;
		tabEls.forEach((b, q) => b.classList.toggle("on", q === i % 4)); lineEls.forEach((b, q) => b.classList.toggle("on", q === lineNo));
		if (!rack[i].length && st.chains[i]) rack[i] = rackOf(st.chains[i].text);
		textbox.value = st.chains[i] ? st.chains[i].text : ""; buildRack(); redraw();
		showOwn();
	};

	const redraw = () => {
		const c = st.chains[ci]; if (!c) return;
		const items = rack[ci], it = items[sel[ci]];
		const pl = cardPlots.find(q => q.i === sel[ci]); const sp = pl && pl.index >= 0 ? (c.stages || [])[pl.index] : null;
		drawWave(cWave, c.wave, sp && sp.wave && sp.wave.length ? upsample(sp.wave) : null);
		drawCurve(cCurve, sp ? sp.curve : null);
		drawSpectrum(cSpec, c.spectrum, c.harm);
		capWave.textContent = "wave of " + (lineNo ? "line 2 " : "") + "chain " + "ABCD"[ci % 4] + (sp ? " (grey: after " + ((it && it.text) || "").split(" ")[0] + ")" : "");
		capCurve.textContent = sp ? "the stage " + ((it && it.text) || "").split(" ")[0] + ", x from -1 to 1" : "pick a stage";
		capSpec.textContent = "harmonics" + (c.harm ? (c.harm >= 1000 ? ": an edge, they go on (aliasing is filtered)" : ": " + c.harm + " matter") : "");
		for (const q of cardPlots) { const s2 = q.index >= 0 ? (c.stages || [])[q.index] : null; drawWave(q.mini, s2 && s2.wave ? s2.wave : null, null); }
		thumbs.forEach((t, i) => { const ch = st.chains[lineNo * 4 + i]; drawWave(t, ch && ch.wave, null); t.classList.toggle("on", lineNo * 4 + i === ci); t.classList.toggle("unused", !ch || !ch.wave); });
		drawWave(nowC, lineNo ? st.blend2 : st.blend, null);
		envs.forEach(e => e.draw());
	};
	const upsample = w => { const n = 256, out = new Array(n); for (let i = 0; i < n; i++) out[i] = w[Math.floor(i * w.length / n)]; return out; };

	root.append(bar, own, plots, rackEl, textbox, err, h("div", "sv-sub", "morph: the chains A to D blended along the morph"), morphBox, h("div", "sv-sub", "envelopes: drag the dots; the buttons set fades"), envBox);
	tabEls[0].classList.add("on"); lineEls[0].classList.add("on"); own.hidden = true;

	return {
		el: root,
		show() {},
		update(s) {
			if (!s || !s.chains) return;
			st = s; const now = performance.now();
			s.chains.forEach((c, i) => {
				const sig = rackOf(c.text).map(x => (x.off ? "-" : "") + x.text).join("|");
				if (now < pending[i]) { c.text = rack[i].length || sig === "" ? textOf(rack[i]) : c.text; return; }
				if (sig !== rackSig[i] && !(i === ci && document.activeElement === textbox)) { rack[i] = rackOf(c.text); rackSig[i] = rack[i].map(x => (x.off ? "-" : "") + x.text).join("|"); if (i === ci) { textbox.value = c.text; buildRack(); } }
			});
			const c = s.chains[ci]; err.textContent = c && c.err ? c.err : ""; err.hidden = !(c && c.err);
			tabEls.forEach((b, i) => { const ch = s.chains[lineNo * 4 + i]; b.classList.toggle("bad", !!(ch && ch.err)); b.classList.toggle("used", !!(ch && ch.wave)); });
			lineEls.forEach((b, n) => b.classList.toggle("bad", s.chains.slice(n * 4, n * 4 + 4).some(ch => ch && ch.err)));
			showOwn();
			lineEls[1].classList.toggle("used", s.used2 > 0);
			for (const [k, sl] of Object.entries(sliders)) if (s.values && s.values[k] !== undefined) sl.set(s.values[k]);
			// the macros that are used
			const used = [0, 1, 2, 3].map(m => s.chains.some(ch => ch && ch.macros && ch.macros[m]));
			macroRows.forEach((r, m) => r.classList.toggle("unused", !used[m]));
			redraw();
		},
		restyle() { restyle(); redraw(); },
	};
});
