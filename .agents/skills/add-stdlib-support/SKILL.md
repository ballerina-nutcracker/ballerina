---
name: add-stdlib-support
description: Port a new ballerina/<name> stdlib package from jBallerina to this Go-native interpreter. Use when the user asks to migrate, port, or add a Ballerina standard library module that does not yet exist under `lib/stdlibs/ballerina/`. For filling gaps in an existing stdlib, use `fill-stdlib-gap` instead.
---

# Adding a New Standard Library Package

End-to-end workflow for porting a `ballerina/<name>` package from jBallerina (Java) into this repo. Follow the steps in order; do not skip the gates.

| Step | Purpose | Gate |
|---|---|---|
| 1 | Acquire the jBallerina reference | blocked until user provides the path |
| 2 | Resolve imports, check existing stdlib coverage | — |
| 3 | Cross-check language support | — |
| 4 | Propose plan + showcase `.bal` | **user approval required** |
| 5 | Behavioural parity analysis | **parity table required** |
| 6 | Evaluate Go libraries (if any) | **user approval before touching `go.mod`** |
| 7 | Implement | — |
| 8 | Tests + coverage | subset choice needs user confirmation |
| 9 | README + docs | — |
| 10 | Verify | full checklist, incl. `validate-stdlib-contract` PASS |

All coding rules and the PAL constraint live in `AGENTS.md` at the repo root — read it before implementing. This skill encodes the *process*, not the rules. Supporting files in this skill's directory (`templates/`, `references/`) hold the manifest/source templates and shared patterns — read each one when its step tells you to.

**Golden rule:** two things must hold when this port is done — (1) the Ballerina public interface must stay identical to jBallerina's (never break customer code — see `validate-stdlib-contract`), and (2) if the jBallerina reference root ships a `docs/spec/spec.md`, its prose must exactly match the Go implementation's actual behaviour for every in-scope feature. A spec sentence describing behaviour the Go code doesn't have, or silent on a divergence the Go code does have, is a defect to resolve (by fixing the implementation to match intended design, or by recording the divergence) before the port is done — not something to leave unreconciled.

If the user wants to fix a gap in a stdlib that already exists under `lib/stdlibs/ballerina/<name>/`, use `fill-stdlib-gap` instead — this skill is heavyweight by design.

## 1. Acquire the jBallerina reference

Ask the user for the path to the corresponding jBallerina **library implementation root**, e.g. `~/github/ballerina-platform/module-ballerina-<name>/`. Do not proceed without it.

That root should contain:

- `ballerina/` — the Ballerina-side source (public API, type declarations, extern function signatures).
- `native/` *(optional)* — the Java native implementation backing the extern functions. Pure-Ballerina libraries do not have this directory; that's fine, just note it.
- `docs/spec/spec.md` *(optional)* — the package's specification, if the jBallerina repo ships one. Prose written for humans; often states intent, edge-case handling, and constraints that neither `.bal` signatures nor Java code spell out. Read it alongside `ballerina/` and `native/`, not as a substitute for either.

Then:

- Read every `.bal` file under `<root>/ballerina/`, excluding `tests/` and `build/`, to enumerate the full jBallerina feature set and identify which functions are `external`.
- If `<root>/native/` exists, read the Java sources backing those extern functions. This is the authoritative source of truth for runtime semantics — error wording, edge-case handling, parsing rules, numeric behaviour — and is what the Go natives must match for parity (see Step 5). Don't infer behaviour from `.bal` signatures alone when Java source is available.
- If `<root>/docs/spec/spec.md` exists, read it in full and note every behavioural claim it makes for features in scope. Treat divergences between the spec and the Java/`.bal` source as a flag to raise with the user (Step 4), not something to silently resolve one way.

## 2. Resolve imports and check existing stdlib coverage

Scan the jBallerina source for `import ballerina/<X>` statements.

- For each `<X>` **not** already present under `lib/stdlibs/ballerina/<X>/`: tell the user that dependency must be implemented first. If they ask to continue anyway, narrow the plan to only features that don't depend on `<X>`.
- For each `<X>` already present: read `lib/stdlibs/ballerina/<X>/0.0.1/go1.2/README.md` and note every row whose status is **Not Yet Supported**, **Partially Supported**, or **Cannot Support**, plus anything under **Notable Behavioural Changes**. If our in-scope features depend on any of those gaps or divergences, surface them in the plan (Step 4) under a **Dependency Limitations** section.
- **Exception**: `ballerina/jballerina.java.arrays` will not get a Go equivalent. Plan to replace its uses with Go-native equivalents inside the `native/` layer.
- **Cross-stdlib imports must be declared in `Dependencies.toml`** — see `templates/manifests.md` for the format and why missing entries cause `Unknown import: ballerina/<dep>` at runtime.
- **Langlib imports need compiler wiring**: if the `.bal` source imports a langlib (`import ballerina/lang.<x>;`), that import only resolves through the `isLangImport` switch in `semantics/symbol_resolver.go` — check the langlib in question is wired there before assuming it works.

Do not silently drop features because of a missing import or inherited dependency gap — always flag and confirm.

If a dependency's README claims a feature is `Supported` but it actually diverges from its own documented behaviour — a bug, not a catalogued gap — that's separate from the check above: follow `references/reporting-limitations.md`.

## 3. Cross-check language support

Read `AGENTS.md` (root) in full, especially the **Interpreter stages** and **Coding style** sections. If a planned feature uses a construct known to fail in this interpreter (`distinct` error subtypes, `readonly &` intersections, `stream` type, XML, full `typedesc` parameter handling), drop or defer the feature and note it in the plan.

### Handling unexpected compile failures during implementation

When the interpreter panics or emits compile errors that are **not explained** by `AGENTS.md`, stop and present the developer with these options — **do not silently pick one**:

> **Unexpected language limitation found:** `<construct>` is not supported (`panic: <message>`).
>
> Options:
> 1. **Fix the interpreter** — implement this construct in the compiler/BIR pipeline. Requires a separate change.
> 2. **Work around in Ballerina** — rewrite the Ballerina source to avoid the construct.
> 3. **Move to Go native** — replace the Ballerina function body with `= external` and implement the logic in `native/`.
> 4. **Scope out this feature** — mark it `Not Yet Supported` in the README and move on.
>
> Which option do you prefer?

After the developer responds, apply the chosen resolution before continuing. Regardless of which option is chosen, this is a language limitation worth tracking upstream independently of the local workaround — draft an issue per `references/reporting-limitations.md` and point the developer to https://github.com/ballerina-nutcracker/ballerina/issues.

## 4. Propose a plan and a showcase `.bal` file *(GATE: wait for user approval)*

Produce both:

- **Plan** — a list of features in scope for this iteration, with explicit "Not Yet Supported" notes for anything left out. Include a **Dependency Limitations** section listing any inherited gaps from the README of every `ballerina/<X>` package we import (per Step 2).
- **Showcase `.bal` file** — a small program that exercises every feature in scope end-to-end. Use `@output` markers for expected output.

**Wait for the user to approve both the plan and the showcase file before touching any Go code.**

## 5. Behavioral parity analysis *(GATE: parity table required)*

The Go-native behaviour **must match the jBallerina (JVM) behaviour** for every supported feature. Users migrating from jBallerina must not observe breaking changes. Before writing any Go code, produce a parity table for each in-scope feature:

| Feature | Known Go/JVM divergence risk | Avoidable? | Resolution |
|---|---|---|---|
| ... | ... | ... | ... |

Read `references/parity-risks.md` for the hot-spots to investigate (decimal precision, UTF-8 vs UTF-16, error-message rules, numeric edge cases) plus a domain-specific example. Where a row's behaviour is verifiable by running code, verify it with the **`run-jballerina`** skill (`bal run` the probe on jBallerina, compare against this interpreter) rather than reasoning from documentation.

### Spec cross-check (only if `docs/spec/spec.md` exists)

Add a row to the parity table for every behavioural claim the spec makes about an in-scope feature, and verify the planned Go implementation will match it exactly — same edge-case handling, same defaults, same error conditions. A spec/implementation mismatch is treated the same as a spec/Java mismatch: raise it with the user rather than picking a side silently. This check is repeated against the *actual* implementation in Step 10 — Step 5 catches mismatches before code is written, Step 10 catches drift introduced while writing it.

### Rules

- **Avoidable** divergences (resolvable in the Go layer) — fix before merging.
- **Unavoidable** divergences (architectural Go/JVM constraint) — record in the README under **Notable Behavioural Changes** *before* implementing.
- **A divergence traced to an actual bug in a dependency stdlib** (not an architectural constraint, and not already a catalogued gap) — don't just record it as unavoidable; also follow `references/reporting-limitations.md` to get it tracked upstream.
- Do not proceed to Step 6 without a complete parity table, even if every row says "No risk identified."

## 6. Evaluate Go libraries *(GATE: wait for approval before touching `go.mod`)*

Only if external Go dependencies are needed. For each external functionality, evaluate 2–3 candidate Go libraries on:

| Axis | What to check |
|---|---|
| Availability | Active maintenance, last release within ~12 months, owner responsive |
| Licensing | Prefer MIT / Apache-2.0 / BSD. **Flag GPL/AGPL/LGPL** — needs explicit user sign-off |
| Stability | v1.x+, release cadence, open-issue health |
| Dependency footprint | Transitive dep count, binary-size impact, CGo |

Present as a small table with a recommendation. **Wait for user approval** before adding the dependency to `go.mod`. If no external deps are needed, skip this step.

## 7. Implement

### File layout

```
lib/stdlibs/ballerina/<name>/0.0.1/go1.2/
├── Ballerina.toml          # package manifest
├── Bala.toml               # build/platform manifest
├── Dependencies.toml       # package dependencies
├── README.md               # via stdlib-readme-format skill
├── <name>.bal              # public API surface
└── native/                 # OPTIONAL — omit if pure Ballerina
    └── <name>.go           # Go native implementations
```

Multi-file `.bal` and multi-file `native/` are both supported — see exemplars below. For dotted names like `math.vector`, the single `.bal` file is named `math.vector.bal`.

Templates:

- **Manifests** (`Ballerina.toml`, `Bala.toml`, `Dependencies.toml` with/without cross-stdlib deps) — `templates/manifests.md`.
- **Source skeletons** (`.bal` with license header, `native/<name>.go`) — `templates/source-files.md`.

Shared patterns — read the relevant file only when the situation applies:

- **PAL hookup** (new platform interaction: io, fs, http, env, time) — `references/pal.md`. Three files must change; missing `TestPal` = nil-pointer panics in corpus tests.
- **Native state behind a map/record or object value** (parsed keys, compiled patterns, handles) — `references/native-state.md`. Never add fields to `values.Map` for this.
- **bal↔Go JSON conversion** — reuse the shared helpers `values.BalToGoJSON` / `values.GoToBalValue`; never duplicate the conversion per-stdlib.

### Wire-up checklist *(every new stdlib — missing any = silent failure)*

1. **`lib/rt/libs.go`** — add a blank import so the `init()` in the native package runs at binary start:
   ```go
   _ "ballerina-lang-go/lib/stdlibs/ballerina/<name>/0.0.1/go1.2/native"
   ```
   Without this, all `= external` functions produce "function not found" at runtime even though the binary compiles cleanly. Skip this line if your stdlib has no `native/` directory.

2. **`test_util/testphases/phases.go`** — append an entry to `builtinStdlibs`:
   ```go
   {"ballerina", "<name>", "0.0.1"},
   ```
   Without this, corpus tests cannot resolve `import ballerina/<name>` even if everything else compiles. If the new stdlib imports other stdlibs, place this entry **after** those dependencies in the list so the loader compiles them in order.

3. **`Dependencies.toml`** — if the `.bal` source imports any other stdlib (`import ballerina/<dep>;`), declare it per `templates/manifests.md`. Without this, the full project resolver will not discover the dependency and every user `.bal` file importing this stdlib will fail with `Unknown import: ballerina/<dep>`.

4. **`projects/module_resolver.go`** — usually no change. The existing `packageNameCandidates` handles dotted names (`math.vector` → tries `math.vector` then `math`). Read it once to confirm the import in question is covered.

### Coding rules

Follow `AGENTS.md` (root) — Coding style, Symbols, and PAL sections. Do not restate or re-derive them; when in doubt, re-read the file.

### Canonical exemplars in this repo

| Exemplar | Use when |
|---|---|
| `lib/stdlibs/ballerina/url/0.0.1/go1.2/` | Smallest viable stdlib — 2 extern functions, 1 native file. |
| `lib/stdlibs/ballerina/io/0.0.1/go1.2/` | Multi-file `.bal` (constants/types/print/file) + multi-file `native/` (`io.go` + `file_io.go`). |
| `lib/stdlibs/ballerina/time/0.0.1/go1.2/` | Heavy native implementation with PAL usage and documented behavioural divergences. |
| `lib/stdlibs/ballerina/http/0.0.1/go1.2/` | Class-based stdlib (Client init wrapper). |
| `lib/stdlibs/ballerina/math.vector/0.0.1/go1.2/` | Pure Ballerina — no `native/` directory at all. |

## 8. Tests

### Where library corpus tests live

Corpus tests for a stdlib port go under `corpus/bal/library/subset<N>/` — a **flat** directory of `<name>-<suffix>.bal` files, e.g. `corpus/bal/library/subset2/crypto-hash1-v.bal`. This is a different directory family from the generic language-feature subsets (`corpus/bal/subset1/` … `corpus/bal/subset9/`, each internally split into `NN-category/` subfolders like `08-network/`) — do not put library tests there.

Each `library/subset<N>` is a released library-support milestone, documented in `doc/library/subset<N>.md` (compare with the language-feature milestones in `doc/lang/subset<N>.md`, which describe an unrelated numbering track — `library/subset2` and `lang/subset2` are not the same milestone). For example, `doc/library/subset2.md` documents the `crypto`, `io` (file I/O), `log`, `os`, `random`, `math.vector`, `time`, and `url` surface, plus expanded `http`.

**Ask the developer which subset this port's tests belong in before writing any test file** — this is a release-scoping decision, not something to infer:
- An **existing** subset (e.g. `subset2`) — the new module joins that release milestone, alongside `corpus/bal/library/subset2/`'s existing files.
- A **new** subset (`subset<N+1>`, one past the highest existing `library/subsetN` directory) — create `doc/library/subset<N+1>.md` following `subset2.md`'s intro-paragraph pattern ("Subset N extends the released subset N-1 with …").

Per-stage golden directories (`corpus/ast/library/subset<N>/`, `corpus/bir/library/subset<N>/`, `corpus/cfg/library/subset<N>/`, `corpus/desugared/library/subset<N>/`, `corpus/parser/library/subset<N>/`) mirror this same layout and are generated automatically via `-update` (Step 8's golden-regeneration step below) — no special handling needed, they follow whichever subset directory the `.bal` files live in.

After the tests pass, add (or extend) the `## [<name>](<jBallerina spec URL>)` section in that subset's `doc/library/subset<N>.md`, documenting the surface actually exercised by these corpus tests — follow the existing heading + `Function | Notes` table (or bullet list) style in `subset1.md`/`subset2.md`. This is a separate, lighter-weight doc from the per-package `README.md` (Step 9) — both need updating.

### Test conventions

- Suffixes per `AGENTS.md`: `*-v.bal` (valid, end-to-end with `@output` markers), `*-e.bal` (compile-time errors, `@error` markers), `*-p.bal` (runtime panics, `@panic` markers), `*-f{v|e|p}.bal` (future, scope-deferred).
- Name files **without leading zeros** in numeric parts (e.g. `print1-v.bal`, not `print01-v.bal`).
- **`*-v.bal` tests must produce empty stderr.** If the stdlib writes to stderr (e.g. logging), structure the test to avoid it — for `log`-style modules, use filtered log levels so nothing is emitted.
- Hand off golden-file regeneration to the **`manage-corpus-tests`** skill:
  ```shell
  go test ./corpus -update
  ```
  Then review `git diff corpus/` before committing (and revert any unrelated golden drift `-update` introduces — see `manage-corpus-tests`).

### Coverage target

Targeting **≥80% coverage** of the new Go code under `native/`.

**This is a real CI gate, not a suggestion.** `.github/workflows/native-ci.yml` runs `.github/scripts/run_native_tests.py --with-coverage` and uploads the resulting profiles to Codecov (`flags: native`); `codecov.yml` sets `coverage.status.patch.default.target: 80%`, which fails the PR check if **patch coverage** (coverage of just the lines added/changed in the diff) drops below 80%. For a brand-new stdlib, essentially every line under `native/` is new, so the whole-package coverage number below is a reliable local stand-in for that patch-coverage check.

**Measure it locally before declaring done** — this mirrors what CI does, without the full 2h suite:
```shell
go test -count=1 -coverpkg=./lib/stdlibs/ballerina/<name>/... \
  -coverprofile=/tmp/<name>-coverage.out -covermode=atomic \
  ./corpus/... ./lib/stdlibs/ballerina/<name>/...
go tool cover -func=/tmp/<name>-coverage.out | grep total
```
If the total is below 80%, find the gaps with `go tool cover -func=/tmp/<name>-coverage.out` (sort by the trailing `%` column) or `go tool cover -html=/tmp/<name>-coverage.out` for an annotated view, then add corpus `.bal` cases to exercise the missing branches — repeat until ≥80%. Do not move on to Step 9 with a known shortfall.

**Drive coverage from `.bal`, not Go unit tests.** The coverage harness runs `./corpus/...` under `-coverpkg=./lib/stdlibs/...`, so a corpus test that calls your extern functions exercises and measures the native Go through the full compiler → BIR → interpreter pipeline. Reach for a Go unit test (`native/<name>_test.go`) **only** for branches genuinely unreachable from Ballerina — defensive type/arity guards, nil guards, interface-contract paths — and keep them minimal with a comment stating why they cannot be hit from `.bal`. Do not add a wrong-type extern arg guard at all (the type checker rejects wrong types at compile time; use `x, _ := args[i].(T)`). See the **`manage-corpus-tests`** skill's "Test philosophy" section.

## 9. README

Author `lib/stdlibs/ballerina/<name>/0.0.1/go1.2/README.md` using the **`stdlib-readme-format`** skill. Load that skill now and run its validation checklist before saving the file. Copy every unavoidable divergence from the Step 5 parity table into **Notable Behavioural Changes** — these must be present before merge.

Then update the top-level aggregator `lib/stdlibs/ballerina/README.md` (same `stdlib-readme-format` skill): add the new package row (alphabetical), recompute the **Total** footer, and mirror this package's behavioural changes into a `### <name>` subsection (only if it has any).

Separately, confirm Step 8's `doc/library/subset<N>.md` update is done — it documents released library-feature milestones and is independent of the per-package `README.md` (which tracks jBallerina-parity status, not release scoping).

## 10. Verify

Before declaring done, check every box:

### Code
- [ ] `go build ./...` — no compilation errors.
- [ ] `go vet ./...` — no vet warnings.

### Tests
- [ ] `go test ./corpus/...` — all corpus tests pass.
- [ ] `go run ./cli/cmd run <showcase>.bal` (or `./bal run <showcase>.bal` if the binary is built) — output matches the `@output` markers exactly.
- [ ] `git diff corpus/` reviewed; every regenerated golden-file line is intentional.
- [ ] New corpus test files follow naming (no leading zeros, correct suffix) and live under `corpus/bal/library/subset<N>/` (the subset confirmed with the developer in Step 8), not the generic `corpus/bal/subset1..9/` tree.
- [ ] Local coverage of the new `native/` package is **≥80%** (Step 8's `go tool cover -func=... | grep total` command). This is what Codecov's patch-coverage check in CI (`native-ci.yml` + `codecov.yml`) will otherwise fail the PR on.

### Parity & contract
- [ ] Every Step 5 parity-table row marked **"Avoidable / Fixed"** verified against jBallerina for at least one representative input, via the **`run-jballerina`** skill.
- [ ] Every unavoidable divergence recorded in **Notable Behavioural Changes**.
- [ ] **Run the `validate-stdlib-contract` skill on the new package — the verdict must be PASS** (or PASS with notes, each note reviewed). This is the final public-interface gate.
- [ ] If `docs/spec/spec.md` exists in the jBallerina reference root: every in-scope behavioural claim it makes matches the shipped Go implementation exactly. Any mismatch found during implementation (not just at Step 5's planning stage) has been resolved — implementation fixed, or the divergence explicitly documented — not left unreconciled.

### Documentation
- [ ] `lib/stdlibs/ballerina/<name>/0.0.1/go1.2/README.md` support table reflects current implementation (no stale `Not Yet Supported` rows for things just implemented).
- [ ] `lib/stdlibs/ballerina/README.md` aggregator updated (new row, recomputed Total footer, behavioural changes mirrored).
- [ ] `stdlib-readme-format` validation checklist passes.
- [ ] `doc/library/subset<N>.md` (the subset agreed with the developer in Step 8) documents this module's newly-supported surface — created fresh if it's a new subset, extended if existing.

### Wire-up
- [ ] `lib/rt/libs.go` blank import added (skip only if pure Ballerina).
- [ ] `test_util/testphases/phases.go` `builtinStdlibs` entry added; placed after any stdlib dependencies in the list.
- [ ] `Dependencies.toml` declares every `import ballerina/<dep>` that appears in the `.bal` source (only needed when cross-stdlib imports exist; omit otherwise).
- [ ] PAL fields (if any added) implemented in `palnative/` and wired into `TestPal`.

### Final report

Summarise:
- What was implemented and what was scoped out (with reasons).
- Any new PAL methods or external Go dependencies added.
- The complete parity table from Step 5.
- The measured `native/` coverage % from Step 8's verify command.
- The `validate-stdlib-contract` verdict.
- Which `corpus/bal/library/subset<N>/` the tests were added to (new or existing) and confirmation `doc/library/subset<N>.md` was updated.
- Any language-limitation or dependency-bug issue drafted per `references/reporting-limitations.md`, and whether the developer filed it.
