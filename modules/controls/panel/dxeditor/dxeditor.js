// the editor of the six operator FM synth: the algorithm (one of the 32, or one of your own), the
// operators, the LFO and the pitch envelope
psalm.register("dx7", (p, api) => {
	"use strict";
	const OPS = 6;
	const part = {}; // setting -> its control
	for (const pt of p.parts) part[pt.name.slice(p.name.length + 1)] = pt;
	const vals = {algorithm: p.value};
	for (const [k, pt] of Object.entries(part)) vals[k] = pt.value;
	const binds = {}; // setting -> what shows it
	const bind = (key, f) => { (binds[key] = binds[key] || []).push(f); f(vals[key]); };
	const render = key => { for (const f of binds[key] || []) f(vals[key]); if (routingKey(key)) drawAlgorithm(); };
	const routingKey = k => k === "algorithm" || k === "custom" || /^(carrier|mod|fb)/.test(k) || /^op\dLevel$/.test(k);
	const set = (key, v) => {
		const pt = key === "algorithm" ? p : part[key];
		if (!pt) return;
		v = Math.max(pt.min, Math.min(pt.max, Math.round(v)));
		if (vals[key] === v) return;
		vals[key] = v;
		render(key);
		api.send(pt.name, v);
	};
	const el = (tag, cls, text) => { const e = document.createElement(tag); if (cls) e.className = cls; if (text !== undefined) e.textContent = text; return e; };

	// a number to drag up and down (Shift: slower), turn with the wheel, step with the arrow keys,
	// double click for its default
	function num(key, fmt = v => String(v), title = "") {
		const e = el("span", "dx-num");
		e.tabIndex = 0; e.title = title || key;
		const pt = key === "algorithm" ? p : part[key];
		let startY = 0, startV = 0, dragging = false;
		e.addEventListener("pointerdown", ev => { dragging = true; startY = ev.clientY; startV = vals[key]; e.setPointerCapture(ev.pointerId); ev.preventDefault(); });
		e.addEventListener("pointermove", ev => { if (dragging) set(key, startV + Math.round((startY - ev.clientY) / (ev.shiftKey ? 14 : 4))); });
		e.addEventListener("pointerup", () => { dragging = false; });
		e.addEventListener("pointercancel", () => { dragging = false; });
		e.addEventListener("wheel", ev => { ev.preventDefault(); set(key, vals[key] + (ev.deltaY < 0 ? 1 : -1)); }, {passive: false});
		e.addEventListener("dblclick", () => set(key, pt.default));
		e.addEventListener("keydown", ev => {
			const d = {ArrowUp: 1, ArrowRight: 1, ArrowDown: -1, ArrowLeft: -1}[ev.key];
			if (d) { ev.preventDefault(); set(key, vals[key] + d * (ev.shiftKey ? 10 : 1)); }
		});
		bind(key, v => { e.textContent = fmt(v); });
		return e;
	}
	// a switch of a setting that is on or off
	function flag(key, label, title = "") {
		const b = el("button", "dx-flag", label);
		b.title = title || key;
		b.addEventListener("click", () => set(key, vals[key] ? 0 : 1));
		bind(key, v => b.classList.toggle("on", v > 0));
		return b;
	}
	const labelled = (label, ...kids) => { const b = el("label", "dx-field"); b.append(el("span", "dx-label", label), ...kids); return b; };

	const waveNames = ["sine", "half sine", "abs sine", "pulse sine", "even sine", "even abs", "square", "log saw"];
	const root = el("div", "dx");
	const envCanvases = []; // [operator, the small envelope, the big one]

	// --- the algorithm ---
	const top = el("div", "dx-top");
	const svgNS = "http://www.w3.org/2000/svg";
	const svg = document.createElementNS(svgNS, "svg");
	svg.setAttribute("class", "dx-alg");
	svg.setAttribute("viewBox", "0 0 240 190");
	const head = el("div", "dx-head");
	const name = el("div", "dx-name", "");
	const prev = el("button", "dx-step", "◀"), next = el("button", "dx-step", "▶");
	prev.title = "the algorithm before"; next.title = "the next algorithm";
	prev.addEventListener("click", () => set("algorithm", vals.algorithm - 1 < 1 ? 32 : vals.algorithm - 1));
	next.addEventListener("click", () => set("algorithm", vals.algorithm + 1 > 32 ? 1 : vals.algorithm + 1));
	const algNum = num("algorithm", v => String(v), "the algorithm, 1 to 32");
	const own = el("button", "dx-flag", "my own");
	own.title = "an algorithm of your own: starts as the one shown, then click operators to make them carriers and set the modulations below";
	own.addEventListener("click", () => {
		if (vals.custom) { set("custom", 0); return; }
		// start from the algorithm shown
		const a = p.data.algorithms[vals.algorithm - 1];
		for (let op = 1; op <= OPS; op++) set("carrier" + op, a.carriers.includes(op) ? 1 : 0);
		for (let f = 2; f <= OPS; f++) for (let t = 1; t < f; t++) set(`mod${f}${t}`, a.mods.some(m => m[0] === f && m[1] === t) ? 1 : 0);
		set("fbFrom", a.feedback[0]); set("fbTo", a.feedback[1]);
		set("custom", 1);
	});
	bind("custom", v => { own.classList.toggle("on", v > 0); root.classList.toggle("custom", v > 0); });
	const algRow = el("div", "dx-row");
	algRow.append(el("span", "dx-label", "algorithm"), prev, algNum, next, own);
	const fbRow = el("div", "dx-row");
	fbRow.append(labelled("feedback", num("feedback", v => String(v), "feedback, 0 to 7")));
	head.append(name, algRow, fbRow);
	if (p.data.bank && p.data.bank.length) {
		const sel = el("select", "dx-bank");
		p.data.bank.forEach((n, i) => { const o = el("option", "", `${i + 1} ${n}`); o.value = i; sel.append(o); });
		sel.addEventListener("change", () => api.send(part.program.name, Number(sel.value)));
		bind("program", v => { sel.value = String(v); });
		head.append(labelled("patch", sel));
	}

	// your own: the modulations a higher operator can make into a lower one, the feedback loop
	const routing = el("div", "dx-routing");
	const grid = el("div", "dx-matrix");
	grid.append(el("span", "dx-corner", "mod ↓ into →"));
	for (let t = 1; t < OPS; t++) grid.append(el("span", "dx-head-cell", String(t)));
	for (let f = OPS; f >= 2; f--) {
		grid.append(el("span", "dx-head-cell", String(f)));
		for (let t = 1; t < OPS; t++) {
			if (t >= f) { grid.append(el("span", "dx-off")); continue; }
			const c = el("button", "dx-cell");
			c.title = `operator ${f} modulates operator ${t}`;
			c.addEventListener("click", () => set(`mod${f}${t}`, vals[`mod${f}${t}`] ? 0 : 1));
			bind(`mod${f}${t}`, v => c.classList.toggle("on", v > 0));
			grid.append(c);
		}
	}
	const fb = el("div", "dx-row");
	fb.append(labelled("feedback from", num("fbFrom", v => "op " + v)), labelled("into", num("fbTo", v => "op " + v)));
	routing.append(grid, fb, el("div", "dx-hint", "click an operator in the diagram to make it a carrier (heard) or not"));
	// patches of your own: saved under a name, loaded back, deleted
	const mineRow = el("div", "dx-row dx-mine");
	const mineSel = el("select", "dx-bank");
	const mineName = el("input", "dx-name-in"); mineName.placeholder = "name"; mineName.maxLength = 64;
	const mineSave = el("button", "dx-flag", "save"), mineDel = el("button", "dx-flag", "delete");
	mineSave.title = "save the patch as it is under the name"; mineDel.title = "delete this patch of yours";
	const mineNote = el("span", "dx-hint");
	mineRow.append(el("span", "dx-label", "mine"), mineSel, mineName, mineSave, mineDel, mineNote);
	head.append(mineRow);
	let mine = {mine: [], keeps: false, current: ""}, mineShown = "";
	const showMine = () => {
		mineRow.hidden = !mine.keeps;
		const key = JSON.stringify(mine.mine || []);
		if (key !== mineShown) {
			mineShown = key;
			mineSel.innerHTML = "";
			const none = el("option", "", "your patches…"); none.value = ""; mineSel.append(none);
			for (const n of mine.mine || []) { const o = el("option", "", n); o.value = n; mineSel.append(o); }
		}
		const isMine = (mine.mine || []).includes(mine.current);
		mineSel.value = isMine ? mine.current : "";
		mineDel.disabled = !isMine;
		if (isMine && document.activeElement !== mineName) mineName.value = mine.current;
		mineNote.textContent = mine.note || "";
	};
	mineSel.addEventListener("change", () => { if (mineSel.value) api.message(p.name, {presets: {load: mineSel.value, mine: true}}); });
	const saveMine = () => { const n = mineName.value.trim(); if (n) api.message(p.name, {presets: {save: n}}); };
	mineSave.addEventListener("click", saveMine);
	mineName.addEventListener("keydown", ev => { if (ev.key === "Enter") saveMine(); });
	mineDel.addEventListener("click", () => { if (mineSel.value && confirm(`Delete "${mineSel.value}"?`)) api.message(p.name, {presets: {delete: mineSel.value}}); });
	showMine();
	top.append(svg, head, routing);
	root.append(top);

	// the algorithm playing: one of the 32, or your own as set
	const playing = () => {
		if (!vals.custom) return p.data.algorithms[vals.algorithm - 1];
		const a = {carriers: [], mods: [], feedback: [vals.fbFrom, vals.fbTo]};
		for (let op = 1; op <= OPS; op++) if (vals["carrier" + op]) a.carriers.push(op);
		for (let f = 2; f <= OPS; f++) for (let t = 1; t < f; t++) if (vals[`mod${f}${t}`]) a.mods.push([f, t]);
		return a;
	};
	function drawAlgorithm() {
		const a = playing();
		svg.textContent = "";
		// rows: the carriers at the bottom, each modulator a row above the highest operator it modulates
		const row = {};
		const rowOf = (op, seen = new Set()) => {
			if (row[op] !== undefined) return row[op];
			if (seen.has(op)) return 0;
			seen.add(op);
			const targets = a.mods.filter(m => m[0] === op).map(m => m[1]);
			let r = a.carriers.includes(op) ? 0 : -1;
			for (const t of targets) r = Math.max(r, rowOf(t, seen) + 1);
			return (row[op] = r);
		};
		for (let op = 1; op <= OPS; op++) rowOf(op);
		const unused = [];
		for (let op = 1; op <= OPS; op++) if (row[op] < 0) unused.push(op);
		// columns: by the carrier each modulator ends in, in order
		const order = [];
		const visit = op => { if (order.includes(op)) return; order.push(op); for (const m of a.mods.filter(m => m[1] === op).map(m => m[0]).sort((x, y) => x - y)) visit(m); };
		for (const c of [...a.carriers].sort((x, y) => x - y)) visit(c);
		const rows = Math.max(1, ...Object.values(row).map(r => r + 1));
		const byRow = {};
		for (const op of order) (byRow[row[op]] = byRow[row[op]] || []).push(op);
		const pos = {}, W = 240, H = 190, box = 30;
		const dy = Math.min(44, (H - 30) / rows);
		for (const [r, ops] of Object.entries(byRow)) {
			ops.forEach((op, i) => { pos[op] = [W * (i + 1) / (ops.length + 1), H - 26 - Number(r) * dy]; });
		}
		unused.forEach((op, i) => { pos[op] = [18 + i * 36, 18]; });
		const line = (x1, y1, x2, y2, cls) => {
			const l = document.createElementNS(svgNS, "line");
			l.setAttribute("x1", x1); l.setAttribute("y1", y1); l.setAttribute("x2", x2); l.setAttribute("y2", y2);
			l.setAttribute("class", cls); svg.append(l);
		};
		// the outputs of the carriers onto a bus
		const xs = a.carriers.map(c => pos[c][0]);
		if (xs.length) {
			const by = H - 6;
			line(Math.min(...xs), by, Math.max(...xs), by, "dx-wire out");
			for (const c of a.carriers) line(pos[c][0], pos[c][1] + box / 2, pos[c][0], by, "dx-wire out");
		}
		for (const [f, t] of a.mods) line(pos[f][0], pos[f][1] + box / 2, pos[t][0], pos[t][1] - box / 2, "dx-wire");
		// the feedback loop, on the right of the operators it goes between
		{
			const [ff, ft] = a.feedback, [x1, y1] = pos[ff], [x2, y2] = pos[ft];
			const x = Math.max(x1, x2) + box / 2 + 8;
			const path = document.createElementNS(svgNS, "path");
			path.setAttribute("d", `M ${x1 + box / 2} ${y1} H ${x} V ${y2 - box / 2 - 6} H ${x2} V ${y2 - box / 2}`);
			path.setAttribute("class", "dx-wire fb" + (vals.feedback > 0 ? "" : " idle"));
			svg.append(path);
		}
		for (let op = 1; op <= OPS; op++) {
			const [x, y] = pos[op];
			const g = document.createElementNS(svgNS, "g");
			const carrier = a.carriers.includes(op);
			g.setAttribute("class", "dx-op" + (carrier ? " carrier" : "") + (row[op] < 0 ? " unused" : ""));
			const r = document.createElementNS(svgNS, "rect");
			r.setAttribute("x", x - box / 2); r.setAttribute("y", y - box / 2); r.setAttribute("width", box); r.setAttribute("height", box); r.setAttribute("rx", 5);
			const lvl = document.createElementNS(svgNS, "rect"); // the output level, as a bar along the bottom
			lvl.setAttribute("x", x - box / 2 + 3); lvl.setAttribute("y", y + box / 2 - 5);
			lvl.setAttribute("width", Math.max(0, (box - 6) * vals[`op${op}Level`] / 99)); lvl.setAttribute("height", 2.5);
			lvl.setAttribute("class", "dx-level");
			const t = document.createElementNS(svgNS, "text");
			t.setAttribute("x", x); t.setAttribute("y", y + 3); t.textContent = op;
			g.append(r, lvl, t);
			g.addEventListener("click", () => { if (vals.custom) set("carrier" + op, vals["carrier" + op] ? 0 : 1); });
			svg.append(g);
		}
	}

	// --- the operators ---
	const fmtFreq = op => () => {
		const c = vals[`op${op}Coarse`], f = vals[`op${op}Fine`];
		if (vals[`op${op}Fixed`]) {
			const hz = Math.pow(10, c & 3) * Math.pow(10, f / 100);
			return hz >= 1000 ? (hz / 1000).toFixed(2) + " kHz" : hz.toFixed(hz < 10 ? 3 : hz < 100 ? 2 : 1) + " Hz";
		}
		return "× " + ((c === 0 ? 0.5 : c) * (1 + f / 100)).toFixed(2);
	};
	const curves = ["-LIN", "-EXP", "+EXP", "+LIN"];
	const noteName = n => ["C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"][n % 12] + (Math.floor(n / 12) - 1);
	const table = el("div", "dx-ops");
	const headRow = el("div", "dx-op-row dx-op-head");
	for (const h of ["", "frequency", "", "detune", "level", "velocity", "rate scale", "amp mod", "envelope"]) headRow.append(el("span", "", h));
	table.append(headRow);
	for (let op = 1; op <= OPS; op++) {
		const r = el("div", "dx-op-row");
		const badge = el("button", "dx-badge", `OP ${op}`);
		badge.title = "the envelope and the keyboard scaling";
		const freq = el("span", "dx-freq");
		const freqText = el("span", "dx-freq-text");
		const showFreq = fmtFreq(op);
		for (const k of [`op${op}Coarse`, `op${op}Fine`, `op${op}Fixed`]) bind(k, () => { freqText.textContent = showFreq(); });
		freq.append(freqText, num(`op${op}Coarse`, v => "c" + v, "coarse: the ratio (0 is 0.5), or the decade of a fixed frequency"),
			num(`op${op}Fine`, v => "f" + v, "fine"));
		const mode = flag(`op${op}Fixed`, "fixed", "a fixed frequency instead of a ratio of the note");
		const detune = num(`op${op}Detune`, v => (v > 7 ? "+" : "") + (v - 7), "detune, -7 to +7");
		const level = el("span", "dx-levelbox");
		const bar = el("span", "dx-bar");
		bind(`op${op}Level`, v => { bar.style.width = `${v / 99 * 100}%`; });
		level.append(bar, num(`op${op}Level`, v => String(v), "output level, 0 to 99"));
		const env = document.createElement("canvas");
		env.className = "dx-eg";
		const detail = el("div", "dx-detail");
		badge.addEventListener("click", () => { detail.classList.toggle("open"); drawEnvelopes(); });
		env.addEventListener("click", () => { detail.classList.toggle("open"); drawEnvelopes(); });
		r.append(badge, freq, mode, detune, level, num(`op${op}VelocitySens`, String, "velocity sensitivity, 0 to 7"),
			num(`op${op}RateScaling`, String, "rate scaling: higher notes faster, 0 to 7"),
			num(`op${op}AmpModSens`, String, "how much the LFO moves its level, 0 to 3"), env);
		// the detail: the envelope's rates and levels, the keyboard scaling
		const eg = el("div", "dx-row");
		eg.append(el("span", "dx-label", "rates"));
		for (let i = 1; i <= 4; i++) eg.append(num(`op${op}Rate${i}`, v => `R${i} ${v}`));
		eg.append(el("span", "dx-label", "levels"));
		for (let i = 1; i <= 4; i++) eg.append(num(`op${op}Level${i}`, v => `L${i} ${v}`));
		const ks = el("div", "dx-row");
		ks.append(el("span", "dx-label", "scaling"),
			num(`op${op}BreakPoint`, v => "break " + noteName(v + 21), "break point"),
			num(`op${op}Wave`, v => waveNames[v], "the wave of the operator"),
			num(`op${op}LeftDepth`, v => "left " + v), num(`op${op}LeftCurve`, v => curves[v]),
			num(`op${op}RightDepth`, v => "right " + v), num(`op${op}RightCurve`, v => curves[v]));
		detail.append(eg, ks);
		const big = document.createElement("canvas");
		big.className = "dx-eg big";
		detail.append(big);
		for (let i = 1; i <= 4; i++) for (const k of [`op${op}Rate${i}`, `op${op}Level${i}`]) bind(k, () => drawEnvelopes());
		table.append(r, detail);
		r.dataset.op = op;
		envCanvases.push([op, env, big]);
	}
	root.append(table);

	// --- the LFO, the pitch envelope, the voice ---
	const voice = el("div", "dx-voice");
	const waves = ["TRI", "SAW↓", "SAW↑", "SQR", "SIN", "S&H"];
	const lfo = el("div", "dx-row");
	lfo.append(el("span", "dx-label dx-group", "LFO"), num("lfoWaveform", v => waves[v], "the wave"),
		labelled("speed", num("lfoSpeed")), labelled("delay", num("lfoDelay")),
		labelled("pitch", num("lfoPitchDepth")), labelled("amp", num("lfoAmpDepth")),
		labelled("sens", num("pitchModSens", String, "pitch modulation sensitivity, 0 to 7")), flag("lfoSync", "key sync", "restart at every note"));
	const pitch = el("div", "dx-row");
	pitch.append(el("span", "dx-label dx-group", "pitch"));
	for (let i = 1; i <= 4; i++) pitch.append(num(`pitchRate${i}`, v => `R${i} ${v}`));
	for (let i = 1; i <= 4; i++) pitch.append(num(`pitchLevel${i}`, v => `L${i} ${v - 50 > 0 ? "+" : ""}${v - 50}`));
	pitch.append(labelled("transpose", num("transpose", v => (v > 24 ? "+" : "") + (v - 24))), flag("oscSync", "osc sync", "restart the operators at every note"));
	const types = ["off", "low", "band", "high", "notch", "ladder 24", "ladder 12"];
	const filter = el("div", "dx-row");
	filter.append(el("span", "dx-label dx-group", "filter"), num("filterType", v => types[v], "the filter after the operators"),
		labelled("cutoff", num("filterCutoff")), labelled("res", num("filterResonance")),
		labelled("env", num("filterEnvAmount", String, "how far the envelope opens the cutoff, 0 to 6 octaves")),
		labelled("key", num("filterKeyTrack", String, "how much the cutoff follows the note")));
	for (const [k, l] of [["Attack", "A"], ["Decay", "D"], ["Sustain", "S"], ["Release", "R"]]) filter.append(num("filter" + k, v => `${l} ${v}`, "filter envelope"));
	voice.append(lfo, pitch, filter);
	root.append(voice);

	// the envelopes, small in the rows and big in the details
	function drawEnvelopes() {
		const accent = api.css("accent"), line = api.css("line"), track = api.css("track");
		for (const [op, small, big] of envCanvases) {
			for (const c of [small, big]) {
				if (!c.isConnected || !c.clientWidth) continue;
				const w = c.clientWidth, h = c.clientHeight, dpr = window.devicePixelRatio || 1;
				c.width = Math.round(w * dpr); c.height = Math.round(h * dpr);
				const g = c.getContext("2d");
				g.setTransform(dpr, 0, 0, dpr, 0, 0);
				g.fillStyle = track; g.fillRect(0, 0, w, h);
				const R = i => vals[`op${op}Rate${i}`], L = i => vals[`op${op}Level${i}`];
				const len = r => Math.pow(1.03 - r / 99, 2) + 0.02; // a slow rate takes long
				const seg = [len(R(1)), len(R(2)), len(R(3)), 0.35, len(R(4))];
				const total = seg.reduce((a, b) => a + b, 0), pad = 3;
				const X = t => pad + (w - 2 * pad) * t / total, Y = l => h - pad - (h - 2 * pad) * l / 99;
				let t = 0;
				const pts = [[0, L(4)]];
				for (const [i, lv] of [[0, L(1)], [1, L(2)], [2, L(3)], [3, L(3)], [4, L(4)]]) { t += seg[i]; pts.push([t, lv]); }
				g.strokeStyle = line; g.beginPath(); g.moveTo(X(pts[3][0]), 0); g.lineTo(X(pts[3][0]), h); g.stroke();
				g.beginPath();
				pts.forEach(([pt, lv], i) => i ? g.lineTo(X(pt), Y(lv)) : g.moveTo(X(pt), Y(lv)));
				g.strokeStyle = accent; g.lineWidth = c === big ? 2 : 1.5; g.stroke();
			}
		}
	}
	setTimeout(() => { drawAlgorithm(); drawEnvelopes(); }, 0);
	new ResizeObserver(() => drawEnvelopes()).observe(table);

	return {
		el: api.labelled(p, root),
		show(v) { vals.algorithm = v; render("algorithm"); },
		part(name, v) { const k = name.slice(p.name.length + 1); vals[k] = v; render(k); },
		update(st) {
			if (!st) return;
			if (st.name !== undefined) name.textContent = st.name;
			if (st.presets) { mine = st.presets; showMine(); }
		},
		restyle() { drawAlgorithm(); drawEnvelopes(); },
	};
});
