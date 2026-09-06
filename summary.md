# PR #875 Linux benchmark investigation

## Issue

PR [#875](https://github.com/ballerina-nutcracker/ballerina/pull/875) replaces the CLI's project-based compilation path with the new driver pipeline. The intended result is lower user-visible elapsed time, particularly for projects with enough independent compilation work to run concurrently.

On Apple Silicon, the benchmark results consistently improve. The Linux CI report is mixed: the large-project cases improve substantially, while several execution-heavy cases become slower. The most pronounced reported regressions are in `closure1-v.bal`, `closure2-v.bal`, and `sieve-v.bal` even though the PR does not change the interpreter runtime.

The benchmark measures complete `bal run` wall time, including compilation and execution. That is the relevant metric for how long a user waits.

## Original CI observations

Using non-overlapping `mean ± 1 standard deviation` as the requested filtering heuristic, the original Linux CI report had the following notable results.

### HEAD faster

- `07-bench/3-v.bal`
- `07-bench/4-v.bal`
- `07-bench/5-v.bal`
- `huge-dependency-project-v`
- `huge-project-v`

The large projects improved by approximately 1.56–1.63 times in CI. This improvement was reproduced locally: `huge-project-v` improved from 31.568 ± 0.203 seconds to 16.214 ± 0.327 seconds, or approximately 1.95 times faster.

### Base faster

- `06-bench/3-v.bal`
- `07-bench/1-v.bal`
- `07-bench/2-v.bal`
- `closure1-v.bal`
- `closure2-v.bal`
- `intmergesort-v.bal`
- `mergesort-v.bal`
- `sieve-v.bal`

The most suspicious cases spend almost all their time executing already-generated BIR rather than compiling it. For example, CI reported `closure2-v.bal` as approximately 49% slower on HEAD.

## Reproduction

The exact revisions from the original PR report were used:

- Base: `bb5298bbaf7cc6ccda8645e3129b01b4feed3f5a`
- HEAD: `719d6c5484269b8601c82550362e0fc6b89243e2`

The CI run used Go 1.26.7 on Linux/amd64. Matching that toolchain was important. With Go 1.26.7, the original `closure2-v.bal` regression reproduced locally:

| Revision | Wall time |
|---|---:|
| Base | 24.626 ± 0.825 s |
| HEAD | 31.325 ± 1.562 s |

HEAD was approximately 27% slower locally. Reversing benchmark order produced essentially the same result:

| Revision | Wall time |
|---|---:|
| HEAD, run first | 30.218 ± 0.819 s |
| Base, run second | 24.157 ± 0.373 s |

This established that the Linux regression is reproducible with the CI toolchain and is not primarily a base-before-HEAD ordering artifact.

## Isolation experiments

### Runtime source changes

There are no changes under `runtime/` between the original Base and HEAD revisions.

**Conclusion:** a direct interpreter source change is eliminated.

### Generated BIR and source locations

The Base and HEAD BIR pretty-print output for the closure reproducer is identical. A separate traversal hashed every instruction and terminator source location, including file path, start/end line, and start/end column.

Both builds produced:

- 79 locations
- hash `a53179c87dcc5621781daeb37abad0b9afc92bb33959a1d2690d40e179333b73`

**Conclusion:** different BIR operations or different source-location values are eliminated.

### Imported dependencies

A reduced closure benchmark removed the `ballerina/io` import and did not print its result. The regression remained:

| Build | Wall time |
|---|---:|
| Base | 1.843 s |
| HEAD code with old CLI compilation path | 1.833 s |
| HEAD with new driver CLI path | 2.283 s |

**Conclusion:** dependency resolution, native `io` initialization, and output are eliminated.

### Non-CLI changes in the PR

HEAD was rebuilt with its CLI compilation path restored to the implementation from the first PR commit while retaining the other changes from the second commit.

| Build | Wall time |
|---|---:|
| Base | 1.861 ± 0.029 s |
| HEAD with old CLI path | 1.842 ± 0.013 s |
| HEAD with driver CLI path | 2.289 ± 0.020 s |

**Conclusion:** the slowdown is triggered by linking and using the new driver CLI path, rather than by the PR's AST, semantic-analysis, or BIR API changes in isolation.

### Retained compiler state

The driver context, resolver, BIR references, compilation result, and runtime reference were cleared where possible before execution, followed by a forced garbage collection.

| Build | Wall time |
|---|---:|
| Normal HEAD | 2.460 s |
| HEAD after clearing compilation references | 2.484 s |

**Conclusion:** retention of the new driver/compiler object graph is eliminated as the primary cause.

### BIR heap allocation layout

HEAD's generated BIR was serialized and immediately deserialized before execution. This reconstructs the BIR object graph and changes its heap allocation layout.

| Build | Wall time |
|---|---:|
| Normal HEAD | 2.306 s |
| HEAD with reconstructed BIR | 2.329 s |

**Conclusion:** the original BIR object's heap allocation order is eliminated as the primary cause.

### Garbage collection frequency

For the original closure workload with `GOMAXPROCS=2`, Base performed 2,650 garbage collections and HEAD performed 2,594. HEAD did not perform additional collections.

**Conclusion:** a regression caused by more frequent garbage collection is eliminated.

## CPU-profile observation

CPU profiles localized the additional execution time to the source-location update in the interpreter's basic-block loop:

```go
for _, inst := range bb.Instructions {
    getCallStack(ctx).SetCurrentLocation(inst.GetPos())
    currentFrame = execInstruction(ctx, inst, currentFrame)
}
```

The relevant profile samples were:

| Operation | Base | HEAD |
|---|---:|---:|
| `executeBasicBlock` flat time | 8.22 s | 16.24 s |
| `SetCurrentLocation(inst.GetPos())` line | 6.26 s | 12.72 s |
| `execInstruction` flat time | 1.52 s | 1.55 s |

The actual instruction dispatch work is effectively unchanged. Almost all extra CPU time is attributed to the inlined source-location update.

## Inlining experiment

`callStack.SetCurrentLocation` is small enough that Go inlines it into `executeBasicBlock`. The new driver dependency changes the final linked executable and moves the unchanged interpreter functions. For example, in the original builds:

- Base `executeBasicBlock`: `0x81c500`
- HEAD `executeBasicBlock`: `0x81ce60`

The normalized machine instructions for the function are identical, but the inlined hot loop is placed differently in the Linux executable.

The following experimental change was applied to both Base and HEAD:

```go
//go:noinline
func (cs *callStack) SetCurrentLocation(location bir.Location) {
    if len(cs.elements) == 0 {
        return
    }
    cs.elements[len(cs.elements)-1].location = location
}
```

The complete original `closure2-v.bal` result became:

| Revision | Wall time |
|---|---:|
| Base with `noinline` | 22.839 ± 0.126 s |
| HEAD with `noinline` | 22.853 ± 0.112 s |

The measurable regression disappeared. A reduced reproducer produced the same result:

| Revision | Wall time |
|---|---:|
| Base with `noinline` | 2.285 ± 0.023 s |
| HEAD with `noinline` | 2.272 ± 0.028 s |

Removing the location update entirely also made Base and HEAD equal, while replacing the dynamic location with an empty constant made them equal without removing the setter call.

## Current conclusion

The experiments establish the following causal boundary:

1. The new driver CLI path triggers the regression.
2. It does not do additional interpreter work and does not produce different BIR or source-location values.
3. The extra time is concentrated in the inlined location update inside `executeBasicBlock`.
4. Preventing that one method from being inlined removes the Base-versus-HEAD difference.

The precise low-level CPU mechanism has not yet been measured with hardware counters. The current evidence is consistent with the new CLI dependency graph changing the placement of an unusually large and frequently executed inlined location-copy sequence, making that hot loop perform poorly on the Linux/amd64 runner. This is a specific consequence of this PR's linked CLI composition, not a claim that arbitrary Go or hardware noise explains the result.

## Experimental PR

PR [#881](https://github.com/ballerina-nutcracker/ballerina/pull/881) adds only the `//go:noinline` directive. It is intentionally non-draft and marked **Don't Merge** so the normal benchmark workflows run.

The expected validation is:

- The runtime-heavy regressions, especially `closure1-v.bal`, `closure2-v.bal`, and `sieve-v.bal`, should disappear or become statistically insignificant.
- The large-project compilation improvements should remain.
- Memory and HTTP behavior should remain materially unchanged.

If CI confirms those expectations, it validates the isolated inlining interaction across the same Linux environment that produced the original report. The directive should still be treated as an experimental mitigation until the complete benchmark output is reviewed.
