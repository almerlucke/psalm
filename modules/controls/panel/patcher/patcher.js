// a modular patch: modules to add, move and take away, cables to draw between their ports
psalm.register("patcher", (p, api) => {
	"use strict";
	const kinds = p.data.kinds || [];
	const kindOf = name => kinds.find(k => k.name === name);
	const el = (tag, cls, text) => { const e = document.createElement(tag); if (cls) e.className = cls; if (text !== undefined) e.textContent = text; return e; };
	const send = m => api.message(p.name, m);

	const root = el("div", "pt");

	// --- the bar: add a module, the patches ---
	const bar = el("div", "pt-bar");
	const kindSel = el("select", "pt-kind");
	const none = el("option", "", "add a module…"); none.value = ""; kindSel.append(none);
	const groups = {};
	for (const k of kinds) {
		if (!groups[k.group]) { groups[k.group] = el("optgroup"); groups[k.group].label = k.group || "Modules"; kindSel.append(groups[k.group]); }
		const o = el("option", "", k.name); o.value = k.name; groups[k.group].append(o);
	}
	const presetSel = el("select", "pt-preset"); presetSel.hidden = true;
	const addBtn = el("button", "", "Add"); addBtn.disabled = true;
	kindSel.addEventListener("change", () => {
		const k = kindOf(kindSel.value);
		addBtn.disabled = !k;
		presetSel.innerHTML = "";
		presetSel.hidden = !(k && k.presets && k.presets.length);
		if (k && k.presets) for (const n of k.presets) { const o = el("option", "", n); o.value = n; presetSel.append(o); }
	});
	let placed = 0;
	addBtn.addEventListener("click", () => {
		if (!kindSel.value) return;
		const r = area.getBoundingClientRect();
		const x = area.scrollLeft + 40 + (placed % 6) * 30, y = area.scrollTop + 40 + (placed % 6) * 30;
		placed++;
		send({add: kindSel.value, preset: presetSel.hidden ? "" : presetSel.value, pos: [x, y]});
	});
	const sep = el("span", "pt-sep");
	const patchSel = el("select", "pt-patches");
	const nameIn = el("input", "pt-name"); nameIn.placeholder = "patch name"; nameIn.maxLength = 64;
	const saveBtn = el("button", "", "Save"), delBtn = el("button", "", "Delete"), clearBtn = el("button", "", "Clear");
	saveBtn.title = "keep the patch under the name: its modules, cables and settings";
	clearBtn.title = "take every module away";
	patchSel.addEventListener("change", () => { if (patchSel.value) send({load: patchSel.value}); });
	const save = () => { const n = nameIn.value.trim(); if (n) send({save: n}); };
	saveBtn.addEventListener("click", save);
	nameIn.addEventListener("keydown", e => { if (e.key === "Enter") save(); });
	delBtn.addEventListener("click", () => { if (patchSel.value && confirm(`Delete the patch "${patchSel.value}"?`)) send({delete: patchSel.value}); });
	clearBtn.addEventListener("click", () => { if (confirm("Take every module away?")) send({clear: true}); });
	const note = el("span", "pt-note");
	bar.append(kindSel, presetSel, addBtn, sep, patchSel, nameIn, saveBtn, delBtn, clearBtn, note);

	// --- the patch: modules over a layer of cables ---
	const area = el("div", "pt-area");
	const svgNS = "http://www.w3.org/2000/svg";
	const svg = document.createElementNS(svgNS, "svg");
	svg.setAttribute("class", "pt-cables");
	area.append(svg);
	const inspector = el("div", "pt-inspector");
	root.append(bar, area, inspector);

	let st = {nodes: [], cables: [], patches: [], selected: ""}, shownPatches = "";
	const nodes = {};       // id -> {el, data}
	let dragging = null;    // a module moved: its id
	let wiring = null;      // a cable drawn: {from, out, type, path}
	let picked = "";        // the cable chosen (its id)
	let pickNew = null;     // a mod cable just drawn: chosen when it comes (the cables there before)

	const portEl = (n, dir, i) => nodes[n] && nodes[n].el.querySelector(`.pt-port[data-dir="${dir}"][data-i="${i}"] .pt-dot`);
	const center = dot => {
		const a = area.getBoundingClientRect(), r = dot.getBoundingClientRect();
		return [r.left + r.width / 2 - a.left + area.scrollLeft, r.top + r.height / 2 - a.top + area.scrollTop];
	};
	const curve = (x1, y1, x2, y2) => {
		const dx = Math.max(40, Math.abs(x2 - x1) / 2);
		return `M ${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`;
	};

	function drawCables() {
		svg.setAttribute("width", area.scrollWidth); svg.setAttribute("height", area.scrollHeight);
		for (const old of [...svg.querySelectorAll("path.pt-cable")]) old.remove();
		for (const c of st.cables) {
			const a = portEl(c.from, "out", c.out), b = portEl(c.to, "in", c.type === "mod" ? modIndex(c.to) : c.in);
			if (!a || !b) continue;
			const [x1, y1] = center(a), [x2, y2] = center(b);
			const path = document.createElementNS(svgNS, "path");
			path.setAttribute("d", curve(x1, y1, x2, y2));
			path.setAttribute("class", `pt-cable ${c.type}` + (c.id === picked ? " picked" : ""));
			path.dataset.id = c.id;
			const t = document.createElementNS(svgNS, "title");
			t.textContent = c.type === "mod" ? `${c.from} → ${c.to} ${c.key}` : `${c.from} → ${c.to}`;
			path.append(t);
			path.addEventListener("pointerdown", e => { e.stopPropagation(); pick(c.id); });
			svg.append(path);
		}
	}
	const modIndex = id => { const n = st.nodes.find(x => x.id === id); return n ? (n.ins || []).findIndex(x => x.type === "mod") : -1; };

	function makeNode(n) {
		const box = el("div", "pt-node");
		box.dataset.id = n.id;
		const head = el("div", "pt-head");
		const title = el("span", "pt-title");
		const x = el("button", "pt-x", "×"); x.title = "take it away";
		if (n.id === "out") x.hidden = true;
		head.append(title, x);
		const ports = el("div", "pt-ports");
		const ins = el("div", "pt-ins"), outs = el("div", "pt-outs");
		ports.append(ins, outs);
		const texts = el("div", "pt-texts"); // its settings of text: typed here, sent on Enter or leaving
		box.append(head, ports, texts);
		x.addEventListener("pointerdown", e => e.stopPropagation());
		x.addEventListener("click", () => send({remove: n.id}));
		// moving it by its head; a tap chooses it (its settings below)
		head.addEventListener("pointerdown", e => {
			if (e.button !== 0) return;
			const startX = e.clientX, startY = e.clientY, ox = box.offsetLeft, oy = box.offsetTop;
			let moved = false;
			dragging = n.id;
			head.setPointerCapture(e.pointerId);
			const move = ev => {
				const nx = Math.max(0, ox + ev.clientX - startX), ny = Math.max(0, oy + ev.clientY - startY);
				if (Math.abs(ev.clientX - startX) + Math.abs(ev.clientY - startY) > 3) moved = true;
				box.style.left = nx + "px"; box.style.top = ny + "px";
				drawCables();
			};
			const up = () => {
				head.removeEventListener("pointermove", move);
				head.removeEventListener("pointerup", up);
				dragging = null;
				if (moved) send({move: n.id, pos: [box.offsetLeft, box.offsetTop]});
				else send({select: n.id});
			};
			head.addEventListener("pointermove", move);
			head.addEventListener("pointerup", up);
			e.preventDefault();
		});
		box.addEventListener("pointerdown", e => { if (!e.target.closest(".pt-port")) send({select: n.id}); });
		area.append(box);
		return {el: box, title, ins, outs, texts, inputs: {}, sig: ""};
	}

	function portRow(n, dir, port, i) {
		const row = el("div", "pt-port " + port.type);
		row.dataset.dir = dir; row.dataset.i = i; row.dataset.node = n.id; row.dataset.type = port.type;
		const dot = el("span", "pt-dot"), name = el("span", "pt-pname", port.name);
		row.title = `${port.type} ${dir === "in" ? "input" : "output"}: ${port.name}`;
		if (dir === "in") row.append(dot, name); else row.append(name, dot);
		if (dir === "out") dot.addEventListener("pointerdown", e => startWire(e, n.id, i, port.type, dot));
		return row;
	}

	function updateNodes() {
		const seen = new Set();
		for (const n of st.nodes) {
			seen.add(n.id);
			const v = nodes[n.id] || (nodes[n.id] = makeNode(n));
			v.data = n;
			v.title.textContent = n.kind + (n.preset ? " · " + n.preset : "");
			const sig = JSON.stringify([n.ins, n.outs]);
			if (sig !== v.sig) {
				v.sig = sig;
				v.ins.innerHTML = ""; v.outs.innerHTML = "";
				(n.ins || []).forEach((port, i) => v.ins.append(portRow(n, "in", port, i)));
				(n.outs || []).forEach((port, i) => v.outs.append(portRow(n, "out", port, i)));
			}
			for (const t of n.texts || []) {
				let inp = v.inputs[t.key];
				if (!inp) {
					const row = el("label", "pt-text");
					row.append(el("span", "", t.label));
					const box = t.lines > 1; // a box of lines (a script): Ctrl+Enter, or leaving it, takes it
					inp = el(box ? "textarea" : "input");
					if (box) { inp.rows = t.lines; inp.wrap = "off"; }
					inp.spellcheck = false;
					inp.value = t.value;
					const commit = () => send({text: {id: n.id, key: t.key, value: inp.value}});
					inp.addEventListener("keydown", e => {
						if (box && e.key === "Tab") { e.preventDefault(); document.execCommand("insertText", false, "\t"); }
						if (e.key === "Enter" && (!box || e.ctrlKey || e.metaKey)) { e.preventDefault(); commit(); if (!box) inp.blur(); }
						e.stopPropagation();
					});
					inp.addEventListener("change", commit);
					inp.addEventListener("pointerdown", e => e.stopPropagation());
					row.append(inp);
					v.texts.append(row);
					v.inputs[t.key] = inp;
				}
				if (document.activeElement !== inp) inp.value = t.value;
				const err = n.errors && n.errors[t.key];
				inp.classList.toggle("bad", !!err);
				inp.title = err || "";
			}
			if (n.status !== undefined || v.status) {
				if (!v.status) { v.status = el("div", "pt-status"); v.el.append(v.status); }
				v.status.textContent = n.status || "";
				v.status.classList.toggle("bad", /^error/i.test(n.status || ""));
			}
			if (dragging !== n.id) { v.el.style.left = n.x + "px"; v.el.style.top = n.y + "px"; }
			v.el.classList.toggle("selected", st.selected === n.id);
		}
		for (const id of Object.keys(nodes)) if (!seen.has(id)) { nodes[id].el.remove(); delete nodes[id]; }
	}

	// --- drawing a cable: from an output to an input of its type ---
	function startWire(e, from, out, type, dot) {
		e.stopPropagation(); e.preventDefault();
		const path = document.createElementNS(svgNS, "path");
		path.setAttribute("class", "pt-cable drawing " + type);
		svg.append(path);
		wiring = {from, out, type, path};
		area.classList.add("wiring-" + type);
		const [x1, y1] = center(dot);
		const a = area.getBoundingClientRect();
		const move = ev => path.setAttribute("d", curve(x1, y1, ev.clientX - a.left + area.scrollLeft, ev.clientY - a.top + area.scrollTop));
		const up = ev => {
			window.removeEventListener("pointermove", move);
			window.removeEventListener("pointerup", up);
			path.remove();
			area.classList.remove("wiring-" + type);
			const target = document.elementFromPoint(ev.clientX, ev.clientY);
			const row = target && target.closest(".pt-port");
			wiring = null;
			if (!row || row.dataset.dir !== "in" || row.dataset.type !== type || row.dataset.node === from && type === "audio") return;
			if (type === "mod") pickNew = new Set(st.cables.map(c => c.id)); // to choose its setting
			send({connect: {from, out, to: row.dataset.node, in: Number(row.dataset.i)}});
		};
		window.addEventListener("pointermove", move);
		window.addEventListener("pointerup", up);
	}

	// --- a cable chosen: take it away; a mod cable: the setting it moves, over what part of its range ---
	function pick(id) { picked = id; drawCables(); showInspector(); }
	function showInspector() {
		inspector.innerHTML = "";
		const c = st.cables.find(x => x.id === picked);
		if (!c) { inspector.hidden = true; return; }
		inspector.hidden = false;
		const named = id => { const n = st.nodes.find(x => x.id === id); return n ? n.kind + (n.preset ? " · " + n.preset : "") : id; };
		inspector.append(el("span", "pt-label", `${c.type} cable: ${named(c.from)} → ${named(c.to)}`));
		if (c.type === "mod") {
			const to = st.nodes.find(n => n.id === c.to);
			const sel = el("select", "pt-setting");
			(to && to.settings || []).forEach((k, i) => { const o = el("option", "", to.labels[i]); o.value = k; sel.append(o); });
			sel.value = c.key;
			const range = (label, v) => {
				const r = el("input"); r.type = "range"; r.min = 0; r.max = 1; r.step = 0.01; r.value = v;
				const l = el("label", "pt-range"); l.append(el("span", "", label), r);
				return [l, r];
			};
			const [l1, lo] = range("from", c.lo), [l2, hi] = range("to", c.hi);
			const update = () => send({range: {id: c.id, key: sel.value, lo: Number(lo.value), hi: Number(hi.value)}});
			sel.addEventListener("change", update); lo.addEventListener("change", update); hi.addEventListener("change", update);
			inspector.append(el("span", "pt-label", "moves"), sel, l1, l2);
		}
		const del = el("button", "", "Take it away");
		del.addEventListener("click", () => { send({disconnect: c.id}); picked = ""; showInspector(); });
		inspector.append(del);
	}
	area.addEventListener("pointerdown", e => { if (e.target === area || e.target === svg) { picked = ""; drawCables(); showInspector(); } });
	window.addEventListener("keydown", e => {
		if ((e.key === "Delete" || e.key === "Backspace") && picked && !/INPUT|SELECT/.test(document.activeElement.tagName)) {
			send({disconnect: picked}); picked = ""; showInspector();
		}
	});

	function updatePatches() {
		const key = JSON.stringify(st.patches || []);
		if (key !== shownPatches) {
			shownPatches = key;
			patchSel.innerHTML = "";
			const o0 = el("option", "", "patches…"); o0.value = ""; patchSel.append(o0);
			for (const n of st.patches || []) { const o = el("option", "", n); o.value = n; patchSel.append(o); }
		}
		patchSel.value = (st.patches || []).includes(st.current) ? st.current : "";
		delBtn.disabled = !patchSel.value;
		if (document.activeElement !== nameIn && st.current) nameIn.value = st.current;
		for (const e of [patchSel, nameIn, saveBtn, delBtn]) e.hidden = !st.keeps;
		note.textContent = st.note || "";
	}

	new ResizeObserver(drawCables).observe(area);
	area.addEventListener("scroll", drawCables);
	return {
		el: root,
		show() {},
		update(s) {
			if (!s) return;
			const was = JSON.stringify(st);
			st = s;
			if (JSON.stringify(s) === was) return;
			updateNodes();
			if (picked && !st.cables.some(c => c.id === picked)) picked = "";
			if (pickNew) {
				const c = st.cables.find(c => c.type === "mod" && !pickNew.has(c.id));
				if (c) { picked = c.id; pickNew = null; }
			}
			drawCables();
			if (!inspector.contains(document.activeElement)) showInspector();
			updatePatches();
		},
		restyle: drawCables,
	};
});
