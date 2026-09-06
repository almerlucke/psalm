Update to v3123.3.0 failed

Now I'm examining the tools directory to understand how the Add, Mul, and Clear functions are implemented, aiming to grasp their roles and interactions before making any changes.

**Overview of `granular.go` Performance Analysis**

An analysis of `modules/granular/granular.go` reveals several performance bottlenecks in memory layout, CPU cache locality, interface dynamic dispatch, slice allocations, and inner-loop bounds checking. In real-time audio synthesis (DSP), operations execute thousands of times per second (e.g. at 44.1 kHz / 48 kHz), where small per-sample and per-grain inefficiencies compound significantly.

Below are key areas and actionable suggestions to make `granular.go` faster and more CPU/memory efficient without altering functional behavior.

---

**1. Replace Doubly-Linked Lists with Contiguous Slices (Cache Locality & Allocation)**

**Current Issue**  
`Granulator` uses `list.List[*grain]` for `freeGrains` and `activeGrains`.
- Each linked list node is a heap-allocated `*Element[*grain]` holding a `*grain`.
- Traversing, unlinking, and pushing elements involves pointer chasing across non-contiguous heap memory, causing CPU L1/L2 cache misses.
- Node manipulations incur unnecessary pointer updates.

**Suggestion**  
Use a flat slice `[]*grain` (or a flat pre-allocated slice `[]grain`) for both free and active grain pools:
- **Activating a grain**: Pop from the end of `freeGrains` in $\mathcal{O}(1)$ (`g := freeGrains[len(freeGrains)-1]; freeGrains = freeGrains[:len(freeGrains)-1]`).
- **Deactivating a finished grain**: Use the swap-and-pop pattern in $\mathcal{O}(1)$ (`activeGrains[i] = activeGrains[len(activeGrains)-1]; activeGrains = activeGrains[:len(activeGrains)-1]`).
- **Iterating active grains**: Iterating over a slice is linear in memory, prefetcher-friendly, and SIMD-friendly.

---

**2. Eliminate Closure & Callback Overhead in Active Grain Traversal**

**Current Issue**  
In `synthesizeActiveGrains`, traversal relies on `gl.activeGrains.ForEachElement(func(e *list.Element[*grain], index int) { ... })`.
- Calling a closure per active grain on every audio block introduces indirect function call overhead and may trigger closure allocations depending on compiler escape analysis.

**Suggestion**  
Use a direct, standard loop over the active grain collection:
```go
for i := 0; i < len(gl.activeGrains); {
    g := gl.activeGrains[i]
    n := g.synthesize(gl.sourceBufs, bufSize)

    // ... mix into output buffers ...

    if g.sampsToGo == 0 {
        // Swap with last active grain and return to free pool
        lastIdx := len(gl.activeGrains) - 1
        gl.activeGrains[i] = gl.activeGrains[lastIdx]
        gl.activeGrains = gl.activeGrains[:lastIdx]
        gl.freeGrains = append(gl.freeGrains, g)
    } else {
        i++
    }
}
```

---

**3. Hoist Repeated Method Calls & Invariant Expressions Outside Loops**

**Current Issue**  
Inside `synthesizeActiveGrains`, `onsetSynthesize`, and `perFrameSynthesize`, `gl.Output(0).Channels` and `gl.Config()` are repeatedly called inside inner loops:
```go
for c, ch := range gl.Output(0).Channels { // Evaluated for every single grain!
    for i := range n {
        ch[outIndex+i] += gl.sourceBufs[c][i]
    }
}
```
- `gl.Output(0)` incurs method dispatch and slice lookups on each grain iteration.
- `gl.Config()` creates a copy of the `psalm.Config` struct.

**Suggestion**  
Cache references to output channels and config before entering grain processing loops:
```go
outChannels := gl.Output(0).Channels
cfg := gl.Config()
```

---

**4. Enable Compiler Bounds-Check Elimination (BCE) and Vectorization**

**Current Issue**  
In the sample accumulation loop:
```go
for c, ch := range gl.Output(0).Channels {
    for i := range n {
        ch[outIndex+i] += gl.sourceBufs[c][i]
    }
}
```
The Go compiler must check array bounds on `ch[outIndex+i]` and `gl.sourceBufs[c][i]` for each sample iteration unless it can prove slices are large enough.

**Suggestion**  
Reslice destination and source slices once before the loop:
```go
for c, ch := range outChannels {
    dst := ch[outIndex : outIndex+n]
    src := gl.sourceBufs[c][:n]
    for i := range dst {
        dst[i] += src[i]
    }
}
```
By slicing `dst` and `src` to length `n`, Go compiler's Bounds Check Elimination (BCE) eliminates runtime boundary checks inside the inner loop, enabling loop unrolling and SIMD auto-vectorization.

---

**5. Prevent Slice Allocations in `ParameterGenerator.Next`**

**Current Issue**  
`ParameterGenerator.Next(int64, psalm.Config) []Parameter` returns a slice `[]Parameter`.
- In `onsetSynthesize()`, `gl.paramGen.Next(timestamp, cfg)[0]` is called for each grain onset. Creating a new slice `[]Parameter` for every single grain trigger introduces heap allocations and Garbage Collector (GC) pressure.
- In `perFrameSynthesize()`, `params := gl.paramGen.Next(timestamp, cfg)` allocates a slice on every single DSP frame.

**Suggestion**
- **For `Onset` mode**: Introduce a method like `NextOnset(int64, psalm.Config) Parameter` that returns a single `Parameter` without allocating a slice.
- **For `PerFrame` mode**: Allow `Next` to accept a reusable scratch slice (e.g. `Next(int64, psalm.Config, []Parameter) []Parameter`) or maintain an internal reusable slice in the generator.

---

**6. Contiguous Pre-allocation of Grain Instances**

**Current Issue**  
In `New(...)`:
```go
for range grainPoolSize {
    gl.freeGrains.Push(&grain{
        source:   sf.New(config),
        envelope: ef.New(config),
    })
}
```
Each `grain` is allocated independently on the heap via `&grain{}`.

**Suggestion**  
Allocate all `grain` structs in a single contiguous backing slice during initialization:
```go
grains := make([]grain, grainPoolSize)
gl.freeGrains = make([]*grain, grainPoolSize)
for i := range grains {
    grains[i].source = sf.New(config)
    grains[i].envelope = ef.New(config)
    gl.freeGrains[i] = &grains[i]
}
```
This guarantees contiguous memory layout for grain metadata and reduces heap fragmentation.

---

**7. Generic Monomorphization to Avoid Interface Dispatch (Optional Architectural Enhancement)**

**Current Issue**  
`Source` and `Envelope` are interfaces (`granular.Source` and `granular.Envelope`). Calling `g.source.Synthesize(...)` and `g.envelope.Synthesize(...)` requires dynamic `iface` method table lookups on every grain per frame, preventing function inlining.

**Suggestion**  
If the architecture permits parameterized types, using Go generics (e.g. `Granulator[S Source, E Envelope]`) allows the Go compiler to inline `Synthesize` methods directly and fuse source generation, envelope calculation, and buffer writing into a single pass.

---

**Summary Table of Proposed Improvements**

| Optimization | Target Area | Benefit |
| :--- | :--- | :--- |
| **Flat Slices vs. Linked Lists** | `freeGrains`, `activeGrains` | Eliminates pointer chasing, improves L1/L2 cache locality, reduces memory footprint |
| **Direct Loop over Grains** | `synthesizeActiveGrains` | Eliminates closure call overhead and function boundary overhead |
| **Hoist Invariant Lookups** | `gl.Output(0).Channels`, `gl.Config()` | Removes redundant method dispatches and struct copying |
| **Slice Bounds Reslicing** | Output buffer mixing loop | Triggers Go BCE (Bounds-Check Elimination) and auto-vectorization |
| **Zero-Allocation `Next` API** | `ParameterGenerator` | Removes per-grain / per-frame GC allocations and latency spikes |
| **Contiguous Grain Pool** | `New` initialization | Reduces memory fragmentation and improves memory locality |

