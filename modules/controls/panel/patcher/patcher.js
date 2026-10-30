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
	const logBtn = el("button", "pt-log-btn", "Log");
	logBtn.title = "the log of the patch: what the scripts log, a line each with the time and the module";
	const lanesBtn = el("button", "pt-lanes-btn", "Lanes");
	lanesBtn.title = "the lanes of the automation modules, over the bars of the song: draw them, a mod cable from each moves a setting";
	const gridBtn = el("button", "pt-grid-btn", "Sections");
	gridBtn.title = "the sections of the song as a grid: a pattern for every part in every section, to edit, vary and copy";
	bar.append(maxBtn, logBtn, lanesBtn, gridBtn, el("span", "pt-sep"), kindSel, presetSel, addBtn, sep, patchSel, nameIn, saveBtn, delBtn, clearBtn, note);

	// --- the patch: modules over a layer of cables ---
	const area = el("div", "pt-area");
	const svgNS = "http://www.w3.org/2000/svg";
	const svg = document.createElementNS(svgNS, "svg");
	svg.setAttribute("class", "pt-cables");
	area.append(svg);
	const inspector = el("div", "pt-inspector");
	// toasts: an error of a module, in a corner over the page, gone by itself after a while (or when tapped); the same error is shown once
	// until the module has none
	const toasts = (() => {
		const box = el("div", "pt-toasts"); box.setAttribute("role", "alert");
		document.body.append(box);
		const live = {}; // module id -> the errors shown for it
		const show = (title, text) => {
			const t = el("div", "pt-toast");
			t.append(el("b", "", title), el("span", "", text));
			const close = () => t.remove();
			t.addEventListener("click", close);
			setTimeout(close, 12000);
			box.append(t);
			while (box.children.length > 5) box.firstChild.remove();
		};
		return {
			watch(n, errors) {
				const was = live[n.id] || [];
				for (const e of errors) if (!was.includes(e)) show(n.kind + " " + n.id, e.replace(/^error:\s*/i, ""));
				if (errors.length) live[n.id] = errors; else delete live[n.id];
			},
			forget(id) { delete live[id]; },
		};
	})();

	// the log of the patch: opened from the bar, over the patch, the newest line below
	const logBox = el("div", "pt-log"); logBox.hidden = true;
	const logText = el("textarea", "pt-log-text"); logText.readOnly = true; logText.spellcheck = false;
	const logClear = el("button", "", "Clear log");
	logBox.append(logText, logClear);
	let logOpen = false, logLines = 0;
	const showLogButton = () => { logBtn.textContent = logLines > 0 ? `Log (${logLines})` : "Log"; logBtn.classList.toggle("on", logOpen); };
	logBtn.addEventListener("click", () => { logOpen = !logOpen; logBox.hidden = !logOpen; showLogButton(); logText.scrollTop = logText.scrollHeight; });
	logClear.addEventListener("click", () => send({clearLog: true}));
	const showJournal = (text, lines) => {
		logLines = lines || 0;
		showLogButton();
		if (logText.value === text) return;
		const atEnd = logText.scrollTop + logText.clientHeight >= logText.scrollHeight - 4;
		logText.value = text;
		if (atEnd) logText.scrollTop = logText.scrollHeight;
	};
	// the lanes of the automation modules: over the whole width at the bottom of the patch
	const lanes = psalm.lanesView(send);
	lanes.onToggle = on => lanesBtn.classList.toggle("on", on);
	lanesBtn.addEventListener("click", () => lanes.toggle());
	const grid = psalm.sectionGrid(send);
	grid.onToggle = on => gridBtn.classList.toggle("on", on);
	gridBtn.addEventListener("click", () => { if (!grid.isOpen && lanes.isOpen) lanes.toggle(); grid.toggle(); });
	lanes.onToggle = on => { lanesBtn.classList.toggle("on", on); if (on && grid.isOpen) grid.toggle(); };
	const stage = el("div", "pt-stage");
	stage.append(area, lanes.el, grid.el);
	root.append(bar, logBox, stage, inspector);

	let st = {nodes: [], cables: [], patches: [], selected: ""}, shownPatches = "";
	const nodes = {};       // id -> {el, data}
	const chosen = new Set(); // the modules chosen (shift-click, or a rectangle around them): moved together
	let dragging = new Set(); // the modules moved: their ids
	let wiring = null;      // a cable drawn: {from, out, type, path}
	let picked = "";        // the cable chosen (its id)
	let pickNew = null;     // a mod cable just drawn: chosen when it comes (the cables there before)
	let pastes = -1;        // the pastes seen: another one, its modules are chosen

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
			t.textContent = c.type === "mod" ? `${named(c.from)} → ${named(c.to)} · ${c.key}` : `${named(c.from)} → ${named(c.to)}`;
			hit.append(t);
			const path = document.createElementNS(svgNS, "path");
			path.setAttribute("d", d);
			path.setAttribute("class", `pt-cable ${c.type}` + (c.id === picked ? " picked" : ""));
			path.dataset.id = c.id;
			for (const e of [hit, path]) e.addEventListener("pointerdown", ev => { ev.stopPropagation(); pick(c.id); });
			svg.append(hit, path);
		}
	}
	const named = id => { const n = st.nodes.find(x => x.id === id); return n ? n.kind + (n.preset ? " · " + n.preset : "") + " (" + id + ")" : id; };
	const modIndex = id => { const n = st.nodes.find(x => x.id === id); return n ? (n.ins || []).findIndex(x => x.type === "mod") : -1; };

	function makeNode(n) {
		const box = el("div", "pt-node");
		box.dataset.id = n.id;
		const head = el("div", "pt-head");
		const title = el("span", "pt-title");
		const helpBtn = el("button", "pt-help", "?"); helpBtn.title = "what this module is for: a tutorial and examples to load";
		helpBtn.addEventListener("pointerdown", e => e.stopPropagation()); // not a move of the module
		helpBtn.addEventListener("click", e => { e.stopPropagation(); openHelp(n.id); });
		head.append(title, helpBtn);
		head.addEventListener("contextmenu", e => { // a name of its own
			e.preventDefault();
			const was = (nodes[n.id] && nodes[n.id].data.name) || "";
			const name = window.prompt("A name for this module (empty: none)", was);
			if (name !== null) send({name: {id: n.id, value: name}});
		});
		const ports = el("div", "pt-ports");
		const ins = el("div", "pt-ins"), outs = el("div", "pt-outs");
		ports.append(ins, outs);
		const texts = el("div", "pt-texts"); // (its settings of text are in its side panel)
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
		box.addEventListener("dblclick", e => { if (onIt(e)) doubleTapped(n.id, e.altKey); });
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
			v.title.textContent = n.name ? n.name + " (" + n.kind + ")" : n.kind + (n.preset ? " · " + n.preset : "");
			const sig = JSON.stringify([n.ins, n.outs]);
			if (sig !== v.sig) {
				v.sig = sig;
				v.ins.innerHTML = ""; v.outs.innerHTML = "";
				(n.ins || []).forEach((port, i) => v.ins.append(portRow(n, "in", port, i)));
				(n.outs || []).forEach((port, i) => v.outs.append(portRow(n, "out", port, i)));
			}
			// an error is a toast, not a line on the module (which gets a red edge while it has one); no text is on a module, a status line is in its side panel
			const failed = /^error/i.test(n.status || "") ? n.status : "", errs = Object.entries(n.errors || {}).map(([k, e]) => `error: ${k}: ${e}`);
			toasts.watch(n, [failed, ...errs].filter(Boolean));
			v.el.classList.toggle("failed", !!failed || errs.length > 0);
			if (!dragging.has(n.id)) { v.el.style.left = n.x + "px"; v.el.style.top = n.y + "px"; }
			v.el.classList.toggle("selected", chosen.has(n.id));
		}
		for (const id of Object.keys(nodes)) if (!seen.has(id)) { nodes[id].el.remove(); delete nodes[id]; toasts.forget(id); }
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
			const top = Object.values(windows).sort((a, b) => (+b.el.style.zIndex) - (+a.el.style.zIndex))[0];
			if (top) closeWindow(top.id);
			else if (picked || chosen.size) { picked = ""; chosen.clear(); showChosen(); drawCables(); showInspector(); }
			else if (maxed) setMax(false);
			return;
		}
		// copy, cut, paste, duplicate the modules chosen (with the cables between them)
		if ((e.ctrlKey || e.metaKey) && !e.altKey && !typing() && ["c", "x", "v", "d"].includes(e.key.toLowerCase())) {
			const k = e.key.toLowerCase(), some = [...chosen].filter(id => id !== "out");
			if (k !== "v" && !some.length) return;
			e.preventDefault();
			if (k === "c" || k === "x" || k === "d") send({copy: some});
			if (k === "x") { for (const id of some) send({remove: id}); chosen.clear(); }
			if (k === "v" || k === "d") send({paste: true});
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

	// --- the window of a module: its settings of text (a pattern, values, a script), each in a box of lines, what is wrong or what it says below them, its settings (its
	// editor) below. A window inside the page (no browser window: it has no address bar), movable by its title, as big as it is made, one for each module, over the
	// patch. What is in it is module.html in a frame: this page tells it the module as it changes, it takes what is typed itself (see module.js) ---
	const windows = {}; // the windows open, by module: {el, frame, id, kind, key}
	let zTop = 60;
	const nodeOf = id => st.nodes.find(x => x.id === id);
	// a tap chooses a module; a double tap opens its window
	function tapped(id) {
		chosen.clear(); chosen.add(id); showChosen();
	}
	const storeKey = n => "psalm.patcher.window." + p.name + "." + n.kind;
	const kept = key => { try { return JSON.parse(localStorage.getItem(key) || "{}") || {}; } catch (e) { return {}; } };
	const num = (v, lo, hi) => Number.isFinite(v) ? Math.max(lo, Math.min(hi, Math.round(v))) : null;
	function keepPlace(w) {
		const r = w.el.getBoundingClientRect();
		try { localStorage.setItem(w.key, JSON.stringify({x: Math.round(r.left), y: Math.round(r.top), w: Math.round(r.width), h: Math.round(r.height)})); } catch (e) { /* no storage */ }
	}
	function toFront(w) { w.el.style.zIndex = ++zTop; }
	// a double tap opens the window of a module, or brings it to the front; a module that has a window of its own (a plugin) shows that, and its settings with Alt held
	function doubleTapped(id, alt) {
		const n = nodeOf(id);
		if (!n) return;
		if (n.native && !alt) { send({open: id}); return; }
		openWindow(id);
	}
	function openWindow(id) {
		const n = nodeOf(id);
		if (!n) return;
		const texts = n.texts || [];
		if (!texts.length && !n.page) return; // nothing to show
		if (windows[id]) { toFront(windows[id]); return; }
		const script = texts.some(t => t.lines > 1);
		const was = kept(storeKey(n));
		const vw = document.documentElement.clientWidth, vh = document.documentElement.clientHeight;
		const width = num(was.w, 300, vw) || Math.min(script ? 760 : 560, vw - 20);
		const height = num(was.h, 200, vh) || Math.min(script ? 780 : texts.length ? 680 : 600, vh - 20);
		const n_open = Object.keys(windows).length;
		const kx = num(was.x, 0, Math.max(0, vw - 120)), ky = num(was.y, 0, Math.max(0, vh - 60));
		const left = kx !== null && ky !== null ? kx : Math.max(10, vw - width - 20 - 30 * n_open);
		const top = kx !== null && ky !== null ? ky : Math.min(Math.max(10, vh - height - 10), 50 + 30 * n_open);
		const el_ = el("div", "pt-win");
		el_.style.cssText = `left:${left}px;top:${top}px;width:${width}px;height:${height}px`;
		const head = el("div", "pt-win-head");
		const title = el("span", "pt-win-title", n.kind + (n.preset ? " · " + n.preset : ""));
		const x = el("button", "pt-x", "×"); x.title = "close (Esc in a box): what is typed is taken";
		const hb = el("button", "pt-help-win", "?"); hb.title = "what this module is for: a tutorial and examples to load";
		hb.addEventListener("click", () => openHelp(id));
		head.append(title, hb, x);
		const frame = el("iframe", "pt-win-frame"); frame.title = n.kind;
		const grip = el("div", "pt-win-grip"); grip.title = "drag to resize";
		el_.append(head, frame, grip);
		root.append(el_);
		const w = {el: el_, frame, title, id, key: storeKey(n), sent: ""};
		windows[id] = w;
		toFront(w);
		const base = new URL(".", location.href).href;
		const q = new URLSearchParams({base, id, patcher: p.name, mount: p.data.mount, theme: api.theme(), mode: document.documentElement.dataset.mode || "", float: "1"});
		frame.src = base + "plugins/patcher/module.html?" + q;
		el_.addEventListener("pointerdown", () => toFront(w));
		x.addEventListener("click", () => closeWindow(id));
		// moved by its title, resized by its corner; the frames do not take the pointer meanwhile
		const drag = (target, onMove) => target.addEventListener("pointerdown", e => {
			if (e.target.closest("button")) return;
			e.preventDefault();
			target.setPointerCapture(e.pointerId);
			root.classList.add("pt-dragging");
			const r = el_.getBoundingClientRect(), x0 = e.clientX, y0 = e.clientY;
			const move = ev => onMove(r, ev.clientX - x0, ev.clientY - y0);
			const up = () => {
				target.removeEventListener("pointermove", move); target.removeEventListener("pointerup", up); target.removeEventListener("pointercancel", up);
				root.classList.remove("pt-dragging");
				keepPlace(w);
			};
			target.addEventListener("pointermove", move);
			target.addEventListener("pointerup", up);
			target.addEventListener("pointercancel", up);
		});
		const vwNow = () => document.documentElement.clientWidth, vhNow = () => document.documentElement.clientHeight;
		drag(head, (r, dx, dy) => {
			el_.style.left = Math.max(-r.width + 80, Math.min(vwNow() - 80, r.left + dx)) + "px";
			el_.style.top = Math.max(0, Math.min(vhNow() - 30, r.top + dy)) + "px";
		});
		drag(grip, (r, dx, dy) => {
			el_.style.width = Math.max(300, Math.min(vwNow() - r.left, r.width + dx)) + "px";
			el_.style.height = Math.max(200, Math.min(vhNow() - r.top, r.height + dy)) + "px";
		});
	}
	function closeWindow(id) {
		const w = windows[id];
		if (!w) return;
		delete windows[id];
		if (w.frame.contentWindow) w.frame.contentWindow.postMessage({psalm: "module-commit"}, "*"); // what is typed is taken
		keepPlace(w);
		setTimeout(() => w.el.remove(), 60); // the frame's request leaves first
	}
	// the windows of modules that are gone close; the others are told their module when it changes
	function updateWindows() {
		for (const [id, w] of Object.entries(windows)) {
			const n = nodeOf(id);
			if (!n) { closeWindow(id); continue; }
			const title = n.name ? n.name + " (" + n.kind + ")" : n.kind + (n.preset ? " · " + n.preset : "");
			if (w.title.textContent !== title) w.title.textContent = title;
			sendNode(w, n);
		}
	}
	function sendNode(w, n) {
		const s = JSON.stringify(n);
		if (s === w.sent || !w.frame.contentWindow || !w.ready) return;
		w.sent = s;
		w.frame.contentWindow.postMessage({psalm: "module", node: n}, "*");
	}
	window.addEventListener("message", e => {
		const m = e.data || {};
		for (const w of Object.values(windows)) {
			if (w.frame.contentWindow !== e.source) continue;
			if (m.psalm === "module-ready") { w.ready = true; w.sent = ""; const n = nodeOf(w.id); if (n) sendNode(w, n); }
			if (m.psalm === "module-close") closeWindow(w.id);
		}
	});

	// --- the help of a module: what it is for, a tutorial and examples (help/<kind>.md, markdown), in a window of its own, one a kind. An example (a block ```example) has a
	// button that loads it into the module the help was opened from: its texts ([pattern], [lanes], [values]... a line of its own names the text) and its settings ([set], a
	// `name = number` a line) ---
	const helps = {}; // the help windows open, by kind
	const inline = (parent, text, kind) => { // `code`, **bold**, [a link](kind.md) to the help of another module
		const re = /`([^`]+)`|\*\*([^*]+)\*\*|\[([^\]]+)\]\(([^)]+)\)/g;
		let at = 0, m;
		while ((m = re.exec(text))) {
			if (m.index > at) parent.append(text.slice(at, m.index));
			if (m[1] !== undefined) parent.append(el("code", "", m[1]));
			else if (m[2] !== undefined) parent.append(el("b", "", m[2]));
			else {
				const to = m[4].match(/^(?:\.\/)?([A-Za-z0-9%+ -]+)\.md$/); // (a kind with a space in its name is %20 in a link)
				const target = to && decodeURIComponent(to[1]);
				if (target && kindOf(target)) {
					const a = el("a", "", m[3]); a.href = "#"; a.title = "the help of " + target;
					a.addEventListener("click", e => { e.preventDefault(); openHelp(null, target); });
					parent.append(a);
				} else parent.append(m[3]);
			}
			at = re.lastIndex;
		}
		if (at < text.length) parent.append(text.slice(at));
		return parent;
	};
	function parseExample(body) { // [name] lines make the texts, [set] the settings
		const texts = {}, settings = {};
		let key = null;
		for (const line of body.split("\n")) {
			const h = line.match(/^\[([a-z]+)\]\s*$/);
			if (h) { key = h[1]; if (key !== "set") texts[key] = []; continue; }
			if (key === "set") {
				const kv = line.match(/^\s*([A-Za-z0-9_ -]+?)\s*=\s*(-?[0-9.]+)/);
				if (kv) settings[kv[1]] = Number(kv[2]);
			} else if (key) texts[key].push(line);
		}
		for (const k of Object.keys(texts)) texts[k] = texts[k].join("\n").replace(/\n+$/, "");
		return {texts, settings};
	}
	function markdown(text, id, kind) {
		const out = el("div", "pt-md");
		const lines = text.split("\n");
		for (let i = 0; i < lines.length;) {
			const line = lines[i];
			let m;
			if (!line.trim()) { i++; continue; }
			if ((m = line.match(/^```(\w*)\s*(.*)$/))) { // a block of code; an example has a button
				const body = [];
				for (i++; i < lines.length && !lines[i].startsWith("```"); i++) body.push(lines[i]);
				i++;
				const box = el("div", m[1] === "example" ? "pt-md-example" : "pt-md-code");
				if (m[1] === "example") {
					const ex = parseExample(body.join("\n"));
					const head = el("div", "pt-md-exhead");
					head.append(el("b", "", m[2] || "Example"));
					const load = el("button", "", "Load into the module"); load.title = "puts its texts and settings in the module (what was typed there is replaced)";
					const say = el("span", "pt-md-say");
					load.disabled = !id || !nodeOf(id);
					load.addEventListener("click", () => {
						send({example: {id, texts: ex.texts, settings: ex.settings}});
						say.textContent = "loaded"; setTimeout(() => { say.textContent = ""; }, 1500);
					});
					head.append(load, say);
					box.append(head);
					for (const [k, v] of Object.entries(ex.texts)) {
						if (Object.keys(ex.texts).length > 1) box.append(el("div", "pt-md-key", k));
						box.append(el("pre", "", v));
					}
					if (Object.keys(ex.settings).length) box.append(el("div", "pt-md-set", Object.entries(ex.settings).map(([k, v]) => k + " " + v).join(" · ")));
				} else box.append(el("pre", "", body.join("\n")));
				out.append(box);
			} else if ((m = line.match(/^(#{1,4})\s+(.*)$/))) {
				out.append(inline(el("h" + Math.min(4, m[1].length + 1)), m[2], kind)); i++;
			} else if (line.startsWith("|")) { // a table
				const rows = [];
				for (; i < lines.length && lines[i].startsWith("|"); i++) rows.push(lines[i]);
				const cells = r => r.replace(/^\||\|$/g, "").split(/(?<!\\)\|/).map(c => c.trim().replace(/\\\|/g, "|"));
				const table = el("table");
				rows.forEach((r, ri) => {
					if (/^\|[\s:|-]+\|?$/.test(r)) return;
					const tr = el("tr");
					for (const c of cells(r)) tr.append(inline(el(ri === 0 ? "th" : "td"), c, kind));
					table.append(tr);
				});
				out.append(table);
			} else if (/^\s*([-*]|\d+\.)\s+/.test(line)) { // a list
				const ordered = /^\s*\d+\./.test(line);
				const list = el(ordered ? "ol" : "ul");
				for (; i < lines.length && /^\s*([-*]|\d+\.)\s+/.test(lines[i]); i++) list.append(inline(el("li"), lines[i].replace(/^\s*([-*]|\d+\.)\s+/, ""), kind));
				out.append(list);
			} else { // a paragraph
				const para = [];
				for (; i < lines.length && lines[i].trim() && !/^(```|#|\||\s*([-*]|\d+\.)\s)/.test(lines[i]); i++) para.push(lines[i]);
				out.append(inline(el("p"), para.join(" "), kind));
			}
		}
		return out;
	}
	// openHelp opens the help of the kind of a module (id), or of a kind (id null: from a link in a help), over the module it was opened from
	function openHelp(id, kindName) {
		const n = id ? nodeOf(id) : null;
		const kind = kindName || (n && n.kind);
		if (!kind) return;
		const was = helps[kind];
		if (was) { if (id) was.id = id; was.render(); toFront(was); return; }
		const el_ = el("div", "pt-win pt-help-window");
		const vw = document.documentElement.clientWidth, vh = document.documentElement.clientHeight;
		const width = Math.min(640, vw - 20), height = Math.min(720, vh - 20), count = Object.keys(helps).length;
		el_.style.cssText = `left:${Math.max(10, 40 + 30 * count)}px;top:${Math.max(10, 40 + 30 * count)}px;width:${width}px;height:${height}px`;
		const head = el("div", "pt-win-head");
		const x = el("button", "pt-x", "×"); x.title = "close";
		head.append(el("span", "pt-win-title", "help · " + kind), x);
		const page = el("div", "pt-help-page");
		const grip = el("div", "pt-win-grip"); grip.title = "drag to resize";
		el_.append(head, page, grip);
		root.append(el_);
		const w = {el: el_, id, kind, render() {
			page.innerHTML = "";
			page.append(el("p", "pt-md-wait", "…"));
			fetch(new URL(".", location.href).href + "plugins/patcher/help/" + encodeURIComponent(kind) + ".md")
				.then(r => r.ok ? r.text() : Promise.reject(new Error("no help yet for " + kind)))
				.then(t => { page.innerHTML = ""; page.append(markdown(t, w.id, kind)); })
				.catch(err => { page.innerHTML = ""; page.append(el("p", "pt-md-wait", String(err.message || err))); });
		}};
		helps[kind] = w;
		toFront(w);
		w.render();
		el_.addEventListener("pointerdown", () => toFront(w));
		x.addEventListener("click", () => { delete helps[kind]; el_.remove(); });
		const drag = (target, onMove) => target.addEventListener("pointerdown", e => {
			if (e.target.closest("button")) return;
			e.preventDefault();
			target.setPointerCapture(e.pointerId);
			const r = el_.getBoundingClientRect(), x0 = e.clientX, y0 = e.clientY;
			const move = ev => onMove(r, ev.clientX - x0, ev.clientY - y0);
			const up = () => { target.removeEventListener("pointermove", move); target.removeEventListener("pointerup", up); target.removeEventListener("pointercancel", up); };
			target.addEventListener("pointermove", move);
			target.addEventListener("pointerup", up);
			target.addEventListener("pointercancel", up);
		});
		drag(head, (r, dx, dy) => {
			el_.style.left = Math.max(-r.width + 80, Math.min(document.documentElement.clientWidth - 80, r.left + dx)) + "px";
			el_.style.top = Math.max(0, Math.min(document.documentElement.clientHeight - 30, r.top + dy)) + "px";
		});
		drag(grip, (r, dx, dy) => {
			el_.style.width = Math.max(300, Math.min(document.documentElement.clientWidth - r.left, r.width + dx)) + "px";
			el_.style.height = Math.max(200, Math.min(document.documentElement.clientHeight - r.top, r.height + dy)) + "px";
		});
	}
	// the Load button of an example needs its module: it goes when the module does
	function updateHelps() {
		for (const w of Object.values(helps)) if (w.id && !nodeOf(w.id)) { w.id = null; w.render(); }
	}

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
			showJournal(s.journal || "", s.lines); // the log changes on its own, the patch is only drawn again when it changes
			const {journal, lines, heads, ...rest} = s;
			lanes.heads(heads); // where the lanes are changes all the time, like the log
			const was = JSON.stringify(st);
			st = rest;
			lanes.update(st.nodes, st.selected);
			grid.update(st);
			if (JSON.stringify(rest) === was) return;
			updateNodes();
			if (picked && !st.cables.some(c => c.id === picked)) picked = "";
			if ((st.pasteN || 0) !== pastes) { // a paste: its modules chosen, to move them where they go
				if (pastes >= 0) { chosen.clear(); for (const id of st.pasted || []) chosen.add(id); showChosen(); }
				pastes = st.pasteN || 0;
			}
			if (pickNew) {
				const c = st.cables.find(c => c.type === "mod" && !pickNew.has(c.id));
				if (c) { picked = c.id; pickNew = null; }
			}
			drawCables();
			if (!inspector.contains(document.activeElement)) showInspector();
			updatePatches();
			updateWindows();
			updateHelps();
		},
		restyle() { drawCables(); lanes.restyle(); },
	};
});

// where the song of a patch is, in the bar at the top: the section playing, its bar, the one asked
// for next and how far it is; nothing without a song
psalm.register("songbar", (p, api) => {
	"use strict";
	const box = document.createElement("div");
	box.className = "pt-songbar"; box.hidden = true;
	box.title = "the song: the section playing, its bar, the section asked for next (a song module's tiles)";
	const text = document.createElement("span"), fill = document.createElement("div");
	const bar = document.createElement("div"); bar.className = "pt-songbar-bar"; bar.append(fill);
	box.append(text, bar);
	return {
		el: box,
		show() {},
		update(s) {
			box.hidden = !s || !(s.section || s.ended);
			if (box.hidden) return;
			text.textContent = s.ended ? "the song is over" : `${s.section} ${s.bar + 1}/${s.bars}` + (s.queued ? ` → ${s.queued}` : "");
			fill.style.width = (s.ended ? 0 : Math.max(0, Math.min(1, s.progress)) * 100) + "%";
		},
	};
});
