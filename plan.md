# `bal test` (POC) — implementation plan

## Context

`spec.md` (authoritative, do not redesign) specifies a `bal test` command for the Go Ballerina
implementation. Today there is no way to run test functions: `bal run` and `bal build` are the only
execution paths, the only compiler plugins are those declared by a package manifest and statically
linked through the generated `compilerpluginregistry`, and there is no `ballerina/test` stdlib.

The feature needs four things that do not exist yet:

1. A **host-injected compiler plugin** — the CLI must supply a plugin for one compilation, activated
   by a module importing `ballerina/test`, so it can see `*ast.BLangPackage` and discover
   `@test:Config` functions. `*ast.BLangPackage` is reachable only from a `PackageTransformer`
   (`moduleContext.bLangPkg` is private and not exposed on `projects.Package`).
2. A minimal **`ballerina/test` stdlib** (`TestConfig`, `Config`, `assertFail`).
3. A **PAL `Testing.Fail` hook** so `assertFail` can signal failure to the CLI, which owns the PAL.
4. A **test runner** in the CLI that drives `runtime.LookupFunction`/`InvokeFunction` directly,
   after `Init` and `Listen`, with per-test stdout capture and outcome reporting.

Outcome: `bal test` runs every enabled, matching `@test:Config` function and reports pass/fail/error/skip;
`bal run` and `bal build` on the same package behave exactly as before (the annotation type-checks and is
a no-op).

## Decisions filling gaps the spec leaves open

- **`--tests` / `--list` module form: bare submodule name.** `ModuleKey{Org, Module}` keeps the full
  dotted runtime identity (`foo.sub`), which is what `runtime.LookupFunction` needs. The *display and
  filter* name is the text after the first `.`, or the whole string when there is none — so package
  `foo`'s default module shows as `foo` and its submodule `sub` as `sub`. Add
  `func (k ModuleKey) displayName() string` and use it in both `Filter.Matches` and the reporters.
  Root-package module names are always `<pkgName>` or `<pkgName>.<rest>`, so this needs no package name.
- **`Result.Duration` is not printed.** The spec's Reporter section fully specifies both output
  formats and neither includes a duration; printing one would make every txtar golden flaky. Keep the
  field (the spec declares it), leave it out of the output.
- **Unit tests live beside the code, not in the corpus.** Three scenarios the corpus cannot reach get
  ordinary Go unit tests. The spec's `bal test` fixtures still go through
  `corpus/cli_integration_test.go`, which drives the real `bal` binary with arbitrary args (it already
  covers `bal build` and `bal pack`) — that is not the `corpus/bal` per-stage corpus.

## Verified facts the implementation depends on

- `att.Symbol()` **is** populated after semantics: `semantics/internal/symbols/symbol_resolver.go:1460`
  calls `target.SetSymbol(symRef)`. `GetAnnotationAttachments()` returns the slice header
  (`ast/ast.go:935`), so the type resolver's in-place mutations are visible to the plugin. Match
  `Config` via `compilerCtx.SymbolPackage(att.Symbol())` (`{Organization:"ballerina", Package:"test"}`,
  ignore `Version`) plus `SymbolName(att.Symbol()) == "Config"`. Guard with `ast.SymbolIsSet`.
- `AnnotationValue` is populated **only when the whole attachment is constant**
  (`semantics/internal/types/type_resolver.go:1475-1490`) — a `before`/`after` function reference makes
  it nil. This is why the spec needs both the AST path and the `*values.Map` path.
- `att.Expr` is **not** desugared before `AfterSemantics` (plugins are phase 3, desugar is phase 4,
  `projects/package_compilation.go:154-167`) and the runtime-value rewrite never fires for function
  annotations (sink is nil, `type_resolver.go:1273`). So
  `att.Expr.(*ast.BLangMappingConstructorExpr)` → `*ast.BLangMappingKeyValueField` is safe.
- `Config`'s record type **passes** the annotation-type check: a `function` value is inherently
  immutable (`btFunction < btFuture`, `semtypes/basic_type_code.go:92`), hence `Cloneable`, hence the
  closed record is a subtype of `map<Cloneable>`. Worth a comment in `test.bal` — it is non-obvious.
- `ballerina/test` **will** be in `resolver.providers` and in `Environment.publicSymbols`:
  `newCompilerPluginResolver` is fed `topologicallySortedModuleList`, which includes dependency
  modules, and phase 1 publishes symbols for every source-loaded module
  (`projects/module_context.go:299`). So the hard `InternalError` at `package_compilation.go:226` is
  unreachable.
- `runtime.LookupFunction` keys on `org + "/" + module + ":" + name`
  (`runtime/internal/exec/dispatch.go:48`) where `module` is `PackageID.PkgName` = the full dotted
  module name (`projects/module_context.go:516`), with no visibility check and no dead-code
  elimination. So `ModuleKey` from `pkg.PackageID` matches exactly, and non-`public` test functions
  are reachable.
- `import ballerina/test;` used only in `@test:Config` does **not** trigger "unused import prefix"
  (`symbol_resolver.go:309-314` marks the prefix used on the annotation path).
- A Ballerina `panic` leaves `InvokeFunction` as a **Go panic**; the Go `error` return is always nil
  for BIR functions (`runtime/internal/exec/handle.go:44-52`). A test that *returns* an error yields it
  as the `values.BalValue` result — type-assert to `*values.Error`.
- `returns never` + `panic <call-expr>` is accepted (`corpus/bal/subset9/09-function/never-3-v.bal`).
- `isTerminal()` already exists at `cli/cmd/utils.go:72` — reuse it, at command level, not inside the
  reporter.

## Stages

Each stage builds (`go build ./...` per module; `go.work` already covers every module) and has a test
that passes at that stage. Stages 3 and 5 are the transient states the spec permits.

### Stage 1 — `feat(pal): add Testing group`
`platform/pal/platform.go`: add `Testing Testing` to `Platform` and
`type Testing struct{ Fail func(message string) }` in the same `type (...)` block. `palnative.NewPlatform`
leaves it zero. Purely additive; proof is compilation of `platform`, `runtime`, `lib`, `cli`.

### Stage 2 — `fix(runtime): release held locks when an invoked function panics`
`runtime/runtime.go:40-43`: add a deferred `recover` to `InvokeFunction` that calls
`cx.ReleaseAllHeldLocks()` and re-panics with the same value. `cx` is `*extern.Context`
(`runtime/internal/exec/exec.go:28`), so `ReleaseAllHeldLocks` (`runtime/extern/extern.go:181`) is in
scope. Mirrors `exec.RunEntrypoints` (`runtime/internal/exec/interpreter.go:33-38`) and
`exec.startFuture`. The three existing callers (`corpus/integration_test.go:373`,
`runtime/lifecycle_test.go:374`, `test_util/testharness/test_harness.go:549`) do not rely on locks
staying held.

**Test:** `runtime/lifecycle_test.go` — a `lock`-holding function that panics, then a second function
entering the same `lock`; assert it completes under a `time.After` guard. The file already has
`newLifecycleTestRuntime` (line 377), `invokeAndRecover` (line 370) and that guard idiom (line 365).

### Stage 3 — `feat(compilerplugin): add injected plugin types`
`compilerplugin/compilerplugin.go`: add `Provider{Org, Package}` and
`InjectedPlugin{Provider, Plugin}`. Data only; no consumer yet.

### Stage 4 — `feat(projects): apply host-injected compiler plugins`
- `projects/env.go` — private `injectedPlugins` field + `injectedCompilerPlugins()` accessor.
- `projects/project_environment_builder.go` — field + `WithCompilerPlugins`, copied in `Build()`.
- `projects/project_loader.go` — `CompilerPlugins` on `ProjectLoadConfig`; thread it through
  `createEnvironmentWithRepositories` and `createWorkspaceEnvironment`. `loadBalaProjectWithEnv`
  inherits via the shared environment, as the spec states.
- `projects/compiler_plugin_registry.go` — `rootPackage PackageDescriptor` and
  `injected []compilerplugin.InjectedPlugin` on the resolver; new `newCompilerPluginResolver`
  signature; `injected bool` on `resolvedCompilerPlugin` (error messages only); `pluginsFor` appends
  injected plugins **after** manifest plugins, only when the module's package descriptor equals
  `rootPackage` and `module.explicitImports` contains the provider. `declaration.function` is
  `"<injected>"`; `position` is the matching import — `runCompilerPlugins` uses only
  `resolved.position.position` as a diagnostic location, so no change is needed there.
- `projects/package_compilation.go:155` — pass `c.rootPackageContext.getDescriptor()` and
  `...Environment().injectedCompilerPlugins()`.

**Test:** `projects/compiler_plugin_registry_test.go` — three cases (root-package module importing the
provider → applied; dependency-package module → not applied; injected runs after manifest). The
existing tests already build `&compilerPluginResolver{...}` and `&moduleContext{...}` literals directly
(lines 32-120), so these drop into that pattern. Plus `go test ./projects/...` for regressions.

### Stage 5 — `feat(lib): add ballerina/test stdlib`
New `lib/stdlibs/ballerina/test/0.0.1/go1.27/`: `Ballerina.toml`, `Bala.toml`, `Dependencies.toml`,
`README.md`, `test.bal`, `native/test.go` — copy the shapes from
`lib/stdlibs/ballerina/random/0.0.1/go1.27/*` (`Bala.toml` needs `schema_version = "4"`,
`platform = "go1.27"`). No `CompilerPlugin.toml`, so plugin-gen and the manifest path ignore it.
`lib/stdlibs/embed.go` needs no change (`//go:embed all:ballerina`). Add the blank import to
`lib/rt/libs.go` (alphabetically, after `random`).

`native/test.go` follows the `random` template: `init()` → `runtime.RegisterModuleInitializer`,
`RegisterExternFunction(rt, "ballerina", "test", "externFail", externFail)`; `externFail` calls
`ctx.Env.Platform.Testing.Fail` when non-nil and returns `values.NewErrorWithMessage(msg), nil`.

**Test:** a corpus lib fixture under `corpus/lib/subset<N>/` (per `AGENTS.md`) — `trap test:assertFail("boom")`
printing the message, plus a `-p` case for the bare default-arg call. This proves `returns never` +
`panic externFail(msg)` + the defaultable param compile and run, and that a nil `Testing.Fail`
degrades to a plain panic, all before any CLI code exists. Golden via `go test ./corpus -update`.

### Stage 6 — `feat(cli): add testerina plan, collector and schedule`
New `cli/internal/testerina/{plan.go,collector.go,schedule.go}` exactly as specified.
`collector.collect` must hold `c.mu` — `runCompilerPlugins` runs one goroutine per module
(`package_compilation.go:156` via `runModulePhase`). Source-order sorting is by
`(Position.FileIndex(), Position.StartOffset())`; those accessors have pointer receivers
(`tools/diagnostics/location.go:69-79`), so keep the `slices.SortFunc` comparator over
`TestFunction` values.

`cli/go.mod`: promote `ast`, `model`, `values` to direct requires and add `compilerplugin`
(`context` is already direct). `go.work` already has the `replace` directives, so no `go.sum` change.

**Test:** `cli/internal/testerina/schedule_test.go` for `ParseFilter`/`Matches` only — wildcard
placement (leading, trailing, interior, bare `*`, `*:*`), `module:name` vs bare `name`, empty-entry
rejection, and the bare-submodule display-name rule. Pure string logic, no compiler dependency.

### Stage 7 — `feat(cli): add bal test command`
New `cli/internal/testerina/{runner.go,reporter.go}` and `cli/cmd/test.go`. Follow `build.go`'s style
(`type testOptions struct`, `var testCmd = createTestCmd()`, `cmd.OutOrStdout()`/`cmd.ErrOrStderr()`)
rather than `run.go`'s package-level var and raw `os.Stdout`. Register in `cli/cmd/bal.go`.

Two constraints that are not optional:

- **PAL install order.** `extern.InitEnv` copies `pal.Platform` **by value**
  (`runtime/extern/extern.go:55-64`), so every override must be in place before `NewRuntime`:
  `NewRunner(platform, osSignals, tyEnv)` → `runtime.NewRuntime(runner.Platform(), tyEnv)` →
  `runner.Start(rt)` → `rt.Init` per package → `rt.Listen()` → `runner.Run(schedule, reporter)` →
  `runner.Shutdown(rt)`. The signal channel must be non-nil at the first `Init` —
  `setupSignalListeners` panics on nil (`runtime/lifecycle.go:293`).
- **`Shutdown` must not signal an already-stopped runtime.** With no listeners, `Listen()` runs
  `GracefulStopping → Stopped`, and `stoppedAction` writes and **closes** `exitCodeChan`
  (`runtime/lifecycle.go:272-284`). The signal watcher checks `state == StateStopped` *before* its
  receive (`lifecycle.go:288-305`), so a later `GracefulStop` send wakes it and it calls
  `go rt.transition(StateGracefulStopping)` — and `transitionTable[StateStopped][StateGracefulStopping]`
  is nil, so `transition` **panics on a bare goroutine and kills the process** after the summary.
  This would hit most fixtures. Guard as the spec's own wording implies ("if no listeners existed the
  channel already holds `0`"): non-blocking read of `rt.ExitStatus` first and return if it yields;
  otherwise non-blocking `GracefulStop` send, then `<-rt.ExitStatus`. Then close the runner's signal
  channel (`lifecycle.go:73` requires the sender to close) and stop the OS-signal forwarder.
  Running tests after the runtime reaches `Stopped` is fine — `LookupFunction`/`InvokeFunction` consult
  only the module registry and never check lifecycle state.

Also: `runner.fail`, the `IO.Stdout` closure and `current` are called from arbitrary strands (workers,
`$start` hooks, HTTP resource goroutines), so all of that state shares one mutex. `invoke` detects
`Errored` three ways — a recovered panic (`*values.Error` → `.Message`, else `fmt.Sprint(v)`), a
returned `*values.Error` result, and a non-nil Go `error`.

`corpus/cli/output/help/help.txtar` will change: registering a non-hidden `test` alters `bal --help`
and breaks `TestBalHelp` (`corpus/cli_integration_test.go:62`). Regenerate deliberately — the golden is
already stale, so expect the diff to also add `build`, `pack`, `push`.

**Test:** `cli/internal/testerina/reporter_test.go` for `ttyReporter` (feed a fixed `[]Result` to
`NewTTYReporter(&bytes.Buffer{})` and assert the `\r`-rewritten line) — the corpus harness runs over
pipes and can never reach it. Plus one smoke corpus fixture and `TestBalHelp`.

### Stage 8 — `test(corpus): bal test scenarios`
Fixtures under `corpus/cli/testdata/test/<case>/`, goldens under `corpus/cli/output/test/*.txtar`, and a
table-driven `TestBalTestScenarios` in `corpus/cli_integration_test.go` modelled on
`TestBalBuildScenarios` (line 567). Cover every scenario in the spec's Tests section.

**This stage needs a new exact-match assertion helper.** All three existing helpers
(`assertBalCommandMatchesTxtarFragmentsLoose` at line 471, `...WithEnv` at 518,
`...ForBinary` at 3201) match stdout by **substring**, which cannot express the spec's negative
assertions: "non-matching tests absent from output", "`io:println` in a passing test is not printed",
"`before` panic → body not run", "`--list` runs nothing". Factor the exact comparison already written
inline in `TestBalBuildWorkspace` (lines 740-757: `test_util.LoadTxtarStdoutStderrExitcode` +
equality + `test_util.FormatExpectedGot`) into a local `assertBalCommandMatchesTxtarExact`.

The `http:Listener` fixture: model it on `corpus/extern/testdata/http-service/http-svc-basic-v.bal:33-40`
(a service on `new http:Listener(port)` plus a client call), renaming its driver to a `@test:Config`
function. `ballerina/http` is a bundled stdlib, not a bala with native Go sources, so
`execWithNativeRunner` finds nothing and does not re-exec. Give it a port not used by
`corpus/extern/testdata/http-service/*` and keep it out of the parallel table. This is the fixture that
proves the `Shutdown` path, since `Listen()` genuinely parks in `StateListening`.

## Verification

- `make build && make vet` — every module compiles.
- `go test ./runtime/... ./projects/... ./cli/...` — the unit gates from stages 2, 4, 6, 7.
- `go test ./corpus -run 'TestLibIntegration'` — stage 5's stdlib fixture.
- `go test ./corpus -run 'TestBalTestScenarios|TestBalHelp|TestBalRunCorpus|TestBalBuildScenarios'` —
  the `bal test` acceptance suite, the regenerated help golden, and proof that `bal run`/`bal build`
  still behave as before.
- Manual end-to-end: `go run ./cli/cmd test <fixture>` on the basic fixture (2 passing), the
  `assertFail` fixture (exit 1, message in summary), `--list`, `--tests`, and the listener fixture
  (must exit, not hang).
- `make test` before finishing — the full native suite.
