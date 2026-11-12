// the window of a module of the patcher (a double click on the module opens it, inside the page, in a frame): its texts, each in a box of lines (what is wrong or what it
// says below them), its settings (its editor, served by the patcher's mount for this module: mount/<name>@<id>/) below. The page that holds the frame tells it its
// module as it changes (a message) and moves and closes the window; what is typed goes to the server from here. Query: base (the folder of the page), id, patcher
// (the name of the patcher on the page), mount, theme, mode
"use strict";
(() => {
	const q = new URLSearchParams(location.search);
	const base = q.get("base") || "../../", id = q.get("id") || "", patcher = q.get("patcher") || "", mount = q.get("mount") || "";
	const theme = q.get("theme") || "", mode = q.get("mode") || "";
	const $ = name => document.getElementById(name);
	const el = (tag, cls, text) => { const e = document.createElement(tag); if (cls) e.className = cls; if (text !== undefined) e.textContent = text; return e; };
	if (mode) document.documentElement.dataset.mode = mode;
	$("theme").href = base + "theme.css" + (theme ? "?name=" + encodeURIComponent(theme) : "");

	const root = document.querySelector(".mw");
	let node = null, boxes = {}, log = null, built = "", framed = false, texts = [];
	const send = data => fetch(base + "api/message", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify({name: patcher, data}), keepalive: true}).catch(() => {});
	const commitText = (key, value) => send({text: {id, key, value}});
	// what is typed taken: every text (only those changed, when closing)
	function commit(changed) {
		for (const [key, ta] of Object.entries(boxes)) {
			const t = texts.find(x => x.key === key);
			if (!changed || !t || ta.value !== t.value) commitText(key, ta.value);
		}
	}
	function help(text) { // what to know to type it: folded away when it is long, so it does not push the box down
		const lines = text.split("\n");
		const list = helpList(text);
		if (lines.length < 3) return list;
		const d = el("details", "mw-helpbox");
		d.append(el("summary", "", "How to write it"), list);
		return d;
	}
	function helpList(text) {
		const list = el("ul", "mw-help");
		for (const line of text.split("\n")) {
			const li = el("li");
			line.split("`").forEach((part, i) => { if (part) li.append(i % 2 ? el("code", "", part) : document.createTextNode(part)); });
			list.append(li);
		}
		return list;
	}
	// the lanes of a clock or a part as bars to drag, under their text: a row a lane, a bar a step; the length of a lane is its own (+ and -), a click on the label takes a gap (the
	// default) in or out. What is dragged is written into the text, and sent
	const LANES = {prob: [0, 100, 100, "chance %"], vel: [0, 150, 100, "velocity %"], shift: [-50, 50, 0, "timing %"], len: [10, 300, 100, "length %"], pitch: [-12, 12, 0, "pitch st"]};
	function lanesEditor(ta, key) {
		const root = el("div", "mw-lanes");
		const read = () => {
			const out = {};
			for (const line of ta.value.split("\n")) {
				const [name, vals] = line.split(":");
				const n = (name || "").trim().toLowerCase();
				const canon = {chance: "prob", velocity: "vel", timing: "shift", length: "len"}[n] || n;
				if (!LANES[canon] || vals === undefined) continue;
				out[canon] = vals.trim().split(/\s+/).filter(Boolean).map(v => v === "." ? null : Number(v)).map(v => (v !== null && isNaN(v)) ? null : v);
			}
			return out;
		};
		let model = read();
		const write = () => {
			ta.value = Object.keys(LANES).filter(n => model[n] && model[n].length).map(n => n + ": " + model[n].map(v => v === null ? "." : String(Math.round(v * 100) / 100)).join(" ")).join("\n");
			commitText(key, ta.value);
			draw();
		};
		function draw() {
			root.innerHTML = "";
			for (const [name, [lo, hi, def, label]] of Object.entries(LANES)) {
				const row = el("div", "mw-lane"); row.dataset.lane = name;
				const vals = model[name];
				const head = el("span", "mw-lane-label", label); head.title = label + ": the value of each step, going round on its own length";
				const less = el("button", "mw-lane-btn", "\u2212"), more = el("button", "mw-lane-btn", "+");
				less.title = "a step less"; more.title = "a step more (or start the lane)";
				less.addEventListener("click", () => { if (vals && vals.length > 1) { vals.pop(); write(); } else if (vals) { delete model[name]; write(); } });
				more.addEventListener("click", () => { if (!model[name]) model[name] = [def]; else if (model[name].length < 64) model[name].push(def); write(); });
				const bars = el("div", "mw-lane-bars");
				(vals || []).forEach((v, i) => {
					const b = el("div", "mw-lane-bar"); b.dataset.step = String(i);
					const f = el("div", "mw-lane-fill");
					if (v === null) b.classList.add("gap");
					else {
						const zero = lo < 0 ? 0.5 : 0, at = (v - lo) / (hi - lo);
						f.style.bottom = (Math.min(at, zero) * 100) + "%"; f.style.height = (Math.abs(at - zero) * 100) + "%";
						b.title = String(v);
					}
					b.append(f);
					const set = e => {
						const r = b.getBoundingClientRect();
						let v2 = lo + (1 - Math.max(0, Math.min(1, (e.clientY - r.top) / r.height))) * (hi - lo);
						v2 = (hi - lo) > 40 ? Math.round(v2 / 5) * 5 : Math.round(v2 * 2) / 2;
						if (hi - lo <= 24) v2 = Math.round(v2);
						model[name][i] = Math.max(lo, Math.min(hi, v2));
					};
					b.addEventListener("pointerdown", e => {
						if (e.button === 2 || e.altKey) { e.preventDefault(); model[name][i] = model[name][i] === null ? def : null; write(); return; }
						b.setPointerCapture(e.pointerId);
						set(e); b.dataset.drag = "1";
						const move = ev => { if (b.dataset.drag) { const under = bars.children[Math.max(0, Math.min(bars.children.length - 1, Math.floor((ev.clientX - bars.getBoundingClientRect().left) / b.getBoundingClientRect().width)))]; const j = Number(under.dataset.step); const r = under.getBoundingClientRect(); let v3 = lo + (1 - Math.max(0, Math.min(1, (ev.clientY - r.top) / r.height))) * (hi - lo); v3 = (hi - lo) > 40 ? Math.round(v3 / 5) * 5 : (hi - lo <= 24 ? Math.round(v3) : Math.round(v3 * 2) / 2); model[name][j] = Math.max(lo, Math.min(hi, v3)); } };
						const up = () => { delete b.dataset.drag; b.removeEventListener("pointermove", move); b.removeEventListener("pointerup", up); write(); };
						b.addEventListener("pointermove", move); b.addEventListener("pointerup", up);
					});
					b.addEventListener("contextmenu", e => e.preventDefault());
					bars.append(b);
				});
				row.append(head, less, more, bars);
				root.append(row);
			}
		}
		ta.addEventListener("input", () => { model = read(); draw(); });
		draw();
		return {root, refresh: () => { model = read(); draw(); }};
	}

	// the boxes are made when the module has other texts than they were made for
	function build(n) {
		const key = JSON.stringify([n.kind, n.preset, (n.texts || []).map(t => [t.key, t.label, t.lines]), (n.actions || []).map(a => [a.key, a.label])]) + (n.log !== undefined && n.log !== null) + !!n.page;
		if (key === built) return;
		built = key;
		boxes = {}; log = null;
		const holder = $("boxes");
		holder.innerHTML = "";
		texts = n.texts || [];
		for (const t of texts) {
			const wrap = el("label", "mw-box");
			wrap.hidden = !!t.hidden; // a view of its own edits it (the shaper's rack)
			if (texts.length > 1) wrap.append(el("span", "mw-label", t.label));
			if (t.help) wrap.append(help(t.help));
			const ta = el("textarea", "mw-text"); ta.spellcheck = false; ta.wrap = "off"; ta.value = t.value;
			ta.addEventListener("keydown", e => {
				e.stopPropagation();
				if (e.key === "Tab") { e.preventDefault(); document.execCommand("insertText", false, "\t"); }
				else if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) { e.preventDefault(); commit(); }
				else if (e.key === "Escape") { e.preventDefault(); commit(true); window.parent.postMessage({psalm: "module-close"}, "*"); }
			});
			wrap.append(ta);
			if (t.key === "lanes") { const ed = lanesEditor(ta, t.key); wrap.append(ed.root); wrap.classList.add("mw-has-lanes"); }
			if (!(t.lines > 1)) wrap.classList.add("mw-natural"); // a box of its own height; the box of a script takes what is left
			holder.append(wrap);
			boxes[t.key] = ta;
		}
		if ((n.actions || []).length) { // the buttons of the module
			const row = el("div", "mw-actions");
			for (const a of n.actions) {
				const b = el("button", "", a.label); b.title = a.title || "";
				b.addEventListener("click", () => { commit(true); send({action: {id, key: a.key}}); });
				row.append(b);
			}
			holder.append(row);
		}
		if (n.log !== undefined && n.log !== null) {
			for (const w of holder.children) w.style.flex = "0 0 auto";
			const wrap = el("label", "mw-box");
			const ta = el("textarea", "mw-text"); ta.readOnly = true; ta.spellcheck = false; ta.wrap = "off";
			wrap.append(ta); holder.append(wrap);
			log = ta;
		}
		const shown = texts.filter(t => !t.hidden);
		$("code").hidden = $("run").hidden = !shown.length && !(n.actions || []).length;
		$("settings").hidden = !n.page;
		const script = shown.some(t => t.lines > 1);
		root.classList.toggle("both", !!shown.length && !!n.page && script);
		root.classList.toggle("short", !!shown.length && !!n.page && !script);
		document.title = $("title").textContent = n.kind + (n.preset ? " · " + n.preset : "");
		const first = holder.querySelector(".mw-box:not([hidden]) textarea");
		if (first) first.focus();
	}
	function show(n) {
		$("gone").hidden = !!n;
		if (!n) { root.querySelector(".mw-head").hidden = $("code").hidden = $("settings").hidden = true; return; }
		root.querySelector(".mw-head").hidden = false;
		if (!node || node.id !== n.id) framed = false;
		node = n;
		build(n);
		if (n.page && !framed) {
			framed = true;
			$("frame").src = base + "mount/" + encodeURIComponent(mount) + "@" + encodeURIComponent(id) + "/?embed&id=" + encodeURIComponent(id) + "&theme=" + encodeURIComponent(theme);
		}
		if (log && typeof n.log === "string" && log.value !== n.log) { // the newest lines stay in view
			const atEnd = log.scrollTop + log.clientHeight >= log.scrollHeight - 4;
			log.value = n.log;
			if (atEnd) log.scrollTop = log.scrollHeight;
		}
		let say = "";
		for (const [key, ta] of Object.entries(boxes)) { // what is wrong with a text: below the boxes
			const e = n.errors && n.errors[key];
			ta.classList.toggle("bad", !!e);
			if (e && !say) say = "error: " + (Object.keys(boxes).length > 1 ? key + ": " : "") + e;
		}
		if (!say) say = n.status || "";
		$("status").textContent = say;
		$("status").classList.toggle("bad", /^error/i.test(say));
		document.body.dataset.ready = "1"; // it has its module (tests wait for it)
	}

	// the settings frame tells its height; it gets the theme and the mode like the frames of the page
	window.addEventListener("message", e => {
		const m = e.data || {};
		if (e.source === $("frame").contentWindow && m.psalm === "height") $("frame").style.height = m.height + "px";
		if (e.source === window.parent) {
			if (m.psalm === "module") show(m.node);
			if (m.psalm === "module-commit") commit(true);
		}
	});
	$("frame").addEventListener("load", () => {
		const w = $("frame").contentWindow;
		if (w && theme) w.postMessage({psalm: "theme", name: theme}, "*");
		if (w && mode) w.postMessage({psalm: "mode", mode}, "*");
	});
	$("run").addEventListener("click", () => commit());
	window.addEventListener("pagehide", () => commit(true));

	// the page tells the module as it changes: it is asked for it now
	window.parent.postMessage({psalm: "module-ready"}, "*");
})();
