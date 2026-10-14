// a note editor: a piano roll of a loop to draw notes on. The notes are [start, length, key,
// velocity] (start and length in beats). A click on an empty place puts a note (drag to make it
// longer), a drag on a note moves it, a drag on its right end resizes it, a right click takes it
// away; the page keeps its own copy while you work and sends all of it when you let go
psalm.register("noteroll", (p, api) => {
	const canvas = document.createElement("canvas");
	const bar = document.createElement("div"); bar.className = "nr-bar";
	const hint = document.createElement("div"); hint.className = "nr-hint";
	hint.textContent = "click to put a note, drag its end to make it longer, drag it to move it, right click to take it away; the wheel scrolls the keys";
	const el = api.labelled(p, canvas, hint);
	el.classList.add("noteroll");
	canvas.parentNode.insertBefore(bar, canvas);

	let notes = [], length = 8, pos = 0, bps = 2, at = performance.now();
	let selected = -1, snap = 0.25, lastLen = 1, lastVel = 0.8;
	let lo = 48, rows = 36; // the lowest key shown, how many keys
	let first = true; // the first state: the keys shown are where the notes are
	let hold = 0; // until when the page's own notes stay (the server's come back after a send)
	let drag = null;

	// the bar: length, snap, velocity of the one chosen, clear
	const mk = (tag, props, ...kids) => { const e = document.createElement(tag); Object.assign(e, props); e.append(...kids); return e; };
	const lenIn = mk("input", {type: "number", min: 1, max: 64, step: 1, value: length, title: "how many beats the loop has"});
	const snapSel = mk("select", {title: "what the notes snap to"});
	for (const [name, v] of [["1 beat", 1], ["1/2", 0.5], ["1/4 (16th)", 0.25], ["1/8", 0.125], ["1/3 (triplet)", 1 / 3], ["1/6", 1 / 6]]) {
		const o = mk("option", {value: v, textContent: name}); if (v === snap) o.selected = true; snapSel.append(o);
	}
	const velIn = mk("input", {type: "range", min: 1, max: 127, step: 1, value: 100, title: "the velocity of the note chosen"});
	const velLab = mk("label", {}, "Velocity ", velIn);
	const clear = mk("button", {textContent: "Clear", title: "take every note away"});
	const selInfo = mk("span", {className: "nr-sel"});
	bar.append(mk("label", {}, "Beats ", lenIn), mk("label", {}, "Snap ", snapSel), velLab, clear, selInfo);

	const send = () => {
		hold = performance.now() + 700;
		api.message(p.name, {set: {length, notes}});
	};
	lenIn.addEventListener("change", () => {
		length = Math.max(1, Math.min(64, Math.round(Number(lenIn.value)) || 1)); lenIn.value = length;
		notes = notes.filter(n => n[0] < length); selected = -1; send();
	});
	snapSel.addEventListener("change", () => { snap = Number(snapSel.value); });
	velIn.addEventListener("input", () => {
		if (selected < 0) return;
		notes[selected][3] = lastVel = Number(velIn.value) / 127; send();
	});
	clear.addEventListener("click", () => { notes = []; selected = -1; send(); });

	const name = k => ["C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"][k % 12] + (Math.floor(k / 12) - 1);
	const black = k => [1, 3, 6, 8, 10].includes(k % 12);

	let colors = {};
	const restyle = () => { colors = {bg: api.css("track"), ink: api.css("ink"), muted: api.css("muted"), line: api.css("line"), accent: api.css("accent")}; };
	restyle();

	const gutter = 52;
	const geom = () => {
		const w = canvas.clientWidth, h = canvas.clientHeight;
		return {w, h, pxb: (w - gutter) / length, rowH: h / rows};
	};
	const keyAt = (g, y) => lo + rows - 1 - Math.floor(y / g.rowH);
	const yOf = (g, k) => (lo + rows - 1 - k) * g.rowH;
	const snapTo = b => Math.round(b / snap) * snap;
	const find = (g, x, y) => { // the note at a place (the last drawn is on top)
		const k = keyAt(g, y), b = (x - gutter) / g.pxb;
		for (let i = notes.length - 1; i >= 0; i--) if (notes[i][2] === k && b >= notes[i][0] && b <= notes[i][0] + notes[i][1]) return i;
		return -1;
	};
	const head = () => { const t = (performance.now() - at) / 1000; return ((pos + bps * t) % length + length) % length; };

	const draw = () => {
		if (!canvas.isConnected) return;
		const g = geom(), dpr = window.devicePixelRatio || 1;
		if (canvas.width !== Math.round(g.w * dpr) || canvas.height !== Math.round(g.h * dpr)) {
			canvas.width = Math.round(g.w * dpr); canvas.height = Math.round(g.h * dpr);
		}
		const c = canvas.getContext("2d");
		c.setTransform(dpr, 0, 0, dpr, 0, 0);
		c.clearRect(0, 0, g.w, g.h);
		const hd = head();
		// the keys, and the rows
		c.font = "10px sans-serif"; c.textBaseline = "middle";
		for (let k = lo; k < lo + rows; k++) {
			const y = yOf(g, k);
			c.globalAlpha = black(k) ? 0.5 : 0.18; c.fillStyle = colors.line; c.fillRect(gutter, y, g.w - gutter, g.rowH);
			c.globalAlpha = 1;
			c.fillStyle = colors.ink; c.globalAlpha = 0.8;
			c.fillRect(0, y + 0.5, gutter - 2, g.rowH - 1); // the white key under all
			if (black(k)) { c.fillStyle = colors.bg; c.globalAlpha = 0.85; c.fillRect(0, y + 0.5, gutter * 0.62, g.rowH - 1); }
			c.globalAlpha = 1;
			if (k % 12 === 0) { c.fillStyle = colors.bg; c.fillText(name(k), gutter - 24, y + g.rowH / 2); }
		}
		// the beats and the bars
		for (let b = 0; b <= length; b++) {
			c.globalAlpha = b % 4 === 0 ? 0.9 : 0.4; c.fillStyle = colors.line;
			c.fillRect(Math.round(gutter + b * g.pxb), 0, 1, g.h);
		}
		c.globalAlpha = 1;
		notes.forEach((n, i) => {
			const [s, l, k, v] = n;
			if (k < lo || k >= lo + rows) return;
			const x = gutter + s * g.pxb, w = Math.max(3, l * g.pxb - 1), y = yOf(g, k);
			const on = s <= hd && hd < s + l;
			c.globalAlpha = on ? 1 : 0.5 + 0.4 * v;
			c.fillStyle = colors.accent; c.fillRect(x, y + 1, w, g.rowH - 2);
			c.globalAlpha = 1;
			if (i === selected) { c.strokeStyle = colors.ink; c.lineWidth = 2; c.strokeRect(x + 1, y + 2, w - 2, g.rowH - 4); }
		});
		c.fillStyle = colors.accent; c.fillRect(Math.round(gutter + hd * g.pxb) - 1, 0, 2, g.h);
		selInfo.textContent = selected >= 0 && notes[selected] ? `${name(notes[selected][2])}  ${+notes[selected][0].toFixed(3)} + ${+notes[selected][1].toFixed(3)}` : "";
		requestAnimationFrame(draw);
	};
	requestAnimationFrame(draw);

	const where = e => { const r = canvas.getBoundingClientRect(); return [e.clientX - r.left, e.clientY - r.top]; };
	canvas.addEventListener("contextmenu", e => {
		e.preventDefault();
		const [x, y] = where(e), i = find(geom(), x, y);
		if (i >= 0) { notes.splice(i, 1); selected = -1; send(); }
	});
	canvas.addEventListener("pointerdown", e => {
		if (e.button !== 0) return;
		const g = geom(), [x, y] = where(e);
		if (x < gutter) return;
		let i = find(g, x, y);
		const b = (x - gutter) / g.pxb;
		if (i < 0) { // a new note: dragging makes it longer
			const s = Math.max(0, Math.min(snapTo(b - snap / 2 + 1e-9), length - snap));
			notes.push([s, Math.min(lastLen, length - s), keyAt(g, y), lastVel]);
			i = notes.length - 1;
			drag = {mode: "resize", i, moved: false, fresh: true};
		} else {
			const n = notes[i], edge = Math.min(8, n[1] * g.pxb / 3);
			drag = (gutter + (n[0] + n[1]) * g.pxb - x <= edge) ? {mode: "resize", i, moved: false}
				: {mode: "move", i, dx: b - n[0], moved: false};
			lastVel = n[3];
		}
		selected = i; velIn.value = Math.round(notes[i][3] * 127);
		canvas.setPointerCapture(e.pointerId);
	});
	canvas.addEventListener("pointermove", e => {
		const g = geom(), [x, y] = where(e);
		if (!drag) { // the cursor at a note's end
			const i = find(g, x, y);
			canvas.style.cursor = i >= 0 && gutter + (notes[i][0] + notes[i][1]) * g.pxb - x <= Math.min(8, notes[i][1] * g.pxb / 3) ? "ew-resize" : i >= 0 ? "grab" : "crosshair";
			return;
		}
		const n = notes[drag.i], b = (x - gutter) / g.pxb;
		if (drag.mode === "resize") {
			const l = Math.max(snap, Math.min(snapTo(b - n[0] + snap / 2 - 1e-9), length - n[0]));
			if (l !== n[1]) { n[1] = l; drag.moved = true; }
		} else {
			const s = Math.max(0, Math.min(snapTo(b - drag.dx), length - Math.min(n[1], snap)));
			const k = Math.max(0, Math.min(127, keyAt(g, y)));
			if (s !== n[0] || k !== n[2]) { n[0] = s; n[2] = k; drag.moved = true; n[1] = Math.min(n[1], length - s); }
		}
	});
	const up = () => {
		if (!drag) return;
		const n = notes[drag.i];
		if (drag.mode === "resize" && n && (drag.moved || drag.fresh)) lastLen = n[1];
		if (drag.moved || drag.fresh) send();
		drag = null;
	};
	canvas.addEventListener("pointerup", up);
	canvas.addEventListener("pointercancel", up);
	canvas.addEventListener("wheel", e => {
		e.preventDefault();
		lo = Math.max(0, Math.min(128 - rows, lo + (e.deltaY < 0 ? 2 : -2)));
	}, {passive: false});
	// Delete takes the note chosen away (when the roll has the focus)
	canvas.tabIndex = 0;
	canvas.addEventListener("keydown", e => {
		if ((e.key === "Delete" || e.key === "Backspace") && selected >= 0) { e.preventDefault(); notes.splice(selected, 1); selected = -1; send(); }
	});

	return {
		el,
		show() {},
		update(st) {
			pos = st.pos; bps = st.bps; at = performance.now();
			if (first) {
				first = false;
				if (st.notes.length) {
					const ks = st.notes.map(n => n[2]);
					lo = Math.max(0, Math.min(128 - rows, Math.floor((Math.min(...ks) + Math.max(...ks)) / 2) - rows / 2));
				}
			}
			if (drag || performance.now() < hold) return; // our own notes until the server has them
			const same = st.length === length && st.notes.length === notes.length && st.notes.every((n, i) => n.every((v, j) => Math.abs(v - notes[i][j]) < 1e-6));
			if (!same) { // somewhere else (a patch loaded): theirs, the chosen one by where it was
				const was = selected >= 0 ? notes[selected] : null;
				notes = st.notes.map(n => n.slice()); length = st.length; lenIn.value = length;
				selected = was ? notes.findIndex(n => n[0] === was[0] && n[2] === was[2]) : -1;
			}
		},
		restyle,
	};
});
