// the notes of a MIDI file scrolling past a playhead: [start, end, key, channel, velocity] each
psalm.register("pianoroll", (p, api) => {
	const d = p.data, notes = d.notes;
	const canvas = document.createElement("canvas");
	const time = document.createElement("div");
	time.className = "time";
	const el = api.labelled(p, canvas, time);
	el.classList.add("pianoroll");

	let lo = 127, hi = 0, longest = 0;
	for (const n of notes) { lo = Math.min(lo, n[2]); hi = Math.max(hi, n[2]); longest = Math.max(longest, n[1] - n[0]); }
	if (lo > hi) { lo = 48; hi = 72; }
	lo -= 2; hi += 2;

	// where the player is: the last state, moved on at its rate since it came
	let pos = 0, rate = 0, at = performance.now(), muted = new Set(), seconds = d.seconds || 10;
	const now = () => Math.min(d.duration, pos + rate * (performance.now() - at) / 1000);

	let colors = {};
	const restyle = () => { colors = {bg: api.css("track"), ink: api.css("ink"), muted: api.css("muted"), line: api.css("line"), accent: api.css("accent")}; };
	restyle();
	const hue = ch => ch === 10 ? null : ((ch - 1) * 47 + 200) % 360;
	const clock = s => `${Math.floor(s / 60)}:${String(Math.floor(s % 60)).padStart(2, "0")}`;

	// the first note that could still be on screen at a time
	const firstFrom = t => {
		let a = 0, b = notes.length;
		while (a < b) { const m = (a + b) >> 1; if (notes[m][0] < t - longest) a = m + 1; else b = m; }
		return a;
	};

	const draw = () => {
		if (!canvas.isConnected) return;
		const w = canvas.clientWidth, h = canvas.clientHeight, dpr = window.devicePixelRatio || 1;
		if (canvas.width !== Math.round(w * dpr) || canvas.height !== Math.round(h * dpr)) {
			canvas.width = Math.round(w * dpr); canvas.height = Math.round(h * dpr);
		}
		const g = canvas.getContext("2d");
		g.setTransform(dpr, 0, 0, dpr, 0, 0);
		g.clearRect(0, 0, w, h);
		const t = now(), head = w * 0.3, pxs = w / seconds;
		const t0 = t - head / pxs, t1 = t0 + seconds;
		const keyH = h / (hi - lo + 1);
		const y = k => h - (k - lo + 1) * keyH;
		// the Cs, and the seconds
		g.fillStyle = colors.line;
		for (let k = Math.ceil(lo / 12) * 12; k <= hi; k += 12) g.fillRect(0, Math.round(y(k) + keyH), w, 1);
		g.globalAlpha = 0.5;
		for (let s = Math.ceil(t0); s < t1; s++) g.fillRect(Math.round((s - t0) * pxs), 0, 1, h);
		g.globalAlpha = 1;
		for (let i = firstFrom(t0); i < notes.length && notes[i][0] < t1; i++) {
			const [s, e, k, ch, v] = notes[i];
			if (e < t0) continue;
			const on = s <= t && t < e, quiet = muted.has(ch);
			const hh = hue(ch);
			const light = on ? 62 : 50;
			g.fillStyle = hh === null ? colors.muted : `hsl(${hh} 70% ${light}%)`;
			g.globalAlpha = quiet ? 0.15 : on ? 1 : 0.45 + 0.4 * v / 127;
			const x = (s - t0) * pxs;
			g.fillRect(x, y(k) + 0.5, Math.max(2, (e - s) * pxs - 1), Math.max(1.5, keyH - 1));
		}
		g.globalAlpha = 1;
		g.fillStyle = colors.accent;
		g.fillRect(Math.round(head) - 1, 0, 2, h);
		time.textContent = `${clock(t)} / ${clock(d.duration)}`;
		requestAnimationFrame(draw);
	};
	requestAnimationFrame(draw);

	// a click or a drag moves the player there, the wheel zooms
	const timeAt = e => {
		const r = canvas.getBoundingClientRect();
		const t = now(), pxs = r.width / seconds;
		return Math.max(0, Math.min(d.duration, t + (e.clientX - r.left - r.width * 0.3) / pxs));
	};
	let dragging = false, last = 0;
	const seek = e => {
		const s = timeAt(e);
		pos = s; at = performance.now();
		if (performance.now() - last > 60) { last = performance.now(); api.message(p.name, {seek: s}); }
	};
	canvas.addEventListener("pointerdown", e => { dragging = true; canvas.setPointerCapture(e.pointerId); seek(e); });
	canvas.addEventListener("pointermove", e => { if (dragging) seek(e); });
	canvas.addEventListener("pointerup", e => { if (dragging) { dragging = false; last = 0; seek(e); } });
	canvas.addEventListener("wheel", e => {
		e.preventDefault();
		seconds = Math.max(2, Math.min(120, seconds * Math.exp(e.deltaY * 0.002)));
	}, {passive: false});

	return {
		el,
		show() {},
		update(st) {
			if (dragging) return;
			pos = st.pos; rate = st.rate; at = performance.now();
			muted = new Set(st.muted || []);
		},
		restyle,
	};
});
