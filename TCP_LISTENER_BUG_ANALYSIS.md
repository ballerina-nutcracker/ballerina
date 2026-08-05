# Interpreter bugs found while porting `ballerina/tcp`

Found while porting `ballerina/tcp` (client + listener/service stdlib, branch `stdlibs-l3`). Two distinct issues surfaced during smoke-testing. **Bug 1 is fixed and documented already — no action needed.** **Bug 2 is unresolved and is the subject of this handoff**: it's a core-interpreter issue (not specific to tcp's logic), the tcp stdlib port is paused pending a fix.

Everything below was found by hand-editing `.bal` files under `lib/stdlibs/ballerina/tcp/0.0.1/go1.2/` and running `go run ./cli/cmd run <file>.bal` repeatedly — no test framework needed to reproduce.

---

## Bug 1 (fixed, for context only) — included-record init params break cross-module resolution

**Symptom:** `new(...)` on a class whose `init` is shaped `init(<positional params>, *IncludedRecordConfig config)` (an *included-record spread* parameter, i.e. `*Config`, not a plain default-valued `Config config = {}`) fails to resolve with:
```
error[SEMANTIC_ERROR]: failed to determine object type with fitting init function
error[SEMANTIC_ERROR]: expression in 'on' clause is not a listener   // (or similar, depending on context)
```
This only manifests when the *calling* module imports a **second** stdlib package alongside the one declaring the class — a single import compiles fine.

**Repro (minimal, single-module, does NOT reproduce — for contrast):**
```ballerina
import ballerina/io;

type Config record {|
    string x = "a";
|};

class Foo {
    public isolated function init(string a, int b, *Config config) {
    }
}

public function main() returns error? {
    Foo f = new ("hi", 1);   // compiles fine — single module
    io:println("hi");
}
```

**Repro (reproduces — cross-module):** any `.bal` file with `import ballerina/tcp;` + a second import (e.g. `import ballerina/io;` or `import ballerina/random;`), using `tcp:Client c = check new ("host", 80);` (relying on `tcp:Client.init(string, int, *ClientConfiguration)`, the original jBallerina-faithful signature). Fails at the `new(...)` call. `ldap:Client` (whose `init` takes *only* an included record, no leading positional params — `init(*ConnectionConfig config)`) and `http:Client` (whose `init` uses a plain default-valued record, not an included one — `init(string url, ClientConfiguration config = {})`) do **not** trigger this, which is what narrows the trigger to "positional params + trailing included-record param" specifically.

**Fix applied (with user sign-off):** changed `tcp:Client.init`/`tcp:Listener.init` from jBallerina's `*ClientConfiguration`/`*ListenerConfiguration` (included-record spread) to a plain default-valued `ClientConfiguration config = {}` / `ListenerConfiguration config = {}`, matching `http:Client`'s style. This is a real, intentional, documented public-API divergence from jBallerina (call sites use `check new("host", 80, {localHost: "x"})` instead of jBallerina's named-arg-spread `check new("host", 80, localHost = "x")`) — see the doc comments in `tcp.bal`/`listener.bal` and the (to-be-written) README's Notable Behavioural Changes.

**If you want to actually fix this properly instead of relegating it to a workaround**, the likely area is wherever `new ClassName(...)` resolves the "fitting init function" against an included-record parameter — search `semantics/` for `INCLUDED_RECORD` / `IsIncludedRecord`-type handling in call/constructor resolution, and check whether it behaves differently when the target class's owning module is resolved as part of a multi-module (2+ non-builtin-lang imports) compilation versus a single-import one.

---

## Bug 2 (RESOLVED — root cause found and fixed)

> **Resolution (2026-07-07):** The crash was a **module-wide `$default$N` name collision across compilation units**, not anything in dispatch or the registry maps.
>
> Default-parameter value functions are named by a per-module counter (`moduleSymbolResolver.nextDefaultSymbolName` → `$default$0`, `$default$1`, …). But `forCompilationUnit` (`semantics/symbol_resolver.go`) copied `defaultCounter` **by value** into each per-file resolver, so every `.bal` file in a module restarted numbering at the same value. In `ballerina/tcp`, `Client.init`'s `config` default (in `tcp.bal`, takes 2 preceding params: host, port) and `Listener.init`'s `config` default (in `listener.bal`, takes 1: port) both became `ballerina/tcp:$default$0`. The runtime registry (`runtime/internal/modules/registry_impl.go`) keys BIR functions by that string, so one silently overwrote the other — which one won depended on registration order (map iteration in `RegisterModule` over `classDef.VTable` plus desugar append order), hence the per-process-run flakiness. On failing runs, `new Listener(9995)`'s 1-arg call to the config default resolved to Client's 2-param default function, and `initLocalsForFunction` (`runtime/internal/exec/executor.go`) indexed `args[1]` with `len(args)==1`.
>
> The reported crash line was the `check new (9995)` Listener constructor, not `attach` — the original analysis misread the stack line. `http` never hit this because it's a single `.bal` file (one compilation unit → no collision). The map-order hypothesis was half right: map iteration order chose *which* duplicate won, but the duplicate keys were the actual bug. Note this could also silently pick the *wrong default value* (no crash) when colliding defaults have compatible signatures.
>
> **Fix:** `defaultCounter` is now a `*int` shared by all compilation-unit resolvers of a module (created in `newCompilationUnitsSymbolResolver`, shared via `forCompilationUnit`), making `$default$N` unique module-wide. Regression test: `corpus/project/multi-file-defaults-v/` (two files with defaultable-param functions; fails 5/5 pre-fix, passes post-fix). tcp repro: 0/30 failures post-fix (was ~1-in-5). Full test suite: no new failures.

The original (now-historical) investigation notes follow.

### Symptom

Calling a **plain (non-remote) external instance method with a trailing defaulted parameter**, specifically `tcp:Listener.attach(...)`, intermittently crashes with:
```
error: runtime error: index out of range [1] with length 1
        at main(<file>.bal:<line of the .attach(...) call>)
```
Failure rate empirically ~1-in-5 to 1-in-6 runs of the *identical* `.bal` file and binary (`go run ./cli/cmd run` recompiles from source each time via the embedded stdlib FS, but the failure is not tied to compilation — see below).

### Exact repro

This is the **current, already-fixed-for-Bug-1** shape of `tcp.bal`/`listener.bal` on `stdlibs-l3` — i.e. reproduce this against the tree as it stands right now (Bug 1's fix is already committed to the working tree, `Client`/`Listener` both use default-valued config params).

```ballerina
import ballerina/tcp;
import ballerina/io;

service class MyService {
    *tcp:Service;
    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        io:println("connected: ", caller.remoteHost);
        return new EchoService();
    }
}

service class EchoService {
    *tcp:ConnectionService;
    remote function onBytes(tcp:Caller caller, readonly & byte[] data) returns tcp:Error? {
        io:println("onBytes: ", data.length());
        check caller->writeBytes(data);
    }
    remote function onError(readonly & tcp:Error err) returns tcp:Error? {
        io:println("onError: ", err.message());
    }
    remote function onClose() returns tcp:Error? {
        io:println("onClose");
    }
}

public function main() returns error? {
    tcp:Listener lis = check new (9995);
    MyService svc = new MyService();
    check lis.attach(svc);          // <-- crash happens here, ~1-in-5 runs
    check lis.start();

    tcp:Client c = check new ("localhost", 9995, {});
    check c->writeBytes("hello".toBytes());
    readonly & byte[] r = check c->readBytes();
    io:println("client got: ", 'string:fromBytes(r));
    check c->close();
    check lis.gracefulStop();
}
```

Run it repeatedly to observe the flakiness:
```shell
for i in 1 2 3 4 5 6 7 8; do go run ./cli/cmd run /path/to/repro.bal 2>&1 | tail -3; done
```
On success, output is:
```
connected: ::1
onBytes: 5
client got: hello
onClose
```
On failure:
```
error: runtime error: index out of range [1] with length 1
        at main(repro.bal:27)
```
(line number will shift depending on exact file, it always points at the `.attach(...)` call site).

### What's already been ruled out

| Hypothesis | Test | Result |
|---|---|---|
| Related to the omitted trailing default arg (`attach`'s `name` param defaults to `()`) | Called `lis.attach(svc, ())` supplying `name` explicitly | **Still flaky** — not it. |
| Related to inline object construction as a call argument | Pre-constructed `MyService svc = new MyService();` then `lis.attach(svc)` as a separate statement | **Still flaky** — not it. |
| A goroutine/data race in my own tcp native code (accept loop, per-connection dispatch) | Crash happens on `lis.attach(...)`, which runs **before** `lis.start()` is ever called — my code's `go acceptLoop(...)` goroutine doesn't exist yet at that point | **Ruled out** — no concurrent goroutine of mine is running yet. |
| My own `Listener.attach` extern Go handler has a bug (e.g. reads `args[1]` when `len(args)==1`) | Added a `fmt.Printf("attach called, len(args)=%d", len(args))` as the very first line of the handler, then ran the repro repeatedly | On **every successful** run the print fired with `len(args)=3`. On **every failing** run, **the print never fired at all** — my handler is never invoked. The panic happens somewhere in the interpreter's own dispatch/argument-marshaling *before* reaching stdlib native code. |
| General to *any* `Listener.attach()` (i.e. not tcp-specific) | Wrote an equivalent `http:Listener` program (`new`, `attach(svc)`, `start()`, then a real request) with the exact same shape and ran it 8/8 times | **0/8 failures.** `http:Listener.attach` has an identical Ballerina-level signature (`attach(Service tcpService, string[]|string? name = ())`) and is not flaky. |
| A classic goroutine data race, sensitive to scheduling | Ran the Bug-1 repro (a different, but similarly-shaped resolution failure) under `GOMAXPROCS=1` | **Still failed deterministically.** This doesn't disprove a race in Bug 2 specifically (not retested under GOMAXPROCS=1), but is a data point suggesting these two failures may not be classic thread-scheduling races. |

### Leading hypothesis (unconfirmed — needs verification, not yet proven)

Native module registration (`runtime.RegisterExternClassDef`/`RegisterExternFunction`, called from each stdlib's `init()` → `RegisterModuleInitializer` callback) is **fully sequential** — see `runtime/runtime.go` around line 110:
```go
for _, init := range moduleInitializers {
    init(rt)
}
```
No goroutines here, and this completes entirely before any BIR package `Init`/`RunEntrypoints` (i.e. before any Ballerina `main()`/module-init code executes). This weighs *against* a plain "registration races with execution" data race.

**What's genuinely different between `tcp` and `http`:** `tcp` registers **three** native classes (`Client`, `Listener`, `Caller` — see `lib/stdlibs/ballerina/tcp/0.0.1/go1.2/native/{client,listener,dispatch}.go`, each calling `runtime.RegisterExternClassDef`), while `http` registers **two** (`Client`, `Listener`). Both `bir.BIRClassDef.VTable` (`map[string]*bir.BIRFunction`) and the registry's own backing maps (`runtime/internal/modules/registry_impl.go`: `birFunctions`, `birClassDefs`, `nativeFunctions` — **none of these have any mutex**, see the `Registry` struct) are plain Go maps. Go **intentionally randomizes map iteration order per process run** as a language guarantee specifically to surface bugs that wrongly depend on iteration order — this fits the observed symptom much better than a data race would:
- It explains **per-process-run** flakiness (not per-call-within-a-run flakiness — I never saw it fail then succeed within the same `go run` invocation, only across separate invocations, which is exactly what map-seed-per-process would predict).
- It's consistent with `GOMAXPROCS=1` not fixing Bug 1 (Go's map iteration randomization is unrelated to thread scheduling/GOMAXPROCS).
- It would explain why `tcp` (3 registered classes → more/different map entries and iteration order surface area) hits it while `http` (2 classes) apparently doesn't, **without** needing an actual race — just an existing order-dependency bug that's more likely to have an observable effect with a different map shape/content.

**This is a hypothesis, not a confirmed root cause.** It has not been proven that any code actually iterates a map to build an argument list or resolve a default parameter for `attach`-shaped calls. The crash's exact origin (which function, which map) has not been located — my own extern handler is confirmed not to be it (see the ruled-out table above), so the bug is somewhere between "user code calls `lis.attach(svc)`" and "the registered Go extern function gets invoked," i.e. in `runtime/internal/exec/` (method-call resolution/dispatch for a **regular**, non-remote instance method — `dispatchMethodCall`/`execCall` in `terminators.go`, and whatever builds the `args []values.BalValue` slice for that call, including default-parameter substitution for the omitted trailing arg).

### Suggested next steps for the investigating session

1. **Confirm or kill the map-order hypothesis first** — it's cheap to test: run the repro in a loop with `GODEBUG=` map-related env vars, or add temporary instrumentation that dumps `len(classDef.VTable)` / iteration order at the point args are built, and see if it correlates with pass/fail. Alternatively, temporarily replace any suspect `map[string]...` iteration in the args-building path with a sorted-keys iteration and see if the flakiness disappears.
2. **Narrow "3 classes vs 2 classes" further** — in `lib/stdlibs/ballerina/tcp/0.0.1/go1.2/native/listener.go`, the `registerCallerClassDef(rt)` call (around line 72) registers the third class (`Caller`). Temporarily comment it out (accepting that other things will then fail to compile/run — this is just to isolate the count effect) and see if `attach()` becomes stable across many repeated runs. This wasn't done yet due to code-quality/revert risk within this session; it's a cheap, high-value first experiment for the new session.
3. **Trace `dispatchMethodCall`/`execCall`** (`runtime/internal/exec/terminators.go`, `non_terminators.go`) for the specific path taken by a plain instance-method call (`lis.attach(svc)`) with an omitted trailing default parameter, and compare it against the path for a `->` remote-method call with an omitted trailing default (e.g. `ldap:Client.getEntry`, which is known-reliable across many corpus test runs) — the difference between those two paths is a strong candidate location.
4. Once a root cause is found, the fix likely belongs in `runtime/internal/exec/` or `runtime/internal/modules/registry_impl.go` (if it does turn out to need a mutex or deterministic ordering, e.g. sorting map keys before use) — core interpreter code shared by every stdlib, so changes here should be re-verified against the full `go test ./corpus/...` suite (all existing stdlibs), not just tcp.

### Current state of the tcp port

- `lib/stdlibs/ballerina/tcp/0.0.1/go1.2/` exists with the full `.bal` public API (`Client`, `Listener`, `Caller`, `Service`/`ConnectionService`, secure-socket types, error type) and a full native Go implementation (`native/{tcp,client,listener,dispatch,tls}.go`) — compiles cleanly (`go build ./...`, `go vet ./...` both clean).
- Bug 1's fix is applied and the manual-attach round trip **does work correctly** when it doesn't hit Bug 2 (verified full echo round trip: connect → write → onBytes → echo → read → close → exactly-one onClose).
- **Paused before**: writing corpus tests (`corpus/extern/tcp_client_test.go` + fixtures), the README, and the aggregator update — all pending until Bug 2 is understood, since building tests/docs on top of a listener that intermittently crashes isn't productive.
- Wire-up (`lib/rt/libs.go`, `test_util/testphases/phases.go`) is already done, so the package is fully live in the interpreter — safe to keep exercising it for the Bug 2 investigation without any additional setup.
