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
	const maxBtn = el("button", "pt-max-btn", "Full screen");
	maxBtn.title = "the patch over the whole screen (Esc: back)";
	bar.append(maxBtn, el("span", "pt-sep"), kindSel, presetSel, addBtn, sep, patchSel, nameIn, saveBtn, delBtn, clearBtn, note);

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
	const chosen = new Set(); // the modules chosen (shift-click, or a rectangle around them): moved together
	let dragging = new Set(); // the modules moved: their ids
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
		for (const old of [...svg.querySelectorAll("path.pt-cable:not(.drawing), path.pt-hit")]) old.remove();
		for (const c of st.cables) {
			const a = portEl(c.from, "out", c.out), b = portEl(c.to, "in", c.type === "mod" ? modIndex(c.to) : c.in);
			if (!a || !b) continue;
			const [x1, y1] = center(a), [x2, y2] = center(b);
			const d = curve(x1, y1, x2, y2);
			// a wide stroke no one sees takes hold of it (a cable is thin), the cable drawn over it
			const hit = document.createElementNS(svgNS, "path");
			hit.setAttribute("d", d);
			hit.setAttribute("class", "pt-hit");
			hit.dataset.id = c.id;
			const t = document.createElementNS(svgNS, "title");
			t.textContent = c.type === "mod" ? `${c.from} → ${c.to} ${c.key}` : `${c.from} → ${c.to}`;
			hit.append(t);
			const path = document.createElementNS(svgNS, "path");
			path.setAttribute("d", d);
			path.setAttribute("class", `pt-cable ${c.type}` + (c.id === picked ? " picked" : ""));
			path.dataset.id = c.id;
			for (const e of [hit, path]) e.addEventListener("pointerdown", ev => { ev.stopPropagation(); pick(c.id); });
			svg.append(hit, path);
		}
	}
	const modIndex = id => { const n = st.nodes.find(x => x.id === id); return n ? (n.ins || []).findIndex(x => x.type === "mod") : -1; };

	function makeNode(n) {
		const box = el("div", "pt-node");
		box.dataset.id = n.id;
		const head = el("div", "pt-head");
		const title = el("span", "pt-title");
		head.append(title);
		const ports = el("div", "pt-ports");
		const ins = el("div", "pt-ins"), outs = el("div", "pt-outs");
		ports.append(ins, outs);
		const texts = el("div", "pt-texts"); // its settings of text: typed here, sent on Enter or leaving
		box.append(head, ports, texts);
		// moving it by its head (with the other modules chosen, when it is one of them); a tap chooses
		// it (its settings below), with shift (or ctrl, cmd) it is added to the chosen or taken out
		head.addEventListener("pointerdown", e => {
			if (e.button !== 0) return;
			e.preventDefault(); e.stopPropagation();
			if (e.shiftKey || e.ctrlKey || e.metaKey) {
				if (chosen.has(n.id)) { chosen.delete(n.id); showChosen(); return; }
				chosen.add(n.id);
			} else if (!chosen.has(n.id)) {
				chosen.clear(); chosen.add(n.id);
			}
			showChosen();
			const startX = e.clientX, startY = e.clientY;
			const moving = [...chosen].filter(id => nodes[id]).map(id => ({id, el: nodes[id].el, x: nodes[id].el.offsetLeft, y: nodes[id].el.offsetTop}));
			// the group stays together at the edge: none of them goes left of or above the patch
			const minX = Math.min(...moving.map(m => m.x)), minY = Math.min(...moving.map(m => m.y));
			let moved = false;
			dragging = new Set(moving.map(m => m.id));
			head.setPointerCapture(e.pointerId);
			const move = ev => {
				const dx = Math.max(-minX, ev.clientX - startX), dy = Math.max(-minY, ev.clientY - startY);
				if (Math.abs(ev.clientX - startX) + Math.abs(ev.clientY - startY) > 3) moved = true;
				for (const m of moving) { m.el.style.left = m.x + dx + "px"; m.el.style.top = m.y + dy + "px"; }
				drawCables();
			};
			const up = ev => {
				head.removeEventListener("pointermove", move);
				head.removeEventListener("pointerup", up);
				dragging = new Set();
				if (moved) {
					for (const m of moving) send({move: m.id, pos: [m.el.offsetLeft, m.el.offsetTop]});
				} else if (!(ev.shiftKey || ev.ctrlKey || ev.metaKey)) {
					tapped(n.id);
				}
			};
			head.addEventListener("pointermove", move);
			head.addEventListener("pointerup", up);
		});
		// a tap on it (not on a port or a setting typed on it) chooses it; a double tap opens its side
		// panel, or closes it
		const onIt = e => !e.target.closest(".pt-port, input, textarea, button, select") && !(e.shiftKey || e.ctrlKey || e.metaKey);
		box.addEventListener("click", e => { if (onIt(e) && !e.target.closest(".pt-head")) tapped(n.id); });
		box.addEventListener("dblclick", e => { if (onIt(e)) doubleTapped(n.id); });
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
				if (t.lines > 1) continue; // a box of lines (a script): in the side panel
				let inp = v.inputs[t.key];
				if (!inp) {
					const row = el("label", "pt-text");
					row.append(el("span", "pt-text-label", t.label));
					inp = el("input");
					inp.spellcheck = false;
					inp.value = t.value;
					const commit = () => send({text: {id: n.id, key: t.key, value: inp.value}});
					inp.addEventListener("keydown", e => {
						if (e.key === "Enter") { e.preventDefault(); commit(); inp.blur(); }
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
			if (!dragging.has(n.id)) { v.el.style.left = n.x + "px"; v.el.style.top = n.y + "px"; }
			v.el.classList.toggle("selected", chosen.has(n.id));
		}
		for (const id of Object.keys(nodes)) if (!seen.has(id)) { nodes[id].el.remove(); delete nodes[id]; }
		for (const id of [...chosen]) if (!seen.has(id)) chosen.delete(id);
	}

	// the modules chosen shown; choosing them lets go of a cable picked (what Delete takes away)
	function showChosen() {
		for (const [id, v] of Object.entries(nodes)) v.el.classList.toggle("selected", chosen.has(id));
		if (chosen.size && picked) { picked = ""; drawCables(); showInspector(); }
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
	function pick(id) { picked = id; chosen.clear(); showChosen(); drawCables(); showInspector(); } // a cable, or modules: what Delete takes away
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
	area.addEventListener("pointerdown", e => {
		if (e.button !== 0 || (e.target !== area && e.target !== svg)) return;
		const adding = e.shiftKey || e.ctrlKey || e.metaKey;
		const a = area.getBoundingClientRect();
		const at = ev => [ev.clientX - a.left + area.scrollLeft, ev.clientY - a.top + area.scrollTop];
		const [x0, y0] = at(e);
		const before = new Set(chosen);
		let rect = null;
		area.setPointerCapture(e.pointerId);
		const move = ev => {
			const [x1, y1] = at(ev);
			if (!rect && Math.abs(x1 - x0) + Math.abs(y1 - y0) < 4) return;
			if (!rect) { rect = el("div", "pt-marquee"); area.append(rect); }
			const l = Math.min(x0, x1), t = Math.min(y0, y1), w = Math.abs(x1 - x0), h = Math.abs(y1 - y0);
			Object.assign(rect.style, {left: l + "px", top: t + "px", width: w + "px", height: h + "px"});
			chosen.clear();
			if (adding) for (const id of before) chosen.add(id);
			for (const [id, v] of Object.entries(nodes)) {
				const b = v.el;
				if (b.offsetLeft < l + w && b.offsetLeft + b.offsetWidth > l && b.offsetTop < t + h && b.offsetTop + b.offsetHeight > t) chosen.add(id);
			}
			showChosen();
		};
		const up = () => {
			area.removeEventListener("pointermove", move);
			area.removeEventListener("pointerup", up);
			if (rect) { rect.remove(); return; }
			if (!adding) { chosen.clear(); showChosen(); } // a tap on the patch: nothing chosen
			picked = ""; drawCables(); showInspector();
		};
		area.addEventListener("pointermove", move);
		area.addEventListener("pointerup", up);
	});
	const typing = () => /INPUT|SELECT|TEXTAREA/.test(document.activeElement.tagName) || document.activeElement.isContentEditable;
	window.addEventListener("keydown", e => {
		if (e.key === "Escape") {
			if (editing) closeEditor();
			else if (picked || chosen.size) { picked = ""; chosen.clear(); showChosen(); drawCables(); showInspector(); }
			else if (maxed) setMax(false);
			return;
		}
		if ((e.key === "Delete" || e.key === "Backspace") && !typing()) {
			if (picked) { send({disconnect: picked}); picked = ""; showInspector(); return; }
			const gone = [...chosen].filter(id => id !== "out");
			if (!gone.length || (gone.length > 1 && !confirm(`Take ${gone.length} modules away?`))) return;
			for (const id of gone) send({remove: id});
			chosen.clear();
		}
	});

	// --- the patch over the whole screen (the browser's full screen when it may) ---
	let maxed = false;
	function setMax(on) {
		maxed = on;
		root.classList.toggle("pt-max", on);
		maxBtn.textContent = on ? "Exit full screen" : "Full screen";
		if (!on && document.fullscreenElement) document.exitFullscreen().catch(() => {});
		requestAnimationFrame(drawCables);
	}
	maxBtn.addEventListener("click", () => {
		if (maxed) { setMax(false); return; }
		setMax(true);
		if (document.documentElement.requestFullscreen) document.documentElement.requestFullscreen().catch(() => {});
	});
	document.addEventListener("fullscreenchange", () => { if (!document.fullscreenElement && maxed) setMax(false); });

	// --- the side panel of a module: its script (a box of lines) and its settings (its editor,
	// served by the patcher's hidden mount), its error or what it says below the script ---
	const drawer = el("div", "pt-drawer"); drawer.hidden = true;
	const grip = el("div", "pt-drawer-grip"); grip.title = "drag to make it wider or narrower";
	const dHead = el("div", "pt-drawer-head");
	const dTitle = el("span", "pt-drawer-title");
	const runBtn = el("button", "", "Run"); runBtn.title = "take the script (Ctrl+Enter)";
	const closeBtn = el("button", "pt-x", "×"); closeBtn.title = "close (Esc): a script typed is taken";
	dHead.append(dTitle, runBtn, closeBtn);
	const dCode = el("div", "pt-drawer-code");
	const dText = el("textarea", "pt-drawer-text"); dText.spellcheck = false; dText.wrap = "off";
	const dStatus = el("div", "pt-status");
	dCode.append(dText, dStatus);
	const dSettings = el("div", "pt-drawer-settings");
	const frame = el("iframe", "pt-drawer-frame"); frame.title = "settings";
	api.frame(frame); // its height, the theme, learn mode: as the page's own frames
	dSettings.append(frame);
	drawer.append(grip, dHead, dCode, dSettings);
	root.append(drawer);
	let editing = null;  // {id, key}: the module in the side panel, the key of its script ("" none)
	let framed = "";     // the module whose settings the frame shows
	const textOf = (id, key) => { const n = st.nodes.find(x => x.id === id); const t = n && (n.texts || []).find(x => x.key === key); return t ? t.value : null; };
	const nodeOf = id => st.nodes.find(x => x.id === id);
	// a tap chooses a module; with the side panel open, the panel shows it
	let taps = [false, false]; // the last two taps: whether each brought the panel to its module
	function tapped(id) {
		chosen.clear(); chosen.add(id); showChosen();
		const follow = !!editing && editing.id !== id;
		taps = [taps[1], follow];
		if (follow) openSide(id);
	}
	// a double tap opens the side panel of a module, or closes it; not when its first tap brought the
	// panel to it (the tap before the last: a double tap is two taps, then this)
	function doubleTapped(id) {
		if (taps[0]) { taps = [false, false]; return; }
		if (editing && editing.id === id) closeEditor(); else openSide(id);
	}
	function openSide(id) {
		const n = nodeOf(id);
		if (!n) return;
		const box = (n.texts || []).find(t => t.lines > 1);
		if (!box && !n.page) { if (editing) closeEditor(); return; } // nothing to show
		if (editing) closeEditor();
		editing = {id, key: box ? box.key : ""};
		dTitle.textContent = n.kind + (n.preset ? " · " + n.preset : "");
		dCode.hidden = runBtn.hidden = !box;
		if (box) dText.value = box.value;
		dSettings.hidden = !n.page;
		drawer.classList.toggle("both", !!box && !!n.page);
		send({select: n.page ? id : ""}); // the frame follows once it is chosen (update)
		drawer.hidden = false;
		roomForDrawer();
		updateEditor();
		if (box) dText.focus();
	}
	function commitEditor() {
		if (!editing || !editing.key) return;
		send({text: {id: editing.id, key: editing.key, value: dText.value}});
	}
	function closeEditor() {
		if (editing && editing.key && dText.value !== textOf(editing.id, editing.key)) commitEditor();
		editing = null;
		drawer.hidden = true;
		frame.src = "about:blank"; frame.style.height = "0"; framed = "";
		send({select: ""});
		roomForDrawer();
	}
	// over the whole screen the patch goes beside the side panel, not under it
	function roomForDrawer() {
		root.style.setProperty("--drawer-width", drawer.hidden ? "0px" : drawer.offsetWidth + "px");
		requestAnimationFrame(drawCables);
	}
	function updateEditor() {
		if (!editing) return;
		const n = nodeOf(editing.id);
		if (!n) { closeEditor(); return; }
		if (n.page && st.selected === n.id && framed !== n.id) { // its page is the one served now
			framed = n.id;
			frame.src = "mount/" + encodeURIComponent(p.data.mount) + "/?embed&id=" + encodeURIComponent(n.id) + "&theme=" + encodeURIComponent(api.theme());
		}
		if (!editing.key) return;
		const err = n.errors && n.errors[editing.key];
		const say = err ? "error: " + err : n.status || "";
		dStatus.textContent = say;
		dStatus.classList.toggle("bad", /^error/i.test(say));
		dText.classList.toggle("bad", !!err);
	}
	runBtn.addEventListener("click", commitEditor);
	closeBtn.addEventListener("click", closeEditor);
	dText.addEventListener("keydown", e => {
		e.stopPropagation();
		if (e.key === "Tab") { e.preventDefault(); document.execCommand("insertText", false, "\t"); }
		else if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) { e.preventDefault(); commitEditor(); }
		else if (e.key === "Escape") { e.preventDefault(); closeEditor(); }
	});
	grip.addEventListener("pointerdown", e => {
		e.preventDefault();
		grip.setPointerCapture(e.pointerId);
		const right = drawer.getBoundingClientRect().right;
		const move = ev => { drawer.style.width = Math.max(280, Math.min(right - 40, right - ev.clientX)) + "px"; roomForDrawer(); };
		const up = () => { grip.removeEventListener("pointermove", move); grip.removeEventListener("pointerup", up); };
		grip.addEventListener("pointermove", move);
		grip.addEventListener("pointerup", up);
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
			updateEditor();
		},
		restyle: drawCables,
	};
});
