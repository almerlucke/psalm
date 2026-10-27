#include "glue.h"

#include <X11/Xatom.h>
#include <X11/Xlib.h>
#include <dlfcn.h>
#include <poll.h>
#include <pthread.h>
#include <setjmp.h>
#include <signal.h>
#include <time.h>
#include <unistd.h>

#include <algorithm>
#include <atomic>
#include <cstdio>
#include <cstring>
#include <map>
#include <mutex>
#include <string>
#include <vector>

#include "pluginterfaces/base/funknown.h"
#include "pluginterfaces/base/ipluginbase.h"
#include "pluginterfaces/base/ibstream.h"
#include "pluginterfaces/base/ipluginbase.h"
#include "pluginterfaces/gui/iplugview.h"
#include "pluginterfaces/vst/ivstattributes.h"
#include "pluginterfaces/vst/ivstaudioprocessor.h"
#include "pluginterfaces/vst/ivstcomponent.h"
#include "pluginterfaces/vst/ivsteditcontroller.h"
#include "pluginterfaces/vst/ivstevents.h"
#include "pluginterfaces/vst/ivsthostapplication.h"
#include "pluginterfaces/vst/ivstmessage.h"
#include "pluginterfaces/vst/ivstnoteexpression.h"
#include "pluginterfaces/vst/ivstparameterchanges.h"
#include "pluginterfaces/vst/ivstpluginterfacesupport.h"
#include "pluginterfaces/vst/ivstprocesscontext.h"
#include "pluginterfaces/vst/ivstunits.h"
#include "pluginterfaces/vst/vstspeaker.h"

using namespace Steinberg;
using namespace Steinberg::Vst;

// --- the plugin's own thread rules: a crash in a call we made into it is caught (see GUARD below) ---

static __thread sigjmp_buf *tl_jmp;
static struct sigaction old_acts[32];
static const int guarded_signals[] = {SIGSEGV, SIGBUS, SIGFPE, SIGILL, SIGABRT};
static pthread_once_t guard_once = PTHREAD_ONCE_INIT;

static void crash_handler(int sig, siginfo_t *si, void *uc) {
	if (tl_jmp) {
		siglongjmp(*tl_jmp, sig);
	}
	struct sigaction *old = &old_acts[sig];
	if (old->sa_flags & SA_SIGINFO) {
		old->sa_sigaction(sig, si, uc);
	} else if (old->sa_handler == SIG_DFL) {
		signal(sig, SIG_DFL);
		raise(sig);
	} else if (old->sa_handler != SIG_IGN) {
		old->sa_handler(sig);
	}
}

static void install_guard(void) {
	for (size_t i = 0; i < sizeof guarded_signals / sizeof *guarded_signals; i++) {
		struct sigaction sa;
		memset(&sa, 0, sizeof sa);
		sa.sa_sigaction = crash_handler;
		sa.sa_flags = SA_SIGINFO | SA_ONSTACK | SA_NODEFER;
		sigemptyset(&sa.sa_mask);
		sigaction(guarded_signals[i], &sa, &old_acts[guarded_signals[i]]);
	}
}

struct v3_inst;
static void mark_dead(v3_inst *inst, int sig);
static bool inst_dead(v3_inst *inst);

// a call into a plugin: a crash in it jumps back here, the instance is marked dead and the call fails
#define GUARD(inst, call, failure) \
	do { \
		pthread_once(&guard_once, install_guard); \
		v3_inst *gi_ = (inst); \
		if (gi_ && inst_dead(gi_)) { \
			return failure; \
		} \
		sigjmp_buf jb_; \
		sigjmp_buf *prev_ = tl_jmp; \
		int sig_ = sigsetjmp(jb_, 1); \
		if (sig_ == 0) { \
			tl_jmp = &jb_; \
			auto r_ = call; \
			tl_jmp = prev_; \
			return r_; \
		} \
		tl_jmp = prev_; \
		mark_dead(gi_, sig_); \
		return failure; \
	} while (0)

#define GUARD_VOID(inst, call) \
	do { \
		pthread_once(&guard_once, install_guard); \
		v3_inst *gi_ = (inst); \
		if (gi_ && inst_dead(gi_)) { \
			return; \
		} \
		sigjmp_buf jb_; \
		sigjmp_buf *prev_ = tl_jmp; \
		int sig_ = sigsetjmp(jb_, 1); \
		if (sig_ == 0) { \
			tl_jmp = &jb_; \
			call; \
			tl_jmp = prev_; \
			return; \
		} \
		tl_jmp = prev_; \
		mark_dead(gi_, sig_); \
	} while (0)

// a call in the middle of a function: a crash jumps to the end of it
#define SAFE(inst, stmt) \
	do { \
		pthread_once(&guard_once, install_guard); \
		v3_inst *si_ = (inst); \
		if (!inst_dead(si_)) { \
			sigjmp_buf jb_; \
			sigjmp_buf *prev_ = tl_jmp; \
			int sig_ = sigsetjmp(jb_, 1); \
			if (sig_ == 0) { \
				tl_jmp = &jb_; \
				stmt; \
				tl_jmp = prev_; \
			} else { \
				tl_jmp = prev_; \
				mark_dead(si_, sig_); \
			} \
		} \
	} while (0)

// --- strings ---

static void utf16_to_utf8(const char16 *s, char *out, int cap) {
	int n = 0;
	for (; *s && n < cap - 4; s++) {
		uint32_t c = *s;
		if (c >= 0xD800 && c < 0xDC00 && s[1] >= 0xDC00 && s[1] < 0xE000) {
			c = 0x10000 + ((c - 0xD800) << 10) + (s[1] - 0xDC00);
			s++;
		}
		if (c < 0x80) {
			out[n++] = (char)c;
		} else if (c < 0x800) {
			out[n++] = (char)(0xC0 | (c >> 6));
			out[n++] = (char)(0x80 | (c & 0x3F));
		} else if (c < 0x10000) {
			out[n++] = (char)(0xE0 | (c >> 12));
			out[n++] = (char)(0x80 | ((c >> 6) & 0x3F));
			out[n++] = (char)(0x80 | (c & 0x3F));
		} else {
			out[n++] = (char)(0xF0 | (c >> 18));
			out[n++] = (char)(0x80 | ((c >> 12) & 0x3F));
			out[n++] = (char)(0x80 | ((c >> 6) & 0x3F));
			out[n++] = (char)(0x80 | (c & 0x3F));
		}
	}
	out[n] = 0;
}

static void set_err(char *err, int cap, const char *msg) {
	if (err && cap > 0) {
		snprintf(err, cap, "%s", msg);
	}
}

static bool tuid_equal(const TUID a, const TUID b) { return FUnknownPrivate::iidEqual(a, b); }

static std::string cid_to_hex(const TUID cid) {
	static const char *hex = "0123456789ABCDEF";
	std::string s;
	for (int i = 0; i < 16; i++) {
		s += hex[(uint8_t)cid[i] >> 4];
		s += hex[(uint8_t)cid[i] & 15];
	}
	return s;
}

static bool hex_to_cid(const char *s, TUID cid) {
	if (strlen(s) != 32) {
		return false;
	}
	for (int i = 0; i < 16; i++) {
		unsigned v;
		if (sscanf(s + 2 * i, "%2x", &v) != 1) {
			return false;
		}
		cid[i] = (char)v;
	}
	return true;
}

// --- a counted object with the interfaces it has ---

template <class T, class I>
static bool give(T *self, const TUID iid, void **obj) {
	if (tuid_equal(iid, I::iid.toTUID())) {
		*obj = static_cast<I *>(self);
		self->addRef();
		return true;
	}
	return false;
}

template <class First, class... Rest>
class Obj : public First, public Rest... {
public:
	virtual ~Obj() {}
	tresult PLUGIN_API queryInterface(const TUID iid, void **obj) override {
		if (tuid_equal(iid, FUnknown::iid.toTUID())) {
			*obj = static_cast<FUnknown *>(static_cast<First *>(this));
			addRef();
			return kResultOk;
		}
		if (give<Obj, First>(this, iid, obj) || (give<Obj, Rest>(this, iid, obj) || ...)) {
			return kResultOk;
		}
		*obj = nullptr;
		return kNoInterface;
	}
	uint32 PLUGIN_API addRef() override { return ++refs_; }
	uint32 PLUGIN_API release() override {
		uint32 n = --refs_;
		if (n == 0) {
			delete this;
		}
		return n;
	}

private:
	std::atomic<uint32> refs_{1};
};

// --- what the host gives a plugin ---

class HostAttrs : public Obj<IAttributeList> {
public:
	struct Value {
		int kind = 0; // 1 int, 2 float, 3 string, 4 binary
		int64 i = 0;
		double f = 0;
		std::vector<char16> s;
		std::vector<char> b;
	};
	std::map<std::string, Value> m;
	tresult PLUGIN_API setInt(AttrID id, int64 v) override { auto &x = m[id]; x.kind = 1; x.i = v; return kResultOk; }
	tresult PLUGIN_API getInt(AttrID id, int64 &v) override {
		auto it = m.find(id);
		if (it == m.end() || it->second.kind != 1) return kResultFalse;
		v = it->second.i;
		return kResultOk;
	}
	tresult PLUGIN_API setFloat(AttrID id, double v) override { auto &x = m[id]; x.kind = 2; x.f = v; return kResultOk; }
	tresult PLUGIN_API getFloat(AttrID id, double &v) override {
		auto it = m.find(id);
		if (it == m.end() || it->second.kind != 2) return kResultFalse;
		v = it->second.f;
		return kResultOk;
	}
	tresult PLUGIN_API setString(AttrID id, const TChar *s) override {
		auto &x = m[id];
		x.kind = 3;
		x.s.assign(s, s + 1);
		for (const TChar *p = s; *p; p++) x.s.insert(x.s.end() - 1, *p);
		return kResultOk;
	}
	tresult PLUGIN_API getString(AttrID id, TChar *s, uint32 bytes) override {
		auto it = m.find(id);
		if (it == m.end() || it->second.kind != 3) return kResultFalse;
		uint32 n = (uint32)it->second.s.size();
		if (n * sizeof(TChar) > bytes) n = bytes / sizeof(TChar);
		if (n == 0) return kResultFalse;
		memcpy(s, it->second.s.data(), n * sizeof(TChar));
		s[n - 1] = 0;
		return kResultOk;
	}
	tresult PLUGIN_API setBinary(AttrID id, const void *data, uint32 size) override {
		auto &x = m[id];
		x.kind = 4;
		x.b.assign((const char *)data, (const char *)data + size);
		return kResultOk;
	}
	tresult PLUGIN_API getBinary(AttrID id, const void *&data, uint32 &size) override {
		auto it = m.find(id);
		if (it == m.end() || it->second.kind != 4) return kResultFalse;
		data = it->second.b.data();
		size = (uint32)it->second.b.size();
		return kResultOk;
	}
};

class HostMessage : public Obj<IMessage> {
public:
	std::string id;
	HostAttrs *attrs = new HostAttrs;
	~HostMessage() override { attrs->release(); }
	FIDString PLUGIN_API getMessageID() override { return id.c_str(); }
	void PLUGIN_API setMessageID(FIDString i) override { id = i ? i : ""; }
	IAttributeList *PLUGIN_API getAttributes() override { return attrs; }
};

class HostApp : public Obj<IHostApplication, IPlugInterfaceSupport> {
public:
	tresult PLUGIN_API getName(String128 name) override {
		const char *n = "psalm";
		int i = 0;
		for (; n[i]; i++) name[i] = (char16)n[i];
		name[i] = 0;
		return kResultOk;
	}
	tresult PLUGIN_API createInstance(TUID cid, TUID iid, void **obj) override {
		if (tuid_equal(cid, IMessage::iid.toTUID()) && tuid_equal(iid, IMessage::iid.toTUID())) {
			*obj = static_cast<IMessage *>(new HostMessage);
			return kResultOk;
		}
		if (tuid_equal(cid, IAttributeList::iid.toTUID()) && tuid_equal(iid, IAttributeList::iid.toTUID())) {
			*obj = static_cast<IAttributeList *>(new HostAttrs);
			return kResultOk;
		}
		*obj = nullptr;
		return kResultFalse;
	}
	tresult PLUGIN_API isPlugInterfaceSupported(const TUID iid) override {
		static const FUID *known[] = {&IComponent::iid, &IAudioProcessor::iid, &IEditController::iid, &IConnectionPoint::iid, &IUnitInfo::iid, &IPlugView::iid,
		                              &IComponentHandler::iid, &IComponentHandler2::iid, &IUnitHandler::iid, &IPlugFrame::iid, &Linux::IRunLoop::iid};
		for (auto k : known) {
			if (tuid_equal(iid, k->toTUID())) return kResultTrue;
		}
		return kResultFalse;
	}
};

// a block of memory as a stream, to read or to write
class MemStream : public Obj<IBStream, ISizeableStream> {
public:
	std::vector<uint8_t> data;
	int64 pos = 0;
	MemStream() {}
	MemStream(const uint8_t *p, size_t n) : data(p, p + n) {}
	tresult PLUGIN_API read(void *buf, int32 n, int32 *got) override {
		int64 left = (int64)data.size() - pos;
		int32 k = (int32)std::max<int64>(0, std::min<int64>(n, left));
		if (k > 0) memcpy(buf, data.data() + pos, k);
		pos += k;
		if (got) *got = k;
		return kResultOk;
	}
	tresult PLUGIN_API write(void *buf, int32 n, int32 *wrote) override {
		if (pos + n > (int64)data.size()) data.resize(pos + n);
		memcpy(data.data() + pos, buf, n);
		pos += n;
		if (wrote) *wrote = n;
		return kResultOk;
	}
	tresult PLUGIN_API seek(int64 p, int32 mode, int64 *result) override {
		int64 base = mode == kIBSeekSet ? 0 : mode == kIBSeekCur ? pos : (int64)data.size();
		int64 np = base + p;
		if (np < 0) return kResultFalse;
		pos = np;
		if (result) *result = pos;
		return kResultOk;
	}
	tresult PLUGIN_API tell(int64 *p) override {
		if (p) *p = pos;
		return kResultOk;
	}
	tresult PLUGIN_API getStreamSize(int64 &size) override { size = (int64)data.size(); return kResultOk; }
	tresult PLUGIN_API setStreamSize(int64 size) override { data.resize((size_t)size); return kResultOk; }
};

class HostEvents : public Obj<IEventList> {
public:
	std::vector<Event> list;
	int32 PLUGIN_API getEventCount() override { return (int32)list.size(); }
	tresult PLUGIN_API getEvent(int32 i, Event &e) override {
		if (i < 0 || i >= (int32)list.size()) return kResultFalse;
		e = list[i];
		return kResultOk;
	}
	tresult PLUGIN_API addEvent(Event &e) override { list.push_back(e); return kResultOk; }
};

class HostQueue : public Obj<IParamValueQueue> {
public:
	ParamID id = 0;
	std::vector<std::pair<int32, ParamValue>> points;
	ParamID PLUGIN_API getParameterId() override { return id; }
	int32 PLUGIN_API getPointCount() override { return (int32)points.size(); }
	tresult PLUGIN_API getPoint(int32 i, int32 &offset, ParamValue &value) override {
		if (i < 0 || i >= (int32)points.size()) return kResultFalse;
		offset = points[i].first;
		value = points[i].second;
		return kResultOk;
	}
	tresult PLUGIN_API addPoint(int32 offset, ParamValue value, int32 &index) override {
		points.emplace_back(offset, value);
		index = (int32)points.size() - 1;
		return kResultOk;
	}
};

class HostChanges : public Obj<IParameterChanges> {
public:
	std::vector<HostQueue *> queues; // all made; the first used are the ones of the block
	int32 used = 0;
	~HostChanges() override { for (auto q : queues) q->release(); }
	void clear() {
		for (int32 i = 0; i < used; i++) queues[i]->points.clear();
		used = 0;
	}
	int32 PLUGIN_API getParameterCount() override { return used; }
	IParamValueQueue *PLUGIN_API getParameterData(int32 i) override { return i >= 0 && i < used ? queues[i] : nullptr; }
	IParamValueQueue *PLUGIN_API addParameterData(const ParamID &id, int32 &index) override {
		for (int32 i = 0; i < used; i++) {
			if (queues[i]->id == id) {
				index = i;
				return queues[i];
			}
		}
		if (used == (int32)queues.size()) queues.push_back(new HostQueue);
		HostQueue *q = queues[used];
		q->id = id;
		q->points.clear();
		index = used++;
		return q;
	}
};

// --- the library and the instance ---

struct v3_lib {
	void *handle = nullptr;
	IPluginFactory *factory = nullptr;
	bool (*exitFn)() = nullptr;
	std::string path;
};

class HostHandler;
class HostFrame;

struct v3_inst {
	v3_lib *lib = nullptr;
	HostApp *app = nullptr;
	IComponent *comp = nullptr;
	IAudioProcessor *proc = nullptr;
	IEditController *ctrl = nullptr;
	bool ctrlSeparate = false;
	IConnectionPoint *cpComp = nullptr, *cpCtrl = nullptr;
	HostHandler *handler = nullptr;
	double sr = 44100;
	uint32_t maxFrames = 256;
	bool active = false, processing = false;
	ProgramListID progList = kNoProgramListId;
	bool tuningExpression = false; // the plugin takes the tuning of a note as a note expression (else as the tuning of the note on)

	// the audio buses (the first of the inputs, and the second as a side chain; every output up to 16)
	int nIn = 0, nOut = 0;
	std::vector<int> inCh, outCh;
	std::vector<std::string> inName, outName;
	std::vector<float> inbuf, outbuf, silence;
	std::vector<float *> inPtr, outPtr; // two channel pointers a bus
	std::vector<AudioBusBuffers> inBuses, outBuses;

	// a block
	HostEvents *events = nullptr, *outEvents = nullptr;
	HostChanges *inChanges = nullptr, *outChanges = nullptr;
	ProcessContext ctx;
	bool hasTransport = false;
	int64 samplesDone = 0;
	std::mutex guiMu; // what the controller did (a knob turned in the window), for the processor and for the editor
	std::vector<std::pair<ParamID, double>> guiEdits, goChanges, ctrlSets;
	std::atomic<int> restartFlags{0};
	volatile int dead = 0;

	// the window
	IPlugView *view = nullptr;
	HostFrame *frame = nullptr;
	Display *dpy = nullptr;
	Window win = 0;
	Atom wmDelete = 0;
	bool guiCreated = false, guiVisible = false;
	uint32_t gw = 0, gh = 0;
	struct FdReg {
		Linux::IEventHandler *h;
		int fd;
	};
	struct TimerReg {
		Linux::ITimerHandler *h;
		uint64_t period;
		int64_t next;
	};
	std::vector<FdReg> fds;
	std::vector<TimerReg> timers;
	std::mutex rlMu;
};

static bool inst_dead(v3_inst *inst) { return inst && inst->dead; }
static void mark_dead(v3_inst *inst, int sig) {
	if (inst) inst->dead = sig;
}

// the registry of the instances, for the loop of the main thread
static std::mutex g_mu;
static std::vector<v3_inst *> g_insts;
static pthread_t g_main;
static bool g_main_set = false;
static int g_wake[2] = {-1, -1};

class HostHandler : public Obj<IComponentHandler, IComponentHandler2> {
public:
	v3_inst *inst;
	explicit HostHandler(v3_inst *i) : inst(i) {}
	tresult PLUGIN_API beginEdit(ParamID) override { return kResultOk; }
	tresult PLUGIN_API performEdit(ParamID id, ParamValue value) override {
		std::lock_guard<std::mutex> lock(inst->guiMu);
		inst->guiEdits.emplace_back(id, value);
		inst->goChanges.emplace_back(id, value);
		return kResultOk;
	}
	tresult PLUGIN_API endEdit(ParamID) override { return kResultOk; }
	tresult PLUGIN_API restartComponent(int32 flags) override {
		inst->restartFlags |= flags;
		v3_wake();
		return kResultOk;
	}
	tresult PLUGIN_API setDirty(TBool) override { return kResultOk; }
	tresult PLUGIN_API requestOpenEditor(FIDString) override { return kResultFalse; }
	tresult PLUGIN_API startGroupEdit() override { return kResultOk; }
	tresult PLUGIN_API finishGroupEdit() override { return kResultOk; }
};

static int64_t now_ms() {
	struct timespec ts;
	clock_gettime(CLOCK_MONOTONIC, &ts);
	return (int64_t)ts.tv_sec * 1000 + ts.tv_nsec / 1000000;
}

// the frame of the window of a plugin: it asks the host to resize the view, and (on Linux) for the host's run loop, which runs its timers and descriptors
class HostFrame : public Obj<IPlugFrame, Linux::IRunLoop> {
public:
	v3_inst *inst;
	explicit HostFrame(v3_inst *i) : inst(i) {}
	tresult PLUGIN_API resizeView(IPlugView *view, ViewRect *r) override {
		if (!inst->dpy || !inst->win || !r) return kResultFalse;
		inst->gw = (uint32_t)(r->right - r->left);
		inst->gh = (uint32_t)(r->bottom - r->top);
		XResizeWindow(inst->dpy, inst->win, inst->gw, inst->gh);
		XFlush(inst->dpy);
		ViewRect now = *r;
		view->onSize(&now);
		return kResultOk;
	}
	tresult PLUGIN_API registerEventHandler(Linux::IEventHandler *h, Linux::FileDescriptor fd) override {
		std::lock_guard<std::mutex> lock(inst->rlMu);
		h->addRef();
		inst->fds.push_back({h, fd});
		v3_wake();
		return kResultOk;
	}
	tresult PLUGIN_API unregisterEventHandler(Linux::IEventHandler *h) override {
		std::lock_guard<std::mutex> lock(inst->rlMu);
		for (size_t i = 0; i < inst->fds.size();) {
			if (inst->fds[i].h == h) {
				inst->fds[i].h->release();
				inst->fds.erase(inst->fds.begin() + i);
			} else {
				i++;
			}
		}
		return kResultOk;
	}
	tresult PLUGIN_API registerTimer(Linux::ITimerHandler *h, Linux::TimerInterval ms) override {
		std::lock_guard<std::mutex> lock(inst->rlMu);
		h->addRef();
		inst->timers.push_back({h, std::max<uint64_t>(ms, 5), now_ms() + (int64_t)std::max<uint64_t>(ms, 5)});
		v3_wake();
		return kResultOk;
	}
	tresult PLUGIN_API unregisterTimer(Linux::ITimerHandler *h) override {
		std::lock_guard<std::mutex> lock(inst->rlMu);
		for (size_t i = 0; i < inst->timers.size();) {
			if (inst->timers[i].h == h) {
				inst->timers[i].h->release();
				inst->timers.erase(inst->timers.begin() + i);
			} else {
				i++;
			}
		}
		return kResultOk;
	}
};

// --- the main thread ---

void v3_set_main(void) {
	g_main = pthread_self();
	g_main_set = true;
	if (pipe(g_wake) != 0) g_wake[0] = g_wake[1] = -1;
}
int v3_is_main(void) { return g_main_set && pthread_equal(pthread_self(), g_main); }
void v3_wake(void) {
	if (g_wake[1] >= 0) {
		char c = 'w';
		(void)!write(g_wake[1], &c, 1);
	}
}

// --- loading ---

static v3_lib *v3_open_in(const char *bundle, char *err, int errcap) {
	std::string base = bundle;
	// the bundle is a folder with the library in Contents/x86_64-linux/<name>.so; a path of the library itself is taken too
	std::string so = base;
	if (base.size() > 5 && base.compare(base.size() - 5, 5, ".vst3") == 0) {
		std::string name = base.substr(base.rfind('/') == std::string::npos ? 0 : base.rfind('/') + 1);
		name = name.substr(0, name.size() - 5);
		so = base + "/Contents/x86_64-linux/" + name + ".so";
	}
	void *h = dlopen(so.c_str(), RTLD_NOW | RTLD_LOCAL);
	if (!h) {
		set_err(err, errcap, dlerror());
		return nullptr;
	}
	typedef bool (*EntryFn)(void *);
	typedef bool (*ExitFn)();
	typedef IPluginFactory *(*FactoryFn)();
	if (auto entry = (EntryFn)dlsym(h, "ModuleEntry")) {
		if (!entry(h)) {
			set_err(err, errcap, "ModuleEntry said no");
			dlclose(h);
			return nullptr;
		}
	}
	auto getFactory = (FactoryFn)dlsym(h, "GetPluginFactory");
	if (!getFactory) {
		set_err(err, errcap, "no GetPluginFactory");
		dlclose(h);
		return nullptr;
	}
	IPluginFactory *f = getFactory();
	if (!f) {
		set_err(err, errcap, "no plugin factory");
		dlclose(h);
		return nullptr;
	}
	v3_lib *lib = new v3_lib;
	lib->handle = h;
	lib->factory = f;
	lib->exitFn = (ExitFn)dlsym(h, "ModuleExit");
	lib->path = bundle;
	return lib;
}

static int v3_class_count_in(v3_lib *lib) { return lib->factory->countClasses(); }

static int v3_class_in(v3_lib *lib, int i, char *cid, char *name, char *vendor, char *categories, int cap) {
	PClassInfo info;
	if (lib->factory->getClassInfo(i, &info) != kResultOk) return 0;
	if (strcmp(info.category, kVstAudioEffectClass) != 0) return 0;
	snprintf(cid, cap, "%s", cid_to_hex(info.cid).c_str());
	snprintf(name, cap, "%s", info.name);
	vendor[0] = 0;
	categories[0] = 0;
	if (IPluginFactory2 *f2 = nullptr; lib->factory->queryInterface(IPluginFactory2::iid.toTUID(), (void **)&f2) == kResultOk && f2) {
		PClassInfo2 i2;
		if (f2->getClassInfo2(i, &i2) == kResultOk) {
			snprintf(vendor, cap, "%s", i2.vendor);
			snprintf(categories, cap, "%s", i2.subCategories);
		}
		f2->release();
	}
	return 1;
}

// --- an instance ---

static void v3_destroy(v3_inst *inst);

static v3_inst *v3_new_in(v3_lib *lib, const char *cidHex, double sr, uint32_t maxFrames, char *err, int errcap) {
	TUID cid;
	if (!hex_to_cid(cidHex, cid)) {
		set_err(err, errcap, "not a class id");
		return nullptr;
	}
	v3_inst *inst = new v3_inst;
	inst->lib = lib;
	inst->sr = sr;
	inst->maxFrames = maxFrames;
	inst->app = new HostApp;
	inst->handler = new HostHandler(inst);
	inst->events = new HostEvents;
	inst->outEvents = new HostEvents;
	inst->inChanges = new HostChanges;
	inst->outChanges = new HostChanges;
	FUnknown *context = static_cast<FUnknown *>(static_cast<IHostApplication *>(inst->app));

	if (lib->factory->createInstance(cid, IComponent::iid.toTUID(), (void **)&inst->comp) != kResultOk || !inst->comp) {
		set_err(err, errcap, "the factory would not make the component");
		v3_destroy(inst);
		return nullptr;
	}
	if (inst->comp->initialize(context) != kResultOk) {
		set_err(err, errcap, "the component would not initialise");
		inst->comp->release();
		inst->comp = nullptr;
		v3_destroy(inst);
		return nullptr;
	}
	if (inst->comp->queryInterface(IAudioProcessor::iid.toTUID(), (void **)&inst->proc) != kResultOk || !inst->proc) {
		set_err(err, errcap, "the component has no audio processor");
		v3_destroy(inst);
		return nullptr;
	}
	// the controller: the component itself, or a class of its own
	if (inst->comp->queryInterface(IEditController::iid.toTUID(), (void **)&inst->ctrl) != kResultOk) inst->ctrl = nullptr;
	if (!inst->ctrl) {
		TUID ccid;
		if (inst->comp->getControllerClassId(ccid) == kResultOk &&
		    lib->factory->createInstance(ccid, IEditController::iid.toTUID(), (void **)&inst->ctrl) == kResultOk && inst->ctrl) {
			inst->ctrlSeparate = true;
			if (inst->ctrl->initialize(context) != kResultOk) {
				set_err(err, errcap, "the controller would not initialise");
				v3_destroy(inst);
				return nullptr;
			}
		}
	}
	if (inst->ctrl) {
		inst->comp->queryInterface(IConnectionPoint::iid.toTUID(), (void **)&inst->cpComp);
		inst->ctrl->queryInterface(IConnectionPoint::iid.toTUID(), (void **)&inst->cpCtrl);
		if (inst->cpComp && inst->cpCtrl && inst->cpComp != inst->cpCtrl) {
			inst->cpComp->connect(inst->cpCtrl);
			inst->cpCtrl->connect(inst->cpComp);
		}
		inst->ctrl->setComponentHandler(static_cast<IComponentHandler *>(inst->handler));
		MemStream state;
		if (inst->comp->getState(&state) == kResultOk) {
			state.pos = 0;
			inst->ctrl->setComponentState(&state);
		}
	}

	if (inst->ctrl) {
		INoteExpressionController *ne = nullptr;
		if (inst->ctrl->queryInterface(INoteExpressionController::iid.toTUID(), (void **)&ne) == kResultOk && ne) {
			int32 n = ne->getNoteExpressionCount(0, 0);
			for (int32 i = 0; i < n; i++) {
				NoteExpressionTypeInfo info{};
				if (ne->getNoteExpressionInfo(0, 0, i, info) == kResultOk && info.typeId == kTuningTypeID) inst->tuningExpression = true;
			}
			ne->release();
		}
	}

	if (getenv("PSALM_VST3_DEBUG")) fprintf(stderr, "vst3: note expression tuning: %d\n", (int)inst->tuningExpression);

	// the buses: stereo, the first two inputs (the second a side chain) and every output (up to 16) active, the others as they are
	IComponent *c = inst->comp;
	int inAll = c->getBusCount(kAudio, kInput), outAll = c->getBusCount(kAudio, kOutput);
	std::vector<SpeakerArrangement> ia(inAll), oa(outAll);
	for (int i = 0; i < inAll; i++) inst->proc->getBusArrangement(kInput, i, ia[i]);
	for (int i = 0; i < outAll; i++) inst->proc->getBusArrangement(kOutput, i, oa[i]);
	for (int i = 0; i < std::min(inAll, 2); i++) ia[i] = SpeakerArr::kStereo;
	for (int i = 0; i < std::min(outAll, 16); i++) oa[i] = SpeakerArr::kStereo;
	if (inst->proc->setBusArrangements(inAll ? ia.data() : nullptr, inAll, outAll ? oa.data() : nullptr, outAll) != kResultOk) {
		for (int i = 0; i < inAll; i++) inst->proc->getBusArrangement(kInput, i, ia[i]);
		for (int i = 0; i < outAll; i++) inst->proc->getBusArrangement(kOutput, i, oa[i]);
	}
	inst->nIn = inAll;
	inst->nOut = outAll;
	for (int i = 0; i < inAll; i++) {
		BusInfo bi{};
		c->getBusInfo(kAudio, kInput, i, bi);
		char name[256];
		utf16_to_utf8(bi.name, name, sizeof name);
		inst->inName.push_back(name);
		inst->inCh.push_back(std::min(2, (int)SpeakerArr::getChannelCount(ia[i])));
		c->activateBus(kAudio, kInput, i, i < 2);
	}
	for (int i = 0; i < outAll; i++) {
		BusInfo bi{};
		c->getBusInfo(kAudio, kOutput, i, bi);
		char name[256];
		utf16_to_utf8(bi.name, name, sizeof name);
		inst->outName.push_back(name);
		inst->outCh.push_back(std::min(2, (int)SpeakerArr::getChannelCount(oa[i])));
		c->activateBus(kAudio, kOutput, i, i < 16);
	}
	for (int i = 0; i < c->getBusCount(kEvent, kInput); i++) c->activateBus(kEvent, kInput, i, true);
	for (int i = 0; i < c->getBusCount(kEvent, kOutput); i++) c->activateBus(kEvent, kOutput, i, true);

	// the buffers: two channels for every bus
	size_t fr = maxFrames;
	inst->inbuf.assign((size_t)std::max(inAll, 1) * 2 * fr, 0.f);
	inst->outbuf.assign((size_t)std::max(outAll, 1) * 2 * fr, 0.f);
	inst->silence.assign(fr, 0.f);
	inst->inPtr.assign((size_t)std::max(inAll, 1) * 2, nullptr);
	inst->outPtr.assign((size_t)std::max(outAll, 1) * 2, nullptr);
	for (int i = 0; i < inAll * 2; i++) inst->inPtr[i] = inst->inbuf.data() + (size_t)i * fr;
	for (int i = 0; i < outAll * 2; i++) inst->outPtr[i] = inst->outbuf.data() + (size_t)i * fr;
	inst->inBuses.assign(std::max(inAll, 1), AudioBusBuffers());
	inst->outBuses.assign(std::max(outAll, 1), AudioBusBuffers());

	ProcessSetup setup;
	setup.processMode = kRealtime;
	setup.symbolicSampleSize = kSample32;
	setup.maxSamplesPerBlock = (int32)maxFrames;
	setup.sampleRate = sr;
	if (inst->proc->setupProcessing(setup) != kResultOk) {
		set_err(err, errcap, "the plugin would not be set up for processing");
		v3_destroy(inst);
		return nullptr;
	}
	if (inst->comp->setActive(true) != kResultOk) {
		set_err(err, errcap, "the plugin would not become active");
		v3_destroy(inst);
		return nullptr;
	}
	inst->active = true;
	inst->proc->setProcessing(true);
	inst->processing = true;
	memset(&inst->ctx, 0, sizeof inst->ctx);
	inst->ctx.sampleRate = sr;
	{
		std::lock_guard<std::mutex> lock(g_mu);
		g_insts.push_back(inst);
	}
	return inst;
}

static void gui_destroy(v3_inst *inst);

static void v3_free_in(v3_inst *inst) {
	if (!inst) return;
	{
		std::lock_guard<std::mutex> lock(g_mu);
		g_insts.erase(std::remove(g_insts.begin(), g_insts.end(), inst), g_insts.end());
	}
	gui_destroy(inst);
	v3_destroy(inst);
}

// takes what an instance has apart, whatever it got to
static void v3_destroy(v3_inst *inst) {
	if (inst->proc && inst->processing) inst->proc->setProcessing(false);
	if (inst->comp && inst->active) inst->comp->setActive(false);
	if (inst->cpComp && inst->cpCtrl && inst->cpComp != inst->cpCtrl) {
		inst->cpComp->disconnect(inst->cpCtrl);
		inst->cpCtrl->disconnect(inst->cpComp);
	}
	if (inst->cpComp) inst->cpComp->release();
	if (inst->cpCtrl) inst->cpCtrl->release();
	if (inst->ctrl) {
		inst->ctrl->setComponentHandler(nullptr);
		if (inst->ctrlSeparate) {
			inst->ctrl->terminate();
			inst->ctrl->release();
		} else {
			inst->ctrl->release();
		}
	}
	if (inst->proc) inst->proc->release();
	if (inst->comp) {
		inst->comp->terminate();
		inst->comp->release();
	}
	if (inst->handler) inst->handler->release();
	if (inst->app) inst->app->release();
	if (inst->events) inst->events->release();
	if (inst->outEvents) inst->outEvents->release();
	if (inst->inChanges) inst->inChanges->release();
	if (inst->outChanges) inst->outChanges->release();
	delete inst;
}

static void v3_bus_counts_in(v3_inst *inst, int *in, int *out) {
	*in = inst->nIn;
	*out = inst->nOut;
}

static int v3_bus_info_in(v3_inst *inst, int is_input, int i, int *channels, char *name, int cap) {
	auto &ch = is_input ? inst->inCh : inst->outCh;
	auto &nm = is_input ? inst->inName : inst->outName;
	if (i < 0 || i >= (int)ch.size()) return 0;
	*channels = ch[i];
	snprintf(name, cap, "%s", nm[i].c_str());
	return 1;
}

static int v3_param_count_in(v3_inst *inst) { return inst->ctrl ? inst->ctrl->getParameterCount() : 0; }

static int v3_param_info_in(v3_inst *inst, int i, uint32_t *id, char *title, char *units, int cap, int *steps, double *def, int *flags, int *unit) {
	ParameterInfo pi{};
	if (!inst->ctrl || inst->ctrl->getParameterInfo(i, pi) != kResultOk) return 0;
	*id = pi.id;
	utf16_to_utf8(pi.title, title, cap);
	utf16_to_utf8(pi.units, units, cap);
	*steps = pi.stepCount;
	*def = pi.defaultNormalizedValue;
	*flags = pi.flags;
	*unit = pi.unitId;
	return 1;
}

static double v3_param_value_in(v3_inst *inst, uint32_t id) { return inst->ctrl ? inst->ctrl->getParamNormalized(id) : 0.0; }

static int v3_param_text_in(v3_inst *inst, uint32_t id, double value, char *out, int cap) {
	if (!inst->ctrl) return 0;
	String128 s;
	s[0] = 0;
	if (inst->ctrl->getParamStringByValue(id, value, s) != kResultOk || !s[0]) return 0;
	utf16_to_utf8(s, out, cap);
	return 1;
}

static uint32_t v3_latency_in(v3_inst *inst) { return inst->proc ? inst->proc->getLatencySamples() : 0; }

// --- the events and parameter values of a block: made on the audio thread, kept until the next process ---

void v3_note(v3_inst *inst, int on, uint32_t offset, int pitch, double velocity, int32_t noteID, double tuningCents) {
	Event e;
	memset(&e, 0, sizeof e);
	e.busIndex = 0;
	e.sampleOffset = (int32)offset;
	if (on) {
		e.type = Event::kNoteOnEvent;
		e.noteOn.channel = 0;
		e.noteOn.pitch = (int16)pitch;
		e.noteOn.tuning = inst->tuningExpression ? 0.f : (float)tuningCents;
		e.noteOn.velocity = (float)velocity;
		e.noteOn.length = 0;
		e.noteOn.noteId = noteID;
		if (inst->events->list.size() < 4096) inst->events->list.push_back(e);
		if (inst->tuningExpression && tuningCents != 0) { // between the keys: the tuning of the note, in the semitones of its range (-120 to +120, 0.5 none)
			Event x;
			memset(&x, 0, sizeof x);
			x.busIndex = 0;
			x.sampleOffset = (int32)offset;
			x.type = Event::kNoteExpressionValueEvent;
			x.noteExpressionValue.typeId = kTuningTypeID;
			x.noteExpressionValue.noteId = noteID;
			x.noteExpressionValue.value = 0.5 + tuningCents / 100.0 / 240.0;
			e = x;
		} else {
			return;
		}
	} else {
		e.type = Event::kNoteOffEvent;
		e.noteOff.channel = 0;
		e.noteOff.pitch = (int16)pitch;
		e.noteOff.velocity = 0;
		e.noteOff.noteId = noteID;
		e.noteOff.tuning = (float)tuningCents;
	}
	if (inst->events->list.size() < 4096) inst->events->list.push_back(e);
}

void v3_param(v3_inst *inst, uint32_t id, double value, uint32_t offset) {
	int32 index = 0, pt = 0;
	if (IParamValueQueue *q = inst->inChanges->addParameterData(id, index)) q->addPoint((int32)offset, value, pt);
	std::lock_guard<std::mutex> lock(inst->guiMu); // the controller is told on the main thread
	inst->ctrlSets.emplace_back(id, value);
}

// sets a parameter on the controller now (on the main thread), and has the processor told
void v3_param_set_now(v3_inst *inst, uint32_t id, double value) {
	SAFE(inst, if (inst->ctrl) inst->ctrl->setParamNormalized(id, value));
}

void v3_transport(v3_inst *inst, double tempo, double beats, int playing) {
	ProcessContext &c = inst->ctx;
	c.state = ProcessContext::kTempoValid | ProcessContext::kTimeSigValid | ProcessContext::kProjectTimeMusicValid | (playing ? ProcessContext::kPlaying : 0);
	c.tempo = tempo;
	c.timeSigNumerator = 4;
	c.timeSigDenominator = 4;
	c.projectTimeMusic = beats;
	inst->hasTransport = true;
}

float *v3_out(v3_inst *inst, int bus, int channel) {
	if (bus < 0 || bus >= inst->nOut || channel < 0 || channel > 1) return nullptr;
	return inst->outPtr[(size_t)bus * 2 + channel];
}

static int v3_process_in(v3_inst *inst, float *in0, float *in1, float *sc0, float *sc1, uint32_t frames) {
	if (frames > inst->maxFrames || !inst->proc) return 0;
	{ // what the controller did (a knob turned in the plugin's window): to the processor
		std::lock_guard<std::mutex> lock(inst->guiMu);
		for (auto &e : inst->guiEdits) {
			int32 index = 0, pt = 0;
			if (IParamValueQueue *q = inst->inChanges->addParameterData(e.first, index)) q->addPoint(0, e.second, pt);
		}
		inst->guiEdits.clear();
	}
	std::stable_sort(inst->events->list.begin(), inst->events->list.end(), [](const Event &a, const Event &b) { return a.sampleOffset < b.sampleOffset; });
	for (auto &e : inst->events->list) e.sampleOffset = std::min<int32>(e.sampleOffset, (int32)frames - 1);

	size_t fr = inst->maxFrames;
	for (int b = 0; b < inst->nIn; b++) {
		AudioBusBuffers &buf = inst->inBuses[b];
		buf = AudioBusBuffers();
		if (b >= 2) continue; // an input bus beyond the two: inactive
		float *left = b == 0 ? in0 : sc0, *right = b == 0 ? in1 : sc1;
		buf.numChannels = inst->inCh[b];
		buf.channelBuffers32 = &inst->inPtr[(size_t)b * 2];
		if (left) {
			memcpy(inst->inPtr[(size_t)b * 2], left, frames * sizeof(float));
			memcpy(inst->inPtr[(size_t)b * 2 + 1], right ? right : left, frames * sizeof(float));
		} else {
			memset(inst->inPtr[(size_t)b * 2], 0, frames * sizeof(float));
			memset(inst->inPtr[(size_t)b * 2 + 1], 0, frames * sizeof(float));
			buf.silenceFlags = 3;
		}
	}
	for (int b = 0; b < inst->nOut; b++) {
		AudioBusBuffers &buf = inst->outBuses[b];
		buf = AudioBusBuffers();
		if (b >= 16) continue;
		buf.numChannels = inst->outCh[b];
		buf.channelBuffers32 = &inst->outPtr[(size_t)b * 2];
		memset(inst->outPtr[(size_t)b * 2], 0, frames * sizeof(float));
		memset(inst->outPtr[(size_t)b * 2 + 1], 0, frames * sizeof(float));
	}
	(void)fr;
	inst->ctx.continousTimeSamples = inst->samplesDone;
	inst->ctx.systemTime = 0;
	if (inst->hasTransport) inst->ctx.state |= ProcessContext::kContTimeValid;
	ProcessData d;
	d.processMode = kRealtime;
	d.symbolicSampleSize = kSample32;
	d.numSamples = (int32)frames;
	d.numInputs = inst->nIn;
	d.numOutputs = inst->nOut;
	d.inputs = inst->nIn ? inst->inBuses.data() : nullptr;
	d.outputs = inst->nOut ? inst->outBuses.data() : nullptr;
	d.inputParameterChanges = inst->inChanges;
	d.outputParameterChanges = inst->outChanges;
	d.inputEvents = inst->events;
	d.outputEvents = inst->outEvents;
	d.processContext = inst->hasTransport ? &inst->ctx : nullptr;
	tresult r = inst->proc->process(d);
	inst->samplesDone += frames;
	inst->events->list.clear();
	inst->outEvents->list.clear();
	inst->inChanges->clear();
	inst->outChanges->clear();
	for (int b = 0; b < inst->nOut; b++) {
		if (b < 16 && inst->outCh[b] == 1) memcpy(inst->outPtr[(size_t)b * 2 + 1], inst->outPtr[(size_t)b * 2], frames * sizeof(float));
	}
	return r == kResultOk;
}

int v3_take_changes(v3_inst *inst, uint32_t *ids, double *values, int max) {
	std::lock_guard<std::mutex> lock(inst->guiMu);
	int n = 0;
	while (n < max && n < (int)inst->goChanges.size()) {
		ids[n] = inst->goChanges[n].first;
		values[n] = inst->goChanges[n].second;
		n++;
	}
	inst->goChanges.erase(inst->goChanges.begin(), inst->goChanges.begin() + n);
	return n;
}

int v3_take_restart(v3_inst *inst) { return inst->restartFlags.exchange(0); }

static uint8_t *copy_bytes(const std::vector<uint8_t> &v, size_t *len) {
	*len = v.size();
	uint8_t *p = (uint8_t *)malloc(v.size() + 1);
	if (!v.empty()) memcpy(p, v.data(), v.size());
	return p;
}

static int v3_state_save_in(v3_inst *inst, uint8_t **comp, size_t *complen, uint8_t **ctrl, size_t *ctrllen) {
	MemStream cs;
	if (inst->comp->getState(&cs) != kResultOk) return 0;
	*comp = copy_bytes(cs.data, complen);
	*ctrl = nullptr;
	*ctrllen = 0;
	if (inst->ctrl && inst->ctrlSeparate) {
		MemStream ts;
		if (inst->ctrl->getState(&ts) == kResultOk) *ctrl = copy_bytes(ts.data, ctrllen);
	}
	return 1;
}

static int v3_state_load_in(v3_inst *inst, const uint8_t *comp, size_t complen, const uint8_t *ctrl, size_t ctrllen) {
	MemStream cs(comp, complen);
	if (inst->comp->setState(&cs) != kResultOk) return 0;
	if (inst->ctrl) {
		cs.pos = 0;
		inst->ctrl->setComponentState(&cs);
		if (inst->ctrlSeparate && ctrl && ctrllen) {
			MemStream ts(ctrl, ctrllen);
			inst->ctrl->setState(&ts);
		}
	}
	return 1;
}

void v3_free_bytes(void *p) { free(p); }

// --- the programs of a plugin: the parameter that selects one, and their names ---

static int v3_programs_in(v3_inst *inst, uint32_t *param, int *steps) {
	IUnitInfo *ui = nullptr;
	if (!inst->ctrl || inst->ctrl->queryInterface(IUnitInfo::iid.toTUID(), (void **)&ui) != kResultOk || !ui) return 0;
	int count = 0;
	UnitID unit = -1;
	int32 n = inst->ctrl->getParameterCount();
	bool found = false;
	for (int32 i = 0; i < n; i++) {
		ParameterInfo pi{};
		if (inst->ctrl->getParameterInfo(i, pi) == kResultOk && (pi.flags & ParameterInfo::kIsProgramChange)) {
			*param = pi.id;
			*steps = pi.stepCount;
			unit = pi.unitId;
			found = true;
			break;
		}
	}
	if (found) {
		ProgramListID list = kNoProgramListId;
		for (int32 u = 0; u < ui->getUnitCount(); u++) {
			UnitInfo info{};
			if (ui->getUnitInfo(u, info) == kResultOk && info.id == unit) list = info.programListId;
		}
		for (int32 l = 0; l < ui->getProgramListCount(); l++) {
			ProgramListInfo info{};
			if (ui->getProgramListInfo(l, info) == kResultOk && info.id == list) count = info.programCount;
		}
		inst->progList = list;
	}
	ui->release();
	return count;
}

static int v3_program_name_in(v3_inst *inst, int index, char *name, int cap) {
	IUnitInfo *ui = nullptr;
	if (!inst->ctrl || inst->ctrl->queryInterface(IUnitInfo::iid.toTUID(), (void **)&ui) != kResultOk || !ui) return 0;
	String128 s;
	s[0] = 0;
	bool ok = ui->getProgramName(inst->progList, index, s) == kResultOk;
	ui->release();
	if (!ok) return 0;
	utf16_to_utf8(s, name, cap);
	return 1;
}

// the name of a unit (a group of parameters), with the units it is in: "Oscillators / Osc 1"
static int v3_unit_name_in(v3_inst *inst, int unit, char *name, int cap) {
	IUnitInfo *ui = nullptr;
	name[0] = 0;
	if (!inst->ctrl || inst->ctrl->queryInterface(IUnitInfo::iid.toTUID(), (void **)&ui) != kResultOk || !ui) return 0;
	std::string out;
	UnitID id = unit;
	for (int depth = 0; depth < 6 && id != kRootUnitId && id != kNoParentUnitId; depth++) {
		UnitInfo info{};
		bool found = false;
		for (int32 u = 0; u < ui->getUnitCount(); u++) {
			if (ui->getUnitInfo(u, info) == kResultOk && info.id == id) {
				found = true;
				break;
			}
		}
		if (!found) break;
		char part[256];
		utf16_to_utf8(info.name, part, sizeof part);
		out = out.empty() ? std::string(part) : std::string(part) + " / " + out;
		id = info.parentUnitId;
	}
	ui->release();
	snprintf(name, cap, "%s", out.c_str());
	return out.empty() ? 0 : 1;
}

// --- the window of a plugin ---

static bool gui_available_in(v3_inst *inst) {
	if (!inst->ctrl) return false;
	if (inst->guiCreated) return true;
	IPlugView *v = inst->ctrl->createView(ViewType::kEditor);
	if (getenv("PSALM_VST3_DEBUG")) fprintf(stderr, "vst3: createView gave %p\n", (void *)v);
	if (!v) return false;
	tresult supported = v->isPlatformTypeSupported(kPlatformTypeX11EmbedWindowID);
	if (getenv("PSALM_VST3_DEBUG")) fprintf(stderr, "vst3: X11EmbedWindowID supported: %d\n", (int)supported);
	bool ok = supported == kResultTrue;
	v->release();
	return ok;
}

// makes the window if there is none (a window of ours, the view of the plugin embedded in it) and shows it
static int gui_open(v3_inst *inst) {
	if (!inst->guiCreated) {
		if (!inst->ctrl) return 0;
		inst->dpy = XOpenDisplay(nullptr);
		if (!inst->dpy) return 0;
		IPlugView *v = inst->ctrl->createView(ViewType::kEditor);
		if (!v) return 0;
		if (v->isPlatformTypeSupported(kPlatformTypeX11EmbedWindowID) != kResultTrue) {
			v->release();
			return 0;
		}
		ViewRect r{};
		v->getSize(&r);
		uint32_t w = (uint32_t)std::max(100, r.right - r.left), h = (uint32_t)std::max(60, r.bottom - r.top);
		inst->view = v;
		inst->frame = new HostFrame(inst);
		inst->gw = w;
		inst->gh = h;
		inst->win = XCreateSimpleWindow(inst->dpy, DefaultRootWindow(inst->dpy), 0, 0, w, h, 0, 0, 0);
		XStoreName(inst->dpy, inst->win, inst->lib->path.c_str());
		inst->wmDelete = XInternAtom(inst->dpy, "WM_DELETE_WINDOW", False);
		XSetWMProtocols(inst->dpy, inst->win, &inst->wmDelete, 1);
		XSelectInput(inst->dpy, inst->win, StructureNotifyMask);
		XFlush(inst->dpy);
		v->setFrame(static_cast<IPlugFrame *>(inst->frame));
		if (v->attached((void *)(uintptr_t)inst->win, kPlatformTypeX11EmbedWindowID) != kResultOk) {
			v->setFrame(nullptr);
			v->release();
			inst->view = nullptr;
			inst->frame->release();
			inst->frame = nullptr;
			XDestroyWindow(inst->dpy, inst->win);
			inst->win = 0;
			return 0;
		}
		inst->guiCreated = true;
	}
	XMapWindow(inst->dpy, inst->win);
	XFlush(inst->dpy);
	inst->guiVisible = true;
	return 1;
}

static void gui_close(v3_inst *inst) {
	if (inst->guiCreated && inst->dpy && inst->win) {
		XUnmapWindow(inst->dpy, inst->win);
		XFlush(inst->dpy);
	}
	inst->guiVisible = false;
}

static void gui_destroy(v3_inst *inst) {
	if (inst->view) {
		inst->view->removed();
		inst->view->setFrame(nullptr);
		inst->view->release();
		inst->view = nullptr;
	}
	if (inst->frame) {
		inst->frame->release();
		inst->frame = nullptr;
	}
	{
		std::lock_guard<std::mutex> lock(inst->rlMu);
		for (auto &f : inst->fds) f.h->release();
		for (auto &t : inst->timers) t.h->release();
		inst->fds.clear();
		inst->timers.clear();
	}
	if (inst->dpy) {
		if (inst->win) XDestroyWindow(inst->dpy, inst->win);
		XCloseDisplay(inst->dpy);
		inst->dpy = nullptr;
		inst->win = 0;
	}
	inst->guiCreated = false;
	inst->guiVisible = false;
}

// what the plugins asked of the main thread: waits for the wake, a descriptor, a timer or the events of a window (at most timeout_ms), and does it
void v3_pump(int timeout_ms) {
	std::vector<v3_inst *> insts;
	{
		std::lock_guard<std::mutex> lock(g_mu);
		insts = g_insts;
	}
	std::vector<struct pollfd> pfd;
	std::vector<v3_inst *> owner;
	std::vector<int> kind; // 0 the wake pipe, 1 an X connection, 2 a descriptor of a plugin
	pfd.push_back({g_wake[0], POLLIN, 0});
	owner.push_back(nullptr);
	kind.push_back(0);
	int64_t now = now_ms();
	int wait = timeout_ms;
	for (v3_inst *inst : insts) {
		if (inst->dead) continue;
		{
			std::lock_guard<std::mutex> lock(inst->guiMu);
			if (!inst->ctrlSets.empty()) wait = 0;
		}
		if (inst->dpy) {
			if (XPending(inst->dpy) > 0) wait = 0;
			pfd.push_back({ConnectionNumber(inst->dpy), POLLIN, 0});
			owner.push_back(inst);
			kind.push_back(1);
		}
		std::lock_guard<std::mutex> lock(inst->rlMu);
		for (auto &f : inst->fds) {
			pfd.push_back({f.fd, POLLIN, 0});
			owner.push_back(inst);
			kind.push_back(2);
		}
		for (auto &t : inst->timers) {
			int64_t left = t.next - now;
			if (left < wait) wait = left < 0 ? 0 : (int)left;
		}
	}
	poll(pfd.data(), pfd.size(), wait);
	if (pfd[0].revents & POLLIN) {
		char buf[64];
		(void)!read(g_wake[0], buf, sizeof buf);
	}
	for (size_t i = 1; i < pfd.size(); i++) {
		v3_inst *inst = owner[i];
		if (kind[i] == 2 && pfd[i].revents && !inst->dead) {
			Linux::IEventHandler *h = nullptr;
			{
				std::lock_guard<std::mutex> lock(inst->rlMu);
				for (auto &f : inst->fds) {
					if (f.fd == pfd[i].fd) {
						h = f.h;
						h->addRef();
						break;
					}
				}
			}
			if (h) {
				SAFE(inst, h->onFDIsSet(pfd[i].fd));
				h->release();
			}
		}
	}
	now = now_ms();
	for (v3_inst *inst : insts) {
		if (inst->dead) continue;
		std::vector<std::pair<ParamID, double>> sets;
		{
			std::lock_guard<std::mutex> lock(inst->guiMu);
			sets.swap(inst->ctrlSets);
		}
		for (auto &s : sets) {
			if (inst->ctrl) SAFE(inst, inst->ctrl->setParamNormalized(s.first, s.second));
		}
		std::vector<Linux::ITimerHandler *> due;
		{
			std::lock_guard<std::mutex> lock(inst->rlMu);
			for (auto &t : inst->timers) {
				if (t.next <= now) {
					t.next = now + (int64_t)t.period;
					t.h->addRef();
					due.push_back(t.h);
				}
			}
		}
		for (auto h : due) {
			SAFE(inst, h->onTimer());
			h->release();
		}
		while (inst->dpy && XPending(inst->dpy) > 0) {
			XEvent ev;
			XNextEvent(inst->dpy, &ev);
			if (ev.type == ClientMessage && (Atom)ev.xclient.data.l[0] == inst->wmDelete) {
				gui_close(inst);
			} else if (ev.type == ConfigureNotify && ev.xconfigure.window == inst->win && inst->view) {
				uint32_t w = ev.xconfigure.width, h = ev.xconfigure.height;
				if ((w != inst->gw || h != inst->gh) && inst->view->canResize() == kResultTrue) {
					ViewRect r{0, 0, (int32)w, (int32)h};
					inst->view->checkSizeConstraint(&r);
					inst->gw = (uint32_t)(r.right - r.left);
					inst->gh = (uint32_t)(r.bottom - r.top);
					SAFE(inst, inst->view->onSize(&r));
				}
			}
		}
	}
}

// --- the public calls: each is guarded, a plugin that crashes in it is dead and answers nothing ---

v3_lib *v3_open(const char *bundle, char *err, int errcap) { GUARD(nullptr, v3_open_in(bundle, err, errcap), nullptr); }
int v3_class_count(v3_lib *lib) { GUARD(nullptr, v3_class_count_in(lib), 0); }
int v3_class(v3_lib *lib, int i, char *cid, char *name, char *vendor, char *categories, int cap) { GUARD(nullptr, v3_class_in(lib, i, cid, name, vendor, categories, cap), 0); }
v3_inst *v3_new(v3_lib *lib, const char *cid, double sr, uint32_t maxFrames, char *err, int errcap) { GUARD(nullptr, v3_new_in(lib, cid, sr, maxFrames, err, errcap), nullptr); }
void v3_free(v3_inst *inst) {
	if (!inst) return;
	if (inst->dead) { // it crashed: not touched again (its memory is left)
		std::lock_guard<std::mutex> lock(g_mu);
		g_insts.erase(std::remove(g_insts.begin(), g_insts.end(), inst), g_insts.end());
		return;
	}
	GUARD_VOID(inst, v3_free_in(inst));
}
int v3_dead(v3_inst *inst) { return inst->dead; }
void v3_bus_counts(v3_inst *inst, int *in, int *out) { v3_bus_counts_in(inst, in, out); }
int v3_bus_info(v3_inst *inst, int is_input, int i, int *channels, char *name, int cap) { return v3_bus_info_in(inst, is_input, i, channels, name, cap); }
int v3_param_count(v3_inst *inst) { GUARD(inst, v3_param_count_in(inst), 0); }
int v3_param_info(v3_inst *inst, int i, uint32_t *id, char *title, char *units, int cap, int *steps, double *def, int *flags, int *unit) {
	GUARD(inst, v3_param_info_in(inst, i, id, title, units, cap, steps, def, flags, unit), 0);
}
double v3_param_value(v3_inst *inst, uint32_t id) { GUARD(inst, v3_param_value_in(inst, id), 0.0); }
int v3_param_text(v3_inst *inst, uint32_t id, double value, char *out, int cap) { GUARD(inst, v3_param_text_in(inst, id, value, out, cap), 0); }
uint32_t v3_latency(v3_inst *inst) { GUARD(inst, v3_latency_in(inst), 0u); }
int v3_process(v3_inst *inst, float *in0, float *in1, float *sc0, float *sc1, uint32_t frames) { GUARD(inst, v3_process_in(inst, in0, in1, sc0, sc1, frames), 0); }
int v3_state_save(v3_inst *inst, uint8_t **comp, size_t *complen, uint8_t **ctrl, size_t *ctrllen) { GUARD(inst, v3_state_save_in(inst, comp, complen, ctrl, ctrllen), 0); }
int v3_state_load(v3_inst *inst, const uint8_t *comp, size_t complen, const uint8_t *ctrl, size_t ctrllen) { GUARD(inst, v3_state_load_in(inst, comp, complen, ctrl, ctrllen), 0); }
int v3_unit_name(v3_inst *inst, int unit, char *name, int cap) { GUARD(inst, v3_unit_name_in(inst, unit, name, cap), 0); }
int v3_programs(v3_inst *inst, uint32_t *param, int *steps) { GUARD(inst, v3_programs_in(inst, param, steps), 0); }
int v3_program_name(v3_inst *inst, int index, char *name, int cap) { GUARD(inst, v3_program_name_in(inst, index, name, cap), 0); }
int v3_gui_available(v3_inst *inst) { GUARD(inst, gui_available_in(inst) ? 1 : 0, 0); }
int v3_gui_show(v3_inst *inst) { GUARD(inst, gui_open(inst), 0); }
void v3_gui_hide(v3_inst *inst) { GUARD_VOID(inst, gui_close(inst)); }
int v3_gui_visible(v3_inst *inst) { return inst->guiVisible ? 1 : 0; }
