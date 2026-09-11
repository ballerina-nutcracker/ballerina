# `bal test` command (POC)

Specify:

- Goals
  - Add a `bal test` command that compiles the current package with an additional, CLI-injected compiler
    plugin, discovers functions annotated with `@test:Config`, and executes them against a live runtime.
  - Provide a minimal `ballerina/test` standard library: `TestConfig`, the `Config` annotation, and
    `assertFail`.
  - `assertFail` signals failure to the host through PAL; the CLI installs the callback.
  - Support `enable`, `before`, `after`; filtering with `--tests`; progress output; failures-only summary;
    exit code 1 if any test fails or errors.
  - Keep test data (`TestPlan`), execution order (`Schedule`), and execution (`Runner`) as separate pieces.
- Non-goals
  - `tests/` directory support, a separate test module, or any visibility changes. Tests live in ordinary
    source files and follow existing visibility rules.
  - Mocking, `groups`, `dependsOn`, data providers, any assert other than `assertFail`, test reports, code
    coverage, running tests of dependency packages.
  - Skipping `main` in `bal test`. `main` runs exactly as in `bal run`.
  - Stack traces for test failures. Output is the panic message only.
  - A bubbletea TUI. The progress line is plain terminal output.
- Success criteria
  - `bal test` on a package with `@test:Config` functions runs every enabled, matching test and prints a
    progress line (TTY) or one line per test (non-TTY), then a summary that lists only failed/errored tests.
  - `assertFail` marks the running test failed, whether called directly, under `trap`, in a worker, or in a
    resource invoked by the test.
  - `bal run` and `bal build` on the same package behave as before: the annotation type-checks and is
    otherwise a no-op, `main` runs, test functions are never invoked.
  - A test that panics inside a `lock` block does not deadlock later tests.
  - All scenarios in the Tests section pass as corpus tests.

# Design

## Components

- `ballerina/test` stdlib (`lib/stdlibs/ballerina/test/0.0.1/go1.27/`): Ballerina sources plus one native
  function. No `CompilerPlugin.toml`, so plugin-gen and the manifest-driven plugin path ignore it.
- `pal.Testing`: new PAL group with a `Fail(message string)` function field. `palnative` leaves it nil.
- `cli/internal/testerina`: the CLI-owned pieces.
  - `Collector`: owns the injected compiler plugin and accumulates per-module discoveries.
  - `TestPlan` / `Schedule`: data and ordering.
  - `Runner`: executes a `Schedule` on a `*runtime.Runtime`, owns the PAL hook and per-test stdout capture.
  - `Reporter`: progress and summary output.
- `cli/cmd/test.go`: the cobra command.
- Project API: an injection point for compiler plugins supplied by the host, threaded from
  `ProjectLoadConfig` to the plugin resolver.
- Runtime: `InvokeFunction` releases held locks when the invoked function panics.

## Compilation flow

1. `bal test [path]` loads the project exactly like `bal run` (`runBallerina` steps up to
   `pkg.Compilation()`), except that `ProjectLoadConfig.CompilerPlugins` contains `collector.Plugin()`.
2. The project environment stores the injected plugins. During `PackageCompilation` phase 3 the plugin
   resolver merges them with manifest-declared plugins. An injected plugin is applied to a module iff:
   - the module belongs to the root package (not to a dependency), and
   - the module explicitly imports the plugin's provider package (`ballerina/test`).
   It receives the provider's exported symbol space exactly like a manifest plugin.
3. The plugin (`Collector.collect`) runs at `AfterSemantics`, on each qualifying module concurrently:
   - For each `fn` in `pkg.Functions`, for each attachment in `fn.GetAnnotationAttachments()` whose symbol
     package is `ballerina/test` and whose name is `Config`:
     - `enable`: `true` unless the attachment expression contains an `enable` key whose value expression is a
       `*ast.BLangLiteral` with a bool `Value`. If the whole attachment is constant, `ann.AnnotationValue`
       (a `*values.Map`) is used instead. A non-literal `enable` is a semantic error.
     - `before` / `after`: absent, or a `*ast.BLangVarRef` whose symbol kind is `model.SymbolKindFunction`.
       Recorded as `FunctionRef{Org, Module, Name}` from `compilerCtx.SymbolPackage(ref)` and
       `compilerCtx.SymbolName(ref)`. Visibility is already enforced by the type checker; the plugin adds
       nothing. Any other expression shape is a semantic error.
     - The annotated function must have no required, defaultable or rest parameters and must not be named
       `main` or `init`; otherwise a semantic error is reported at the function position. Defaultable
       parameters live inside `RequiredParams` (flagged by `IsDefaultableParam()`), so the check is
       `len(fn.RequiredParams) == 0 && fn.RestParam == nil`.
   - Stores `ModuleTests` for the module under the collector's mutex, keyed by `ModuleKey{Org, Module}`.
   - Returns the package unchanged.
4. Diagnostics are printed as in `run`; any error aborts. BIR is generated with `NewBallerinaBackend`.
5. `collector.Plan(order)` produces the `TestPlan` with modules in the root package's topological module
   order (taken from `compilation.Resolution().ModuleDependencyGraph()` restricted to root-package modules)
   and tests in source order (position order within the module).

## Execution flow

1. `NewSchedule(plan, filter)` flattens the plan into steps and applies `--tests`. Filter syntax is
   jBallerina's: comma-separated entries, each `name` or `module:name`, `*` matches any run of characters.
   A bare `name` matches that function in any module. Tests that do not match are dropped from the schedule
   and never reported. Matching disabled tests stay in the schedule and are reported `Skipped`.
2. `--list` prints the schedule (`module:name` per line, disabled tests suffixed `(disabled)`) and exits 0.
3. The runner builds the PAL:
   - Starts from `palnative.NewPlatform()`.
   - Replaces `IO.Stdout` with a function that writes to the current test's buffer when a test is running,
     else to `os.Stdout`.
   - Replaces `Signals.Signals` with a channel the runner owns; a goroutine forwards OS signals from the
     palnative channel into it. The runtime reads this channel once at init and panics if it is nil, so it is
     always set.
   - Sets `Testing.Fail` to `runner.fail`.
4. `rt := runtime.NewRuntime(pal, tyEnv)`; `rt.Init(pkg)` for every BIR package in order. `Init` runs
   `$init` and `main` as in `bal run`; an `Init` error is printed and the command exits 1 without running
   tests. `rt.Listen()` runs all `$start` hooks and returns.
5. For each step, on the runner goroutine, sequentially:
   1. Set `current = step`, clear its buffer and failure list.
   2. `before` (if any): `invoke`. Panic or non-nil error return → outcome `Errored`, body and `after`
      skipped.
   3. body: `invoke`.
   4. `after` (if any): `invoke`. Panic or error return → `Errored` even when the body passed.
   5. Outcome: `Failed` if `runner.fail` was called at least once during the step (regardless of how any
      invoke ended); else `Errored` if any invoke panicked or returned an `error` value; else `Passed`.
      Disabled tests are `Skipped` without invoking anything.
   6. `reporter.Report(result)`; `current = nil`.
   `invoke` = `runtime.LookupFunction` + `runtime.InvokeFunction` wrapped in `recover`. A recovered
   `*values.Error` contributes its `Message`; any other recovered value contributes `fmt.Sprint(v)`.
6. `runner.fail(msg)` appends `msg` to `current`'s failure list. When `current` is nil (called from `init`,
   `main`, a `$start` hook, or a strand outliving its test) it is appended to the run-level failure list; a
   non-empty run-level list makes the run fail.
7. Shutdown: send `pal.GracefulStop` on the runner's signal channel without blocking, then read
   `<-rt.ExitStatus`. The runtime's exit code is ignored; if no listeners existed the channel already holds
   `0`.
8. `reporter.End(summary)` prints failed and errored tests with their messages and captured stdout, then
   totals. The command returns a non-nil error (exit 1) if any test is `Failed` or `Errored`, or the run-level
   failure list is non-empty.

## Reporter

- `plainReporter` (non-TTY): one line per result: `PASS|FAIL|ERROR|SKIP <module>:<name>`.
- `ttyReporter` (`isTerminal()`): a single line rewritten with `\r`:
  `Running <done>/<total>  passed <p>  failed <f>  errored <e>  skipped <s>`. Final newline before the
  summary.
- Both print the same summary: for each `Failed`/`Errored` result, `module:name`, each message, and captured
  stdout indented; then `N passing, N failing, N errored, N skipped`.

## `assertFail` path

`assertFail` is Ballerina; it calls the native `externFail`, which invokes `Testing.Fail` when set and
returns an `error` value, which `assertFail` panics. Under `bal run` the hook is nil, so `assertFail` is a
plain panic with the message. `returns never` is implemented in Ballerina (validated: a never-returning
Ballerina function under `trap` compiles and runs), so no `never`-returning extern is needed.

# API changes

## lib/stdlibs/ballerina/test/0.0.1/go1.27/test.bal (new)

```ballerina
public type TestConfig record {|
    boolean enable = true;
    (function () returns (any|error)) before?;
    (function () returns (any|error)) after?;
|};

public annotation TestConfig Config on function;

public function assertFail(string msg = "Test Failed!") returns never {
    panic externFail(msg);
}

function externFail(string msg) returns error = external;
```

Behavior: `Config` is runtime-visible (`on function`, not `source function`); nothing reads the value at
runtime today. `externFail` calls the PAL hook and returns `error(msg)`.

## lib/stdlibs/ballerina/test/0.0.1/go1.27/{Ballerina.toml, Bala.toml, Dependencies.toml, README.md} (new)

Same shape as `ballerina/io` with `name = "test"`, `platform = "go1.27"`.

## lib/stdlibs/ballerina/test/0.0.1/go1.27/native/test.go (new)

```go
func init() // runtime.RegisterModuleInitializer(initTestModule)
func initTestModule(rt *runtime.Runtime) // RegisterExternFunction(rt, "ballerina", "test", "externFail", externFail)
func externFail(ctx *extern.Context, args []values.BalValue) (values.BalValue, error)
```

Behavior: reads `args[0]` as `values.BalValue` string; if `ctx.Env.Platform.Testing.Fail != nil` calls it
with the message; returns `values.NewErrorWithMessage(msg), nil`.

## lib/rt/libs.go

- Current: blank imports of every stdlib `native` package.
- Proposed: add `_ ".../lib/stdlibs/ballerina/test/0.0.1/go1.27/native"`.

## platform/pal/platform.go

- Current:
  ```go
  type Platform struct { IO IO; FS FS; OS OS; Time Time; HTTP HTTP; Signals SignalSource }
  ```
- Proposed:
  ```go
  type Platform struct { IO IO; FS FS; OS OS; Time Time; HTTP HTTP; Signals SignalSource; Testing Testing }

  // Testing lets library code signal test outcomes to the host. All fields may be nil.
  type Testing struct {
      Fail func(message string)
  }
  ```
- Behavior: purely additive. `palnative.NewPlatform` leaves `Testing` zero-valued. Callers must nil-check.

## compilerplugin/compilerplugin.go

- Current: `Stage`, `AfterSemantics`, `PackageTransformer`, `CompilerPlugin{After, PackageTransformer}`.
- Proposed additions:
  ```go
  // Provider identifies the package whose exported symbol space a plugin receives and whose import
  // activates it.
  type Provider struct {
      Org     string
      Package string
  }

  // InjectedPlugin is a compiler plugin supplied by the host for one compilation rather than declared by a
  // package manifest.
  type InjectedPlugin struct {
      Provider Provider
      Plugin   CompilerPlugin
  }
  ```
- Behavior: data only.

## projects/project_loader.go

- Current:
  ```go
  type ProjectLoadConfig struct { BuildOptions *BuildOptions; Repositories []Repository; BallerinaEnvFs fs.FS }
  ```
- Proposed: add `CompilerPlugins []compilerplugin.InjectedPlugin`.
- Behavior: `createEnvironmentWithRepositories` and `createWorkspaceEnvironment` pass
  `cfg.CompilerPlugins` to the environment builder via `WithCompilerPlugins`. Both build-project and
  workspace-member environments carry the plugins. Bala projects loaded with a shared environment inherit it.

## projects/project_environment_builder.go

- Current: `ProjectEnvironmentBuilder{fsys, repositories, buildOptions}` with `WithRepositories`,
  `WithBuildOptions`, `Build`.
- Proposed:
  ```go
  func (b *ProjectEnvironmentBuilder) WithCompilerPlugins(plugins []compilerplugin.InjectedPlugin) *ProjectEnvironmentBuilder
  ```
  plus a private `compilerPlugins []compilerplugin.InjectedPlugin` field; `Build` copies it onto the
  `Environment`.

## projects/env.go

- Current: `Environment{fsys, compilerEnv, packageCache, packageResolver, resolutionOptions, publicSymbols}`.
- Proposed: add private field `injectedPlugins []compilerplugin.InjectedPlugin` and accessor
  ```go
  func (e *Environment) injectedCompilerPlugins() []compilerplugin.InjectedPlugin
  ```
  The field cannot share the accessor's name.

## projects/compiler_plugin_registry.go

- Current:
  ```go
  func newCompilerPluginResolver(modules []*moduleContext) *compilerPluginResolver
  func (r *compilerPluginResolver) pluginsFor(module *moduleContext) ([]resolvedCompilerPlugin, error)
  ```
- Proposed:
  ```go
  func newCompilerPluginResolver(
      modules []*moduleContext,
      rootPackage PackageDescriptor,
      injected []compilerplugin.InjectedPlugin,
  ) *compilerPluginResolver
  ```
  `compilerPluginResolver` gains `rootPackage PackageDescriptor` and `injected []compilerplugin.InjectedPlugin`.
  `PackageDescriptor` is passed by value: `packageContext.getDescriptor()` returns a value and
  `PackageDescriptor.Equals` takes one.
  `resolvedCompilerPlugin` gains `injected bool` (for error messages only).
- Behavior of `pluginsFor`: unchanged for manifest plugins. Afterwards, for each injected plugin in order:
  skip unless the module's package descriptor equals `rootPackage` and `module.explicitImports` contains an
  import whose `org`/`moduleName` equals the provider. Look up the provider in `r.providers` (it exists iff
  the provider package is part of the compilation; if absent, skip). Append a `resolvedCompilerPlugin` whose
  `provider` is that entry, `declaration.function` is `"<injected>"`, `position` is the matching import.
  Injected plugins run after manifest plugins for the same module.

## projects/package_compilation.go

- Current: `c.compilerPluginManager = newCompilerPluginResolver(modules)`.
- Proposed: pass `c.rootPackageContext.getDescriptor()` and
  `c.rootPackageContext.project.Environment().injectedCompilerPlugins()`.
- Behavior of `runCompilerPlugins`: unchanged. Error messages for injected plugins use the provider key and
  the literal function name `<injected>`.

## runtime/runtime.go

- Current:
  ```go
  func InvokeFunction(rt *Runtime, fn any, args []values.BalValue) (values.BalValue, error) {
      cx := exec.CreateContext(rt.env)
      return exec.Invoke(cx, fn, args)
  }
  ```
- Proposed: same signature. Adds a deferred `recover` that calls `cx.ReleaseAllHeldLocks()` and re-panics
  with the same value.
- Behavior: panics still propagate to the caller unchanged; locks acquired by the invoked strand are no
  longer leaked. Matches what `RunEntrypoints` and `startFuture` already do.

## cli/internal/testerina/plan.go (new)

```go
type ModuleKey struct{ Org, Module string }

type FunctionRef struct{ Org, Module, Name string } // zero value means none
func (f FunctionRef) IsZero() bool

type TestFunction struct {
    Name     string
    Enabled  bool
    Before   FunctionRef
    After    FunctionRef
    Position diagnostics.Location
}

type ModuleTests struct {
    Key   ModuleKey
    Tests []TestFunction // source order
}

type TestPlan struct {
    Modules []ModuleTests // root package topological module order
}
```

## cli/internal/testerina/collector.go (new)

```go
type Collector struct {
    mu      sync.Mutex
    modules map[ModuleKey]ModuleTests
}

func NewCollector() *Collector
func (c *Collector) Plugin() compilerplugin.InjectedPlugin
func (c *Collector) Plan(order []ModuleKey) TestPlan
func (c *Collector) collect(*context.CompilerContext, model.ExportedSymbolSpace, *ast.BLangPackage) (*ast.BLangPackage, error)
```

Behavior: `Plugin()` returns provider `{"ballerina", "test"}`, stage `AfterSemantics`, transformer
`c.collect`. `collect` is safe for concurrent calls. `Plan` emits modules in `order`, skipping keys with no
tests, and sorts each module's tests by `Position`.

## cli/internal/testerina/schedule.go (new)

```go
type Step struct {
    Module ModuleKey
    Test   TestFunction
}

type Schedule struct{ Steps []Step }

type Filter []string // parsed --tests entries

func ParseFilter(spec string) (Filter, error)      // "" means match all
func (f Filter) Matches(module ModuleKey, name string) bool
func NewSchedule(plan TestPlan, filter Filter) Schedule
```

Behavior: `ParseFilter` splits on `,`, trims, rejects empty entries. An entry with `:` is
`modulePattern:namePattern`; without, `namePattern` against any module. `*` is the only metacharacter.

## cli/internal/testerina/runner.go (new)

```go
type Outcome uint8
const (
    Passed Outcome = iota
    Failed
    Errored
    Skipped
)

type Result struct {
    Step     Step
    Outcome  Outcome
    Messages []string // Fail hook messages, panic/error messages
    Stdout   []byte
    Duration time.Duration
}

type Summary struct {
    Results     []Result
    RunFailures []string // Fail hook calls outside any test
}
func (s Summary) Failed() bool

type Runner struct { /* rt, pal channel, current step state under mutex */ }

func NewRunner(platform pal.Platform, osSignals <-chan pal.Signal, tyEnv semtypes.Env) *Runner
func (r *Runner) Platform() pal.Platform            // platform with Stdout, Signals, Testing overridden
func (r *Runner) Start(rt *runtime.Runtime)         // records rt after Init/Listen
func (r *Runner) Run(schedule Schedule, reporter Reporter) Summary
func (r *Runner) Shutdown(rt *runtime.Runtime)      // non-blocking GracefulStop send, then <-rt.ExitStatus
```

Behavior as described in Execution flow steps 5 to 7. The runner must be constructed before
`runtime.NewRuntime` because the runtime copies `pal.Platform` by value.

## cli/internal/testerina/reporter.go (new)

```go
type Reporter interface {
    Begin(total int)
    Report(Result)
    End(Summary)
}

func NewPlainReporter(w io.Writer) Reporter
func NewTTYReporter(w io.Writer) Reporter
```

## cli/cmd/test.go (new)

```go
var testOpts struct {
    tests string // --tests
    list  bool   // --list
    // dump/stats/log-file flags identical to run
}
var testCmd = &cobra.Command{Use: "test [<package-dir> | .]", RunE: runTests}
func runTests(cmd *cobra.Command, args []string) error
```

Behavior: single `.bal` files are rejected (same rule as `run`, since `Load` on a file yields a single-file
project; tests require a package). Workspace handling copies `run`. Native re-exec via
`execWithNativeRunner` is applied exactly as `run` does, so packages with native Go dependencies work.
Returns `fmt.Errorf("tests failed")` (exit 1) when `Summary.Failed()`; compilation errors and `Init`
failures return errors as `run` does.

## cli/cmd/bal.go

- Current: registers `new, run, pack, build, push, version`.
- Proposed: add `rootCmd.AddCommand(testCmd)`.

## cli/go.mod

- Proposed: add direct requires for `ast`, `context`, `model`, `compilerplugin`, `values` (currently indirect
  or absent) since `cli/internal/testerina` imports them.

## Unchanged

`bir`, `birgen`, `desugar`, `semantics`, `runtime/internal/*`, `palnative`, `compilerpluginregistry`,
`compiler-tools/plugin-gen`.

# Tests

All as corpus fixtures under `corpus/cli/testdata/test/<case>/` driven by a new `bal test` section in
`corpus/cli_integration_test.go`, asserting stdout (non-TTY reporter), stderr, and exit code. Each fixture is
a build project importing `ballerina/test`.

- Basic pass: two enabled tests, no before/after → both `PASS`, summary `2 passing`, exit 0.
- `assertFail` direct: one test calls `test:assertFail("boom")` → `FAIL`, summary shows `boom`, exit 1.
- `assertFail` under `trap`: test traps two `assertFail` calls and returns normally → `FAIL` with both
  messages.
- Panic in body (non-assert): `panic error("x")` → `ERROR` with message `x`, exit 1.
- Error return: test `returns error?` and returns an error → `ERROR`.
- `before` runs and its stdout is captured; `before` panic → `ERROR`, body not run (assert via io:println
  absence).
- `after` runs after a passing body; `after` panic → `ERROR`.
- `before`/`after` referencing a public function in another module of the same package → runs.
- `before` referencing a non-public function in another module → compile error from the type checker, exit 1.
- `enable: false` → `SKIP`, not counted as failure, exit 0.
- `enable` given a non-literal expression → semantic error from the plugin.
- Test function with a parameter → semantic error from the plugin.
- `@test:Config` on `main` → semantic error.
- `--tests` filtering: `name`, `module:name`, `*` patterns; non-matching tests absent from output; matching
  disabled test reported `SKIP`.
- `--list` prints `module:name` per line with `(disabled)` suffix and runs nothing, exit 0.
- Multi-module package: tests in default module and a submodule run in topological order, output grouped
  accordingly.
- Listener present: a module `http:Listener` with a service; a test issues a request via `http:Client` and
  asserts on the response; run completes and process exits (graceful stop delivered).
- `assertFail` inside a resource invoked by a test → the invoking test is `FAIL`.
- `assertFail` inside a worker started by a test and awaited → `FAIL`.
- `assertFail` called from `main` → run-level failure, exit 1, tests still run.
- `main` panics → `Init` error printed, no tests run, exit 1.
- Panic inside a `lock` block in one test, next test enters the same `lock` → second test runs (no deadlock).
- Stdout capture: `io:println` inside a passing test is not printed; inside a failing test it appears in the
  summary.
- Same package under `bal run`: `main` runs, test functions are not invoked, `assertFail` never called, exit 0.
- Same package under `bal build` then executing the binary: as `bal run`.
- Package that does not import `ballerina/test` under `bal test`: `0 passing`, exit 0, plugin not applied.
- Unit tests in `projects/compiler_plugin_registry_test.go`: injected plugin applied only to root-package
  modules that import the provider; not applied to dependency modules; runs after manifest plugins.
- Unit test in `runtime/lifecycle_test.go`: `InvokeFunction` releases a held lock when the callee panics.
