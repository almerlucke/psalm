// the loop of a looper: a cell for every bar with its four beats lit as they pass (the first beat of the first bar red while a layer is recorded), what it
// is waiting for, and a row for every layer with its notes and a button to take it away. The page counts on from the position the server last told
psalm.register("looper", (p, api) => {
	const el = document.createElement("div");
	el.className = "looper";
	const bars = document.createElement("div"); bars.className = "lp-bars";
	const status = document.createElement("div"); status.className = "lp-status";
	const list = document.createElement("div");
	const empty = document.createElement("div"); empty.className = "lp-empty";
	empty.textContent = "no layers yet: switch record on, and play from the start of the next pass";
	el.append(bars, status, list, empty);
	const host = api.labelled(p, el);
	host.classList.add("looper-host");

	let st = {bars: 2, pos: 0, bps: 2, armed: false, recording: false, layers: [], cur: []};
	let at = performance.now(), cells = [], rows = [], lastBars = 0;
	let colors = {};
	const restyle = () => { colors = {ink: api.css("ink"), accent: api.css("accent"), muted: api.css("muted")}; };
	restyle();

	const rebuildBars = () => {
		bars.textContent = ""; cells = [];
		for (let b = 0; b < st.bars; b++) {
			const cell = document.createElement("div"); cell.className = "lp-bar" + (b === 0 ? " first" : "");
			const num = document.createElement("b"); num.textContent = String(b + 1); cell.append(num);
			const ticks = [0, 1, 2, 3].map(i => { const t = document.createElement("i"); t.style.left = (4 + i * 24.5) + "%"; cell.append(t); return t; });
			bars.append(cell); cells.push({cell, ticks});
		}
		lastBars = st.bars;
	};
	const rebuildRows = () => {
		list.textContent = ""; rows = [];
		st.layers.forEach((layer, i) => {
			const row = document.createElement("div"); row.className = "lp-layer";
			const label = document.createElement("span"); label.textContent = "Layer " + (i + 1);
			const canvas = document.createElement("canvas");
			const del = document.createElement("button"); del.textContent = "Take away"; del.title = "take this layer away; the others go on";
			del.addEventListener("click", () => { api.message(p.name, {drop: layer.id}); });
			row.append(label, canvas, del); list.append(row);
			rows.push({id: layer.id, canvas, notes: layer.notes || [], peaks: layer.peaks});
		});
		// the layer being recorded
		const row = document.createElement("div"); row.className = "lp-layer";
		const label = document.createElement("span"); label.textContent = "Now";
		const canvas = document.createElement("canvas"); row.append(label, canvas);
		row.hidden = true; list.append(row);
		rows.push({id: -1, canvas, row, notes: []});
		empty.hidden = st.layers.length > 0 || st.recording;
	};

	const drawNotes = (r, loop, head, live) => {
		const c = r.canvas, dpr = window.devicePixelRatio || 1, w = c.clientWidth, h = c.clientHeight;
		if (c.width !== Math.round(w * dpr) || c.height !== Math.round(h * dpr)) { c.width = Math.round(w * dpr); c.height = Math.round(h * dpr); }
		const g = c.getContext("2d"); g.setTransform(dpr, 0, 0, dpr, 0, 0); g.clearRect(0, 0, w, h);
		if (r.peaks) { // a sound: the loudest of every column, up and down from the middle
			g.globalAlpha = 0.25; g.fillStyle = colors.muted;
			for (let b = 0; b <= loop; b++) g.fillRect(Math.round(b / loop * (w - 1)), 0, 1, h);
			g.globalAlpha = 1; g.fillStyle = live ? "#e5484d" : colors.accent;
			const cw = w / Math.max(1, r.peaks.length);
			r.peaks.forEach((v, i) => { const a = Math.min(1, Math.sqrt(v)) * (h - 4); g.fillRect(i * cw, (h - a) / 2, Math.max(1, cw - 0.5), Math.max(1, a)); });
			g.fillStyle = colors.ink; g.fillRect(Math.round(head / loop * (w - 1)), 0, 2, h);
			return;
		}
		const notes = r.notes;
		let lo = 127, hi = 0;
		for (const n of notes) { lo = Math.min(lo, n[2]); hi = Math.max(hi, n[2]); }
		const span = Math.max(12, hi - lo + 1), rowH = Math.min(8, h / span);
		g.globalAlpha = 0.25; g.fillStyle = colors.muted;
		for (let b = 0; b <= loop; b++) g.fillRect(Math.round(b / loop * (w - 1)), 0, 1, h);
		g.globalAlpha = 1; g.fillStyle = live ? "#e5484d" : colors.accent;
		for (const [s, l, k] of notes) {
			const y = h - 3 - ((k - lo) / span) * (h - 6) - rowH;
			g.fillRect(s / loop * w, Math.max(0, y), Math.max(2, l / loop * w - 1), Math.max(2, rowH));
		}
		g.fillStyle = colors.ink; g.fillRect(Math.round(head / loop * (w - 1)), 0, 2, h);
	};

	const frame = () => {
		if (!el.isConnected) return;
		const loop = st.bars * 4, t = (performance.now() - at) / 1000;
		const head = (((st.pos + st.bps * t) % loop) + loop) % loop;
		const bar = Math.floor(head / 4), beat = Math.floor(head % 4);
		cells.forEach((c, b) => {
			c.cell.classList.toggle("now", b === bar);
			c.ticks.forEach((tk, i) => tk.classList.toggle("on", b === bar && i <= beat));
		});
		const toStart = loop - head;
		if (st.recording) {
			status.className = "lp-status rec"; status.textContent = `recording layer ${st.layers.length + 1}: bar ${bar + 1}, beat ${beat + 1}`;
		} else if (st.armed) {
			status.className = "lp-status rec"; status.textContent = `recording starts at bar 1, in ${toStart.toFixed(1)} beats`;
		} else {
			status.className = "lp-status"; status.textContent = `bar ${bar + 1}, beat ${beat + 1}`;
		}
		for (const r of rows) {
			if (r.id === -1) { r.row.hidden = !st.recording; if (!st.recording) continue; r.notes = st.cur || []; r.peaks = st.curPeaks; }
			drawNotes(r, loop, head, r.id === -1);
		}
		requestAnimationFrame(frame);
	};
	requestAnimationFrame(frame);
	rebuildBars(); rebuildRows();

	return {
		el: host,
		show() {},
		update(s) {
			if (!s || !s.layers) return;
			const ids = s => s.layers.map(l => l.id).join(",");
			const wasRec = st.recording, wasIds = ids(st);
			st = s; at = performance.now();
			if (st.bars !== lastBars) rebuildBars();
			if (ids(st) !== wasIds || wasRec !== st.recording) rebuildRows();
			else rows.forEach(r => { const l = st.layers.find(l => l.id === r.id); if (l) { r.notes = l.notes || []; r.peaks = l.peaks; } });
		},
		restyle,
	};
});
