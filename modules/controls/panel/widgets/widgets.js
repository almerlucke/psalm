// the widgets: an XY pad, a keyboard, pads, an envelope and a scope
(() => {
"use strict";
const clamp01 = x => Math.max(0, Math.min(1, x));
const typing = e => e.target && (e.target.tagName === "INPUT" || e.target.tagName === "SELECT" || e.target.isContentEditable);

// a canvas the size it is drawn at, sharp on any screen
function sized(canvas) {
	const w = canvas.clientWidth, h = canvas.clientHeight, dpr = window.devicePixelRatio || 1;
	if (canvas.width !== Math.round(w * dpr) || canvas.height !== Math.round(h * dpr)) {
		canvas.width = Math.round(w * dpr); canvas.height = Math.round(h * dpr);
	}
	const g = canvas.getContext("2d");
	g.setTransform(dpr, 0, 0, dpr, 0, 0);
	g.clearRect(0, 0, w, h);
	return [g, w, h];
}

// --- XY: x across, y up ---
psalm.register("xy", (p, api) => {
	const py = p.parts[0];
	const pad = document.createElement("div");
	pad.className = "xypad"; pad.tabIndex = 0;
	pad.innerHTML = '<div class="hline"></div><div class="vline"></div><div class="dot"></div>';
	const dot = pad.querySelector(".dot"), hl = pad.querySelector(".hline"), vl = pad.querySelector(".vline");
	const readout = document.createElement("div"); readout.className = "value";
	let x = api.position(p, p.value), y = api.position(py, py.value), dragging = false;
	const draw = () => {
		dot.style.left = vl.style.left = `${x * 100}%`;
		dot.style.bottom = hl.style.bottom = `${y * 100}%`;
		readout.textContent = `${p.data.x} ${api.format(p, api.valueAt(p, x))} · ${p.data.y} ${api.format(py, api.valueAt(py, y))}`;
	};
	const move = e => {
		const r = pad.getBoundingClientRect();
		x = clamp01((e.clientX - r.left) / r.width); y = clamp01(1 - (e.clientY - r.top) / r.height);
		draw();
		api.send(p.name, api.valueAt(p, x)); api.send(py.name, api.valueAt(py, y));
	};
	pad.addEventListener("pointerdown", e => { dragging = true; pad.setPointerCapture(e.pointerId); move(e); e.preventDefault(); });
	pad.addEventListener("pointermove", e => { if (dragging) move(e); });
	const up = () => { dragging = false; };
	pad.addEventListener("pointerup", up); pad.addEventListener("pointercancel", up);
	pad.addEventListener("dblclick", () => {
		x = api.position(p, p.default); y = api.position(py, py.default); draw();
		api.send(p.name, p.default); api.send(py.name, py.default);
	});
	pad.addEventListener("keydown", e => {
		const step = e.shiftKey ? 0.1 : 0.02;
		const dx = {ArrowRight: step, ArrowLeft: -step}[e.key] || 0, dy = {ArrowUp: step, ArrowDown: -step}[e.key] || 0;
		if (!dx && !dy) return;
		e.preventDefault();
		x = clamp01(x + dx); y = clamp01(y + dy); draw();
		api.send(p.name, api.valueAt(p, x)); api.send(py.name, api.valueAt(py, y));
	});
	draw();
	return {
		el: api.labelled(p, pad, readout),
		show(v) { if (!dragging) { x = api.position(p, v); draw(); } },
		part(name, v) { if (!dragging && name === py.name) { y = api.position(py, v); draw(); } },
		bindable: true,
	};
});

// --- Keys: a keyboard, lower on a key louder; the computer's keys play the first one on the page ---
let computerKeys = null; // the keyboard the computer's keys play
const letters = "awsedftgyhujkolp;'"; // C to F, an octave and a half
psalm.register("keys", (p, api) => {
	const from = p.data.from, last = from + 12 * p.data.octaves;
	const black = n => [1, 3, 6, 8, 10].includes(((n % 12) + 12) % 12);
	const box = document.createElement("div");
	box.className = "keys";
	const els = {};
	let whites = 0;
	for (let n = from; n <= last; n++) if (!black(n)) whites++;
	let w = 0;
	for (let n = from; n <= last; n++) {
		const k = document.createElement("div");
		k.dataset.note = n;
		if (black(n)) {
			k.className = "key black";
			k.style.left = `calc(${w} * 100% / ${whites} - 100% / ${whites} * 0.32)`;
			k.style.width = `calc(100% / ${whites} * 0.64)`;
		} else {
			k.className = "key white";
			k.style.left = `calc(${w} * 100% / ${whites})`;
			k.style.width = `calc(100% / ${whites})`;
			if (n % 12 === 0) k.textContent = "C" + (Math.floor(n / 12) - 1);
			w++;
		}
		els[n] = k;
		box.append(k);
	}
	box.style.setProperty("--whites", whites);
	const held = new Map(); // a pointer (or a computer key) -> the note it holds
	const on = (who, n, velocity) => {
		off(who);
		held.set(who, n);
		els[n] && els[n].classList.add("down");
		api.message(p.name, {note: n, velocity, on: true});
	};
	const off = who => {
		if (!held.has(who)) return;
		const n = held.get(who);
		held.delete(who);
		if (![...held.values()].includes(n)) els[n] && els[n].classList.remove("down");
		api.message(p.name, {note: n, on: false});
	};
	const keyAt = e => {
		const el = document.elementFromPoint(e.clientX, e.clientY);
		return el && el.parentNode === box && el.dataset.note ? el : null;
	};
	const velocityOn = (el, e) => {
		const r = el.getBoundingClientRect();
		return clamp01(0.25 + 0.75 * (e.clientY - r.top) / r.height);
	};
	box.addEventListener("pointerdown", e => {
		const el = keyAt(e);
		if (!el) return;
		e.preventDefault();
		on(e.pointerId, Number(el.dataset.note), velocityOn(el, e));
	});
	box.addEventListener("pointermove", e => { // sliding over the keys plays them (a glissando)
		if (!held.has(e.pointerId)) return;
		const el = keyAt(e);
		if (el && Number(el.dataset.note) !== held.get(e.pointerId)) on(e.pointerId, Number(el.dataset.note), velocityOn(el, e));
	});
	for (const t of ["pointerup", "pointercancel", "pointerleave"]) box.addEventListener(t, e => off(e.pointerId));
	window.addEventListener("pointerup", e => off(e.pointerId));

	// the computer's keys: A W S E D ... (Z and X move an octave down and up)
	let octave = Math.max(0, Math.round((60 - from) / 12)) * 12;
	if (!computerKeys) {
		computerKeys = p.name;
		window.addEventListener("keydown", e => {
			if (computerKeys !== p.name || e.repeat || e.metaKey || e.ctrlKey || e.altKey || typing(e)) return;
			const k = e.key.toLowerCase();
			if (k === "z") { octave = Math.max(0, octave - 12); return; }
			if (k === "x") { octave = Math.min(12 * (p.data.octaves - 1), octave + 12); return; }
			const i = letters.indexOf(k);
			if (i < 0) return;
			const n = from + octave + i;
			if (n > last) return;
			e.preventDefault();
			on("key:" + k, n, 0.75);
		});
		window.addEventListener("keyup", e => off("key:" + e.key.toLowerCase()));
		window.addEventListener("blur", () => { for (const who of [...held.keys()]) off(who); });
		box.title = "the computer's keys play it too: A W S E D F T G Y H U J K (Z and X: an octave down, up)";
	}
	window.addEventListener("pagehide", () => { for (const who of [...held.keys()]) off(who); });
	return {el: api.labelled(p, box), show() {}};
});

// --- Pads: hits, louder in the middle of a pad; the number keys hit the first nine ---
let numberKeys = null;
psalm.register("pads", (p, api) => {
	const box = document.createElement("div");
	box.className = "pads";
	box.style.gridTemplateColumns = `repeat(${p.data.columns}, 1fr)`;
	const pads = p.data.names.map((name, i) => {
		const pad = document.createElement("div");
		pad.className = "pad";
		pad.innerHTML = `<span class="num">${i < 9 ? i + 1 : ""}</span><span class="label"></span>`;
		pad.querySelector(".label").textContent = name;
		box.append(pad);
		return pad;
	});
	const hit = (i, velocity) => {
		const pad = pads[i];
		pad.classList.remove("hit"); void pad.offsetWidth; pad.classList.add("hit");
		pad.style.setProperty("--velocity", velocity);
		api.message(p.name, {pad: i, velocity});
	};
	pads.forEach((pad, i) => pad.addEventListener("pointerdown", e => {
		e.preventDefault();
		const r = pad.getBoundingClientRect();
		const dx = (e.clientX - r.left) / r.width - 0.5, dy = (e.clientY - r.top) / r.height - 0.5;
		hit(i, clamp01(1 - 0.9 * Math.hypot(dx, dy) * 1.41));
	}));
	if (!numberKeys) {
		numberKeys = p.name;
		window.addEventListener("keydown", e => {
			if (numberKeys !== p.name || e.repeat || e.metaKey || e.ctrlKey || e.altKey || typing(e)) return;
			const i = Number(e.key) - 1;
			if (i >= 0 && i < Math.min(9, pads.length)) { e.preventDefault(); hit(i, 0.8); }
		});
	}
	return {el: api.labelled(p, box), show() {}};
});

// --- Envelope: drag the peak (the attack), the knee (the decay, and up and down the sustain) and the
// end (the release) ---
psalm.register("envelope", (p, api) => {
	const [pd, ps, pr] = p.parts;
	const canvas = document.createElement("canvas");
	canvas.className = "envelope";
	const readout = document.createElement("div"); readout.className = "value";
	const pos = {a: api.position(p, p.value), d: api.position(pd, pd.value), s: api.position(ps, ps.value), r: api.position(pr, pr.value)};
	let colors = {}, dragging = null;
	const restyle = () => { colors = {line: api.css("line"), accent: api.css("accent"), ink: api.css("ink"), track: api.css("track")}; draw(); };
	const pad = 10;
	const points = (w, h) => {
		const seg = (w - 2 * pad) * 0.27, hold = (w - 2 * pad) * 0.19, top = pad, bottom = h - pad;
		const x1 = pad + Math.max(2, pos.a * seg), x2 = x1 + Math.max(2, pos.d * seg), x3 = x2 + hold, x4 = x3 + Math.max(2, pos.r * seg);
		const ys = bottom - (bottom - top) * pos.s;
		return {seg, x0: pad, x1, x2, x3, x4, top, bottom, ys};
	};
	function draw() {
		if (!canvas.isConnected) return;
		const [g, w, h] = sized(canvas);
		const q = points(w, h);
		g.strokeStyle = colors.line; g.lineWidth = 1;
		g.beginPath(); g.moveTo(pad, q.bottom + 0.5); g.lineTo(w - pad, q.bottom + 0.5); g.stroke();
		g.beginPath();
		g.moveTo(q.x0, q.bottom); g.lineTo(q.x1, q.top); g.lineTo(q.x2, q.ys); g.lineTo(q.x3, q.ys); g.lineTo(q.x4, q.bottom);
		g.globalAlpha = 0.18; g.fillStyle = colors.accent; g.fill(); g.globalAlpha = 1;
		g.strokeStyle = colors.accent; g.lineWidth = 2; g.stroke();
		g.fillStyle = colors.ink;
		for (const [x, y] of [[q.x1, q.top], [q.x2, q.ys], [q.x4, q.bottom]]) { g.beginPath(); g.arc(x, y, 4.5, 0, 2 * Math.PI); g.fill(); }
		const v = (par, x) => api.format(par, api.valueAt(par, x));
		readout.textContent = `A ${v(p, pos.a)} · D ${v(pd, pos.d)} · S ${v(ps, pos.s)} · R ${v(pr, pos.r)}`;
	}
	const send = () => {
		api.send(p.name, api.valueAt(p, pos.a)); api.send(pd.name, api.valueAt(pd, pos.d));
		api.send(ps.name, api.valueAt(ps, pos.s)); api.send(pr.name, api.valueAt(pr, pos.r));
	};
	const at = e => { const r = canvas.getBoundingClientRect(); return [e.clientX - r.left, e.clientY - r.top, r.width, r.height]; };
	canvas.addEventListener("pointerdown", e => {
		const [x, y, w, h] = at(e), q = points(w, h);
		const handles = [["a", q.x1, q.top], ["d", q.x2, q.ys], ["r", q.x4, q.bottom]];
		dragging = handles.reduce((best, hd) => Math.hypot(hd[1] - x, hd[2] - y) < Math.hypot(best[1] - x, best[2] - y) ? hd : best)[0];
		canvas.setPointerCapture(e.pointerId);
		e.preventDefault();
	});
	canvas.addEventListener("pointermove", e => {
		if (!dragging) return;
		const [x, y, w, h] = at(e), q = points(w, h);
		if (dragging === "a") pos.a = clamp01((x - q.x0) / q.seg);
		if (dragging === "d") { pos.d = clamp01((x - q.x1) / q.seg); pos.s = clamp01((q.bottom - y) / (q.bottom - q.top)); }
		if (dragging === "r") pos.r = clamp01((x - q.x3) / q.seg);
		draw(); send();
	});
	const up = () => { dragging = null; };
	canvas.addEventListener("pointerup", up); canvas.addEventListener("pointercancel", up);
	canvas.addEventListener("dblclick", () => {
		pos.a = api.position(p, p.default); pos.d = api.position(pd, pd.default);
		pos.s = api.position(ps, ps.default); pos.r = api.position(pr, pr.default);
		draw(); send();
	});
	new ResizeObserver(draw).observe(canvas);
	setTimeout(restyle, 0);
	const keys = {[p.name]: "a", [pd.name]: "d", [ps.name]: "s", [pr.name]: "r"};
	const params = {a: p, d: pd, s: ps, r: pr};
	return {
		el: api.labelled(p, canvas, readout),
		show(v) { if (!dragging) { pos.a = api.position(p, v); draw(); } },
		part(name, v) { const k = keys[name]; if (k && !dragging) { pos[k] = api.position(params[k], v); draw(); } },
		restyle,
		bindable: true,
	};
});

// --- Scope: the wave (steady on its rising zero crossings) and the spectrum; a click switches ---
function fft(re, im) { // in place, a power of two
	const n = re.length;
	for (let i = 1, j = 0; i < n; i++) {
		let bit = n >> 1;
		for (; j & bit; bit >>= 1) j ^= bit;
		j ^= bit;
		if (i < j) { [re[i], re[j]] = [re[j], re[i]]; [im[i], im[j]] = [im[j], im[i]]; }
	}
	for (let len = 2; len <= n; len <<= 1) {
		const a = -2 * Math.PI / len, wr = Math.cos(a), wi = Math.sin(a);
		for (let i = 0; i < n; i += len) {
			let cr = 1, ci = 0;
			for (let k = 0; k < len / 2; k++) {
				const ur = re[i + k], ui = im[i + k];
				const vr = re[i + k + len / 2] * cr - im[i + k + len / 2] * ci, vi = re[i + k + len / 2] * ci + im[i + k + len / 2] * cr;
				re[i + k] = ur + vr; im[i + k] = ui + vi; re[i + k + len / 2] = ur - vr; im[i + k + len / 2] = ui - vi;
				[cr, ci] = [cr * wr - ci * wi, cr * wi + ci * wr];
			}
		}
	}
}
psalm.register("scope", (p, api) => {
	const canvas = document.createElement("canvas");
	canvas.className = "scope";
	canvas.title = "click: the wave, the spectrum, both";
	let mode = 2, last = null, spectrum = null, colors = {};
	const restyle = () => { colors = {line: api.css("line"), accent: api.css("accent"), muted: api.css("muted"), ok: api.css("meter-low")}; draw(); };
	canvas.addEventListener("click", () => { mode = (mode + 1) % 3; draw(); });
	function wave(g, x0, y0, w, h, s) {
		const n = s.length;
		let start = 0;
		for (let i = 1; i < n / 2; i++) if (s[i - 1] < 0 && s[i] >= 0) { start = i; break; }
		const span = Math.min(n - start, n / 2);
		let peak = 0.05;
		for (let i = 0; i < span; i++) peak = Math.max(peak, Math.abs(s[start + i]));
		const scale = 0.9 / Math.max(peak, 0.25); // quiet sounds made bigger, up to four times
		g.strokeStyle = colors.line; g.lineWidth = 1;
		g.beginPath(); g.moveTo(x0, y0 + h / 2 + 0.5); g.lineTo(x0 + w, y0 + h / 2 + 0.5); g.stroke();
		g.strokeStyle = colors.accent; g.lineWidth = 1.5; g.beginPath();
		for (let i = 0; i < span; i++) {
			const x = x0 + w * i / (span - 1), y = y0 + h / 2 - s[start + i] * scale * h / 2;
			i ? g.lineTo(x, y) : g.moveTo(x, y);
		}
		g.stroke();
	}
	function spec(g, x0, y0, w, h, sr) {
		if (!spectrum) return;
		const n = spectrum.length * 2, lo = 20, hi = sr / 2;
		const xOf = f => x0 + w * Math.log(f / lo) / Math.log(hi / lo);
		g.fillStyle = colors.muted; g.font = "10px sans-serif";
		g.strokeStyle = colors.line; g.lineWidth = 1;
		for (const f of [100, 1000, 10000]) {
			if (f >= hi) continue;
			const x = Math.round(xOf(f)) + 0.5;
			g.beginPath(); g.moveTo(x, y0); g.lineTo(x, y0 + h); g.stroke();
			g.fillText(f >= 1000 ? f / 1000 + "k" : String(f), x + 3, y0 + 11);
		}
		g.beginPath();
		g.moveTo(x0, y0 + h);
		for (let k = 1; k < spectrum.length; k++) {
			const f = k * sr / n;
			if (f < lo) continue;
			const db = spectrum[k], y = y0 + h * clamp01(-db / 90);
			g.lineTo(xOf(f), y);
		}
		g.lineTo(x0 + w, y0 + h);
		g.globalAlpha = 0.25; g.fillStyle = colors.ok; g.fill(); g.globalAlpha = 1;
		g.strokeStyle = colors.ok; g.lineWidth = 1.5; g.stroke();
	}
	function draw() {
		if (!canvas.isConnected) return;
		const [g, w, h] = sized(canvas);
		if (!last) return;
		const s = last.samples;
		if (mode === 0) wave(g, 0, 0, w, h, s);
		if (mode === 1) spec(g, 0, 0, w, h, last.sr || 44100);
		if (mode === 2) { wave(g, 0, 0, w * 0.5 - 4, h, s); spec(g, w * 0.5 + 4, 0, w * 0.5 - 4, h, last.sr || 44100); }
	}
	const analyse = s => {
		const n = s.length, re = new Float64Array(n), im = new Float64Array(n);
		for (let i = 0; i < n; i++) re[i] = s[i] * (0.5 - 0.5 * Math.cos(2 * Math.PI * i / (n - 1)));
		fft(re, im);
		const db = new Float64Array(n / 2);
		for (let k = 0; k < n / 2; k++) db[k] = 20 * Math.log10(Math.hypot(re[k], im[k]) * 4 / n + 1e-9);
		spectrum = spectrum && spectrum.length === db.length ? spectrum.map((old, k) => Math.max(db[k], old - 3)) : db; // falls back slowly
	};
	new ResizeObserver(draw).observe(canvas);
	setTimeout(restyle, 0);
	return {
		el: api.labelled(p, canvas),
		show() {},
		update(st) { if (st && st.samples) { last = st; analyse(st.samples); draw(); } },
		restyle,
	};
});
})();
