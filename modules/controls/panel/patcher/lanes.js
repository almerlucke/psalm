// the lanes of an automation module, over the whole width at the bottom of the patch: a row for every lane, drawn over the bars of the song with its sections on
// the ruler, and edited there. A click on a row adds a point (dragged to place it), a drag moves a point, a double-click gives the point the next shape (the way
// it goes on to the next point), Delete (or a right click, or a click with Alt) takes it away. The module keeps the lanes as text; the page sends them as they are.
// psalm.lanesView(send) returns {el, update(nodes, selected), heads(h)}: el goes over the patch, update is told the modules (those with lanes are shown), heads where the lanes are
psalm.lanesView = function (send) {
	"use strict";
	const el = (tag, cls, text) => { const e = document.createElement(tag); if (cls) e.className = cls; if (text !== undefined) e.textContent = text; return e; };
	const SHAPES = ["lin", "smooth", "step", "exp"];
	const SNAPS = [["free", 0], ["1/8 bar", 0.125], ["1/4 bar", 0.25], ["1/2 bar", 0.5], ["1 bar", 1], ["2 bars", 2], ["4 bars", 4]];
	const PAD = 8, RADIUS = 5;

	const root = el("div", "pt-lanes"); root.hidden = true; root.tabIndex = 0;
	const grip = el("div", "pt-lanes-grip"); grip.title = "drag to change the height";
	const head = el("div", "pt-lanes-head");
	const title = el("b", "", "Lanes");
	const modSel = el("select", "pt-lanes-mod"); modSel.title = "the automation module shown";
	const snapSel = el("select", "pt-lanes-snap"); snapSel.title = "points land on the grid (Shift: free)";
	SNAPS.forEach(([n, v], i) => { const o = el("option", "", "snap: " + n); o.value = String(v); if (v === 0.25) o.selected = true; snapSel.append(o); });
	const info = el("span", "pt-lanes-info");
	const close = el("button", "pt-x", "×"); close.title = "close the lanes";
	head.append(title, modSel, snapSel, info, close);
	const body = el("div", "pt-lanes-body");
	const empty = el("div", "pt-lanes-empty", "no automation module in the patch: add one (Song › automation), each lane is a mod output to cable to any setting");
	root.append(grip, head, body, empty);

	let mods = [];            // the modules with lanes: their nodes
	let current = "";         // the module shown
	let model = null;         // the lanes shown: {id, length, sections, lanes: [{name, points: [{x, y, shape}]}]}
	let rows = [];            // per lane: {canvas, wrap, head, dot, val, name}
	let ruler = null;
	let sel = null;           // the point chosen: {lane, point}
	let drag = null;
	let heads = {}, lastKey = "";
	let headAt = performance.now();
	let sentAt = 0, pending = 0;
	let colors = {};
	const restyle = () => {
		const cs = getComputedStyle(root);
		const get = n => cs.getPropertyValue("--" + n).trim();
		colors = {ink: get("ink") || "#222", accent: get("accent") || "#4a90e2", muted: get("muted") || "#888", line: get("line") || "#ccc", track: get("track") || "#eee", card: get("card") || "#fff", danger: get("danger") || "#e5484d"};
		draw();
	};

	// the value of a lane at a position (as the module reads it)
	const valueAt = (pts, x) => {
		if (!pts.length) return 0;
		if (x <= pts[0].x) return pts[0].y;
		const last = pts[pts.length - 1];
		if (x >= last.x) return last.y;
		let i = 1;
		while (pts[i].x <= x) i++;
		const a = pts[i - 1], b = pts[i];
		let t = (x - a.x) / (b.x - a.x);
		switch (a.shape) {
			case "step": return a.y;
			case "smooth": t = 0.5 - 0.5 * Math.cos(Math.PI * t); break;
			case "exp": if (a.y > 0 && b.y > 0) return a.y * Math.pow(b.y / a.y, t); break;
		}
		return a.y + (b.y - a.y) * t;
	};
	const snap = (x, free) => {
		const g = free ? 0 : Number(snapSel.value);
		x = Math.max(0, Math.min(x, model.length));
		return g > 0 ? Math.min(model.length, Math.round(x / g) * g) : Math.round(x * 1000) / 1000;
	};
	const num = (v, p) => String(Math.round(v * Math.pow(10, p)) / Math.pow(10, p));

	// --- drawing ---
	const size = c => {
		const dpr = window.devicePixelRatio || 1, w = c.clientWidth, h = c.clientHeight;
		if (c.width !== Math.round(w * dpr) || c.height !== Math.round(h * dpr)) { c.width = Math.round(w * dpr); c.height = Math.round(h * dpr); }
		const g = c.getContext("2d"); g.setTransform(dpr, 0, 0, dpr, 0, 0); g.clearRect(0, 0, w, h);
		return [g, w, h];
	};
	// the spacing of the lines of the bars: the least that leaves a line every 14 pixels
	const barStep = w => { for (const s of [1, 2, 4, 8, 16, 32, 64, 128]) if (model.length / s * 14 <= w) return s; return 256; };
	const px = (x, w) => x / model.length * w;
	const py = (y, h) => h - PAD - y * (h - 2 * PAD);

	function drawRuler() {
		if (!ruler || !model) return;
		const [g, w, h] = size(ruler);
		g.font = "11px system-ui, sans-serif"; g.textBaseline = "middle";
		const step = barStep(w);
		g.fillStyle = colors.muted; g.strokeStyle = colors.line; g.lineWidth = 1;
		for (let b = 0; b <= model.length; b += step) {
			const x = Math.round(px(b, w)) + 0.5;
			g.beginPath(); g.moveTo(x, h - 7); g.lineTo(x, h); g.stroke();
			if (b < model.length) g.fillText(String(b + 1), x + 3, h - 7);
		}
		g.textBaseline = "top";
		for (const s of model.sections || []) {
			const x0 = px(s.start, w), x1 = px(s.start + s.bars, w);
			g.fillStyle = colors.line; g.globalAlpha = 0.55; g.fillRect(x0 + 1, 1, Math.max(1, x1 - x0 - 2), 13); g.globalAlpha = 1;
			g.fillStyle = colors.ink;
			g.save(); g.beginPath(); g.rect(x0, 0, x1 - x0, 14); g.clip(); g.fillText(s.name, x0 + 4, 2); g.restore();
		}
	}

	function drawRow(i) {
		const r = rows[i], lane = model.lanes[i];
		const [g, w, h] = size(r.canvas);
		g.strokeStyle = colors.line; g.lineWidth = 1;
		const step = barStep(w);
		g.globalAlpha = 0.35;
		for (let b = 0; b <= model.length; b += step) { const x = Math.round(px(b, w)) + 0.5; g.beginPath(); g.moveTo(x, 0); g.lineTo(x, h); g.stroke(); }
		g.globalAlpha = 0.9;
		for (const s of model.sections || []) { const x = Math.round(px(s.start, w)) + 0.5; g.beginPath(); g.moveTo(x, 0); g.lineTo(x, h); g.stroke(); }
		g.globalAlpha = 0.3;
		for (const y of [0, 0.5, 1]) { const yy = Math.round(py(y, h)) + 0.5; g.beginPath(); g.moveTo(0, yy); g.lineTo(w, yy); g.stroke(); }
		g.globalAlpha = 1;
		const pts = lane.points;
		if (pts.length) {
			// the line, a value for every pixel (a step has its edge)
			g.beginPath();
			g.moveTo(0, py(pts[0].y, h));
			for (let x = 0; x <= w; x++) g.lineTo(x, py(valueAt(pts, x / w * model.length), h));
			g.strokeStyle = colors.accent; g.lineWidth = 2; g.stroke();
			g.lineTo(w, h); g.lineTo(0, h); g.closePath();
			g.globalAlpha = 0.14; g.fillStyle = colors.accent; g.fill(); g.globalAlpha = 1;
			for (const p of pts) {
				const x = px(p.x, w), y = py(p.y, h), chosen = sel && sel.point === p;
				g.beginPath(); g.arc(x, y, chosen ? RADIUS + 2 : RADIUS, 0, Math.PI * 2);
				g.fillStyle = chosen ? colors.accent : colors.card; g.fill();
				g.lineWidth = 2; g.strokeStyle = colors.accent; g.stroke();
				if (p.shape && p.shape !== "lin") { // the shape to the next point, small below it
					g.font = "10px system-ui, sans-serif"; g.fillStyle = colors.muted; g.textBaseline = "top";
					g.fillText(p.shape, Math.min(x + 6, w - 30), Math.min(y + 6, h - 12));
				}
			}
		}
	}

	function draw() {
		if (!model || root.hidden) return;
		drawRuler();
		model.lanes.forEach((_, i) => drawRow(i));
		showHead();
	}

	// the heads: where the song is, a line and the value of every lane on it
	function showHead() {
		if (!model || root.hidden) return;
		const h = heads[model.id];
		rows.forEach((r, i) => {
			if (!h || !h.live) { r.head.hidden = true; r.dot.hidden = true; return; }
			const t = Math.min(0.3, (performance.now() - headAt) / 1000);
			const x = Math.max(0, Math.min(model.length, h.x + h.rate * t));
			r.head.hidden = false; r.head.style.left = (x / model.length * 100) + "%";
			const v = (h.values || [])[i] || 0;
			r.dot.hidden = false; r.dot.style.left = (x / model.length * 100) + "%";
			r.dot.style.top = `calc(${PAD}px + (100% - ${2 * PAD}px) * ${1 - v})`;
			r.val.textContent = v.toFixed(2);
		});
	}
	let frame = 0;
	const tick = () => { frame = 0; if (!root.hidden && root.isConnected) { showHead(); frame = requestAnimationFrame(tick); } };

	// --- the rows ---
	function build() {
		body.textContent = ""; rows = []; ruler = null;
		if (!model) return;
		const top = el("div", "pt-lane-row ruler");
		top.append(el("span", "pt-lane-side"));
		const rwrap = el("div", "pt-lane-cv ruler-cv"); ruler = el("canvas"); rwrap.append(ruler); top.append(rwrap);
		body.append(top);
		model.lanes.forEach((lane, i) => {
			const row = el("div", "pt-lane-row");
			const side = el("span", "pt-lane-side");
			const name = el("input", "pt-lane-name"); name.value = lane.name; name.maxLength = 40; name.spellcheck = false; name.title = "the name of the lane";
			name.addEventListener("keydown", e => { e.stopPropagation(); if (e.key === "Enter") name.blur(); });
			name.addEventListener("change", () => { lane.name = name.value.trim() || "lane " + (i + 1); push(true); });
			const val = el("span", "pt-lane-val", "0.00"); val.title = "its value now";
			const clear = el("button", "", "Clear"); clear.title = "take every point of this lane away";
			clear.addEventListener("click", () => { if (lane.points.length) { lane.points = []; sel = null; push(true); draw(); } });
			side.append(name, val, clear);
			const wrap = el("div", "pt-lane-cv");
			const canvas = el("canvas");
			const headEl = el("div", "pt-lane-head"), dot = el("div", "pt-lane-dot");
			wrap.append(canvas, headEl, dot);
			row.append(side, wrap);
			body.append(row);
			rows.push({canvas, wrap, head: headEl, dot, val, name});
			wire(i, canvas);
		});
		draw();
	}

	// --- editing ---
	// the lanes go to the module: at most every 60 ms while dragging, always at the end
	function push(now) {
		if (!model) return;
		const go = () => {
			pending = 0; sentAt = performance.now();
			send({lanes: {id: model.id, lanes: model.lanes.map(l => ({name: l.name, points: l.points.map(p => ({x: p.x, y: p.y, shape: p.shape || ""}))}))}});
		};
		if (now) { if (pending) clearTimeout(pending); go(); return; }
		if (!pending) pending = setTimeout(go, Math.max(0, 60 - (performance.now() - sentAt)));
	}
	const sortPoints = lane => lane.points.sort((a, b) => a.x - b.x);
	const at = (e, canvas) => { const r = canvas.getBoundingClientRect(); return [e.clientX - r.left, e.clientY - r.top, r.width, r.height]; };
	const near = (lane, mx, my, w, h) => {
		let best = null, bd = (RADIUS + 5) * (RADIUS + 5);
		for (const p of lane.points) { const d = (px(p.x, w) - mx) ** 2 + (py(p.y, h) - my) ** 2; if (d <= bd) { best = p; bd = d; } }
		return best;
	};
	const remove = (lane, p) => { lane.points.splice(lane.points.indexOf(p), 1); if (sel && sel.point === p) sel = null; };
	const say = (p) => { info.textContent = p ? `bar ${num(p.x + 1, 3)} · ${num(p.y, 3)}` + (p.shape && p.shape !== "lin" ? " · " + p.shape : "") : ""; };

	function wire(i, canvas) {
		const lane = () => model.lanes[i];
		canvas.addEventListener("contextmenu", e => {
			e.preventDefault();
			const [mx, my, w, h] = at(e, canvas), p = near(lane(), mx, my, w, h);
			if (p) { remove(lane(), p); push(true); draw(); }
		});
		canvas.addEventListener("dblclick", e => {
			const [mx, my, w, h] = at(e, canvas), p = near(lane(), mx, my, w, h);
			if (!p) return;
			p.shape = SHAPES[(SHAPES.indexOf(p.shape || "lin") + 1) % SHAPES.length];
			if (p.shape === "lin") p.shape = "";
			sel = {lane: i, point: p}; say(p); push(true); draw();
		});
		canvas.addEventListener("pointerdown", e => {
			if (e.button !== 0) return;
			root.focus({preventScroll: true});
			const [mx, my, w, h] = at(e, canvas);
			let p = near(lane(), mx, my, w, h);
			if (p && e.altKey) { remove(lane(), p); push(true); draw(); return; }
			if (!p) { // a new point where it was clicked
				p = {x: snap(mx / w * model.length, e.shiftKey), y: Math.max(0, Math.min(1, (h - PAD - my) / (h - 2 * PAD))), shape: ""};
				p.y = Math.round(p.y * 1000) / 1000;
				lane().points.push(p); sortPoints(lane());
				push(true);
			}
			sel = {lane: i, point: p};
			drag = {canvas, lane: i, point: p, moved: false};
			canvas.setPointerCapture(e.pointerId);
			say(p); draw();
		});
		canvas.addEventListener("pointermove", e => {
			if (!drag || drag.canvas !== canvas) return;
			const [mx, my, w, h] = at(e, canvas);
			const p = drag.point;
			p.x = snap(mx / w * model.length, e.shiftKey);
			p.y = Math.round(Math.max(0, Math.min(1, (h - PAD - my) / (h - 2 * PAD))) * 1000) / 1000;
			drag.moved = true;
			sortPoints(lane());
			say(p); draw(); push(false);
		});
		const up = e => {
			if (!drag || drag.canvas !== canvas) return;
			drag = null;
			try { canvas.releasePointerCapture(e.pointerId); } catch (_) {}
			push(true);
		};
		canvas.addEventListener("pointerup", up);
		canvas.addEventListener("pointercancel", up);
	}

	root.addEventListener("keydown", e => {
		if (e.target !== root && /INPUT|SELECT|TEXTAREA/.test(e.target.tagName)) return;
		if ((e.key === "Delete" || e.key === "Backspace") && sel && model && model.lanes[sel.lane]) {
			e.preventDefault(); e.stopPropagation(); // the patch would take the modules chosen away
			remove(model.lanes[sel.lane], sel.point); say(null); push(true); draw();
		} else if (e.key === "Escape") {
			e.stopPropagation(); setOpen(false);
		}
	});

	// --- the modules ---
	function choose(id) {
		const n = mods.find(n => n.id === id);
		current = n ? n.id : "";
		modSel.value = current;
		empty.hidden = !!n;
		sel = null; say(null);
		model = n ? take(n) : null;
		lastKey = n ? n.id + JSON.stringify(n.lanes) : "";
		build();
	}
	const take = n => ({id: n.id, length: Math.max(1, n.lanes.length || 1), sections: n.lanes.sections || [],
		lanes: n.lanes.lanes.map(l => ({name: l.name, points: (l.points || []).map(p => ({x: p.x, y: p.y, shape: p.shape || ""}))}))});

	function update(nodes, selected) {
		mods = nodes.filter(n => n.lanes);
		const ids = mods.map(n => n.id + ":" + n.kind + (n.preset || "")).join(",");
		if (modSel.dataset.ids !== ids) {
			modSel.dataset.ids = ids; modSel.innerHTML = "";
			for (const n of mods) { const o = el("option", "", n.id + (n.lanes.lanes.length ? " · " + n.lanes.lanes.length + " lanes" : "")); o.value = n.id; modSel.append(o); }
			modSel.hidden = mods.length < 2;
		}
		if (!mods.length) { current = ""; if (model) { model = null; build(); } empty.hidden = false; return; }
		const wanted = mods.some(n => n.id === current) ? current : (mods.find(n => n.id === selected) || mods[0]).id;
		const n = mods.find(n => n.id === wanted);
		const key = wanted + JSON.stringify(n.lanes);
		if (wanted !== current) { lastKey = key; choose(wanted); return; }
		// what the module has now, unless a lane is being drawn (it gets what was drawn)
		if (key === lastKey || drag || pending || performance.now() - sentAt < 400) return;
		lastKey = key;
		const was = model && model.lanes.length, kept = sel && model && model.lanes[sel.lane] ? {lane: sel.lane, x: sel.point.x, y: sel.point.y} : null;
		model = take(n); empty.hidden = true;
		sel = null;
		if (kept && model.lanes[kept.lane]) { const p = model.lanes[kept.lane].points.find(p => p.x === kept.x && p.y === kept.y); if (p) sel = {lane: kept.lane, point: p}; }
		if (was !== model.lanes.length) build();
		else {
			rows.forEach((r, i) => { if (document.activeElement !== r.name) r.name.value = model.lanes[i].name; });
			draw();
		}
	}
	modSel.addEventListener("change", () => choose(modSel.value));

	function setOpen(on) {
		root.hidden = !on;
		if (on) { restyle(); build(); if (!frame) frame = requestAnimationFrame(tick); } else if (frame) { cancelAnimationFrame(frame); frame = 0; }
		if (api.onToggle) api.onToggle(on);
	}
	close.addEventListener("click", () => setOpen(false));

	// the height: dragged from the top edge
	grip.addEventListener("pointerdown", e => {
		e.preventDefault(); grip.setPointerCapture(e.pointerId);
		const stage = root.parentElement, bottom = root.getBoundingClientRect().bottom;
		const move = ev => { root.style.height = Math.max(150, Math.min(stage.clientHeight - 20, bottom - ev.clientY)) + "px"; body.style.flex = "1 1 auto"; draw(); };
		const up = () => { grip.removeEventListener("pointermove", move); grip.removeEventListener("pointerup", up); };
		grip.addEventListener("pointermove", move); grip.addEventListener("pointerup", up);
	});
	new ResizeObserver(() => draw()).observe(body);

	const api = {
		el: root,
		open: () => setOpen(true),
		toggle: () => setOpen(root.hidden),
		get isOpen() { return !root.hidden; },
		update,
		heads(h) { heads = h || {}; headAt = performance.now(); },
		restyle,
		onToggle: null,
	};
	return api;
};
