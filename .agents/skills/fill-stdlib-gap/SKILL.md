---
name: fill-stdlib-gap
description: Fill a gap in an existing ballerina/<name> stdlib — implement a function marked Not Yet Supported, promote a Partially Supported row, or fix a behavioural divergence. Use when the target stdlib already exists under `lib/stdlibs/ballerina/`. For brand-new stdlibs, use `add-stdlib-support`.
---

# Filling a Gap in an Existing Standard Library

Lightweight 5-step workflow for adding a missing function, promoting a `Not Yet Supported` row to `Supported`, or fixing a divergence. Unlike `add-stdlib-support`, this skill has **no plan-approval gate** and **no library-evaluation gate** — the stdlib already exists, its file layout and wire-up are already in place, and the surface change is small.

If the user wants to port a brand-new stdlib (`lib/stdlibs/ballerina/<name>/` does not exist), use `add-stdlib-support` instead.

Coding rules and the PAL constraint live in `AGENTS.md` at the repo root — read it before editing. Shared templates and patterns live in `add-stdlib-support`'s directory (`../add-stdlib-support/templates/`, `../add-stdlib-support/references/`) — this skill points there rather than repeating them.

**Golden rule** (same as `add-stdlib-support`): the Ballerina public interface must stay identical to jBallerina's, and if the jBallerina reference ships a `docs/spec/spec.md`, its prose must exactly match the Go implementation's actual behaviour for the surface being touched. A spec/implementation mismatch is a defect to resolve before the gap-fill is done — not something to leave unreconciled.

## 1. Identify the gap

Open the target stdlib's README, e.g. `lib/stdlibs/ballerina/io/0.0.1/go1.2/README.md`, and confirm the row to be promoted.

- If the row exists and is **Not Yet Supported** / **Partially Supported** — proceed.
- If the row exists and is already **Supported** — clarify with the user whether they want to fix a divergence (different scope; use behavioural-change analysis only) or whether the row is stale.
- If the row does not exist at all — ask the user to clarify scope. New surface area may belong under `add-stdlib-support` or may just need a new row added to the table.

State back to the user, in one sentence, exactly what will change (e.g., "Promoting *File read — stream of lines* from Not Yet Supported to Supported by implementing `fileReadLinesAsStream`").

## 2. Read jBallerina reference for just this surface

**This step is mandatory and blocking.** Do not proceed to Step 3 until the jBallerina source has been read and the behaviour is confirmed from code — not from doc comments, not from training knowledge.

Ask the user for the path to the corresponding jBallerina **library implementation root**, e.g. `~/github/ballerina-platform/module-ballerina-<name>/`. If the user has not provided it, stop and ask before doing anything else.

Read only the `.bal` and Java code relevant to the targeted function(s) — do not enumerate the whole library. Note:

- Signature and return type.
- Error types raised, and the wording of any error messages produced by the *outer* Ballerina error (not the underlying Java cause).
- Edge cases handled in Java (empty input, malformed input, large inputs, encoding).
- Whether the function is `isolated`, `public`, has a default value, etc.

If `<root>/docs/spec/spec.md` exists *(optional — not every jBallerina repo ships one)*, read the section covering the targeted surface. It's prose written for humans and often states intent or edge-case handling that neither the `.bal` signature nor the Java code spells out directly. Treat any mismatch between the spec and the Java/`.bal` source as something to raise with the user, not something to silently resolve one way.

### What to read for config fields, enums, and modes

When the feature involves a configuration record, enum, or multi-mode flag (e.g. a `compression`, `httpVersion`, or `retryConfig` field), doc comments and type signatures are **insufficient** — they describe intent, not mechanics. For these cases you **must** also read the Java action or handler that consumes the config value at runtime and trace the actual code path for each enum variant or flag value. Common locations:

- Action classes (e.g. `AbstractHTTPAction.java`, `HttpClientAction.java`)
- Configuration handler/builder classes (e.g. `HttpUtil.java`, `ConnectionManager.java`)
- Test files that assert the wire-level behaviour (e.g. header values actually sent)

### Do not infer from training knowledge

Do not assume you know what a jBallerina feature does from prior training. Implementations frequently differ from what documentation or naming implies. If reading the source leaves behaviour ambiguous (e.g. conflicting comments, dead code, platform-specific branches), **stop and ask the user** rather than making an assumption. A wrong assumption that ships silently is worse than a clarifying question.

## 3. Quick parity check

Produce a focused 3-column table for the touched surface only:

| Feature | Risk | Resolution |
|---|---|---|
| ... | ... | ... |

Check the hot-spots in `../add-stdlib-support/references/parity-risks.md`, scoped to just this surface (decimal precision, UTF-8 vs UTF-16 string ops, NaN/overflow, error-message wording on the outer error, plus the module's domain-specific risks). Where a row's behaviour is verifiable by running code, verify it with the **`run-jballerina`** skill — run a small probe on jBallerina (`bal run`) and on this interpreter, and compare — rather than reasoning from documentation.

If `docs/spec/spec.md` exists: check every behavioural claim it makes about the touched surface against what the Go implementation will actually do. Carry any mismatch found here forward to Step 5's verify checklist to confirm it was resolved, not just noticed.

Rules:
- **Avoidable** divergences — fix during Step 4.
- **Unavoidable** divergences — record in the README under **Notable Behavioural Changes** during Step 5.

If every row is "No risk identified", say so and move on.

## 4. Implement

You are editing existing files, **not** creating new ones. In particular:

- **Do not** create new manifest files (`Ballerina.toml`, `Bala.toml` already exist).
- **Do not** modify `lib/rt/libs.go` — the blank import is already there.
- **Do not** modify `test_util/testphases/phases.go` — the `builtinStdlibs` entry is already there.
- **Exception — `Dependencies.toml`**: if the gap being filled adds a new `import ballerina/<dep>;` to the `.bal` source that was not there before, you **must** declare the dependency in `Dependencies.toml` — format and rationale in `../add-stdlib-support/templates/manifests.md` (missing entries cause `Unknown import: ballerina/<dep>` at runtime). Also verify `<dep>` appears before this package in the `builtinStdlibs` list in `test_util/testphases/phases.go` (it almost certainly already does).
- **Langlib imports**: if the gap adds an `import ballerina/lang.<x>;`, that import only resolves through the `isLangImport` switch in `semantics/symbol_resolver.go` — check the langlib is wired there.

What you *do* edit:

- **`lib/stdlibs/ballerina/<name>/0.0.1/go1.2/<name>.bal`** (or sibling `.bal` files like `file.bal`, `types.bal`) — add the public function, type declaration, or extern signature. Preserve the existing license header and doc-comment style. Function names match jBallerina exactly.
- **`lib/stdlibs/ballerina/<name>/0.0.1/go1.2/native/<name>.go`** (or sibling `.go` files like `file_io.go`) — add the Go implementation. Register it in the existing `init<Name>Module` function:
  ```go
  func init<Name>Module(rt *runtime.Runtime) {
      // existing registrations...
      runtime.RegisterExternFunction(rt, orgName, moduleName, "externNewFn", externNewFnExtern(rt))
  }
  ```
  If the new logic is large enough to warrant a new file, create `native/<feature>.go` alongside the existing ones — keep `package native` and reuse the `orgName` / `moduleName` constants already defined.

Shared patterns — read the relevant file only when the situation applies:

- **PAL hookup** (new platform op not already covered by the PAL) — `../add-stdlib-support/references/pal.md`. Three files must change; missing `TestPal` = nil-pointer panics in corpus tests.
- **Native state behind a map/record or object value** — `../add-stdlib-support/references/native-state.md`. Never add fields to `values.Map` for this; see `crypto/native/keydata.go` for the weak-map reference implementation.
- **bal↔Go JSON conversion** — reuse the shared helpers `values.BalToGoJSON` / `values.GoToBalValue`; never duplicate the conversion per-stdlib.

Coding rules: follow `AGENTS.md` (license header on every new file, no per-line comments, no new public symbols unless required by the public API).

## 5. Test, document, and verify

### Tests

Library corpus tests live under `corpus/bal/library/subset<N>/` — a flat directory of `<name>-<suffix>.bal` files, e.g. `corpus/bal/library/subset2/crypto-hash1-v.bal` — a different, stdlib-specific directory family from the generic `corpus/bal/subset1..9/NN-category/` language-feature tests. Each `library/subset<N>` is a released library-support milestone documented in `doc/library/subset<N>.md`.

Find this stdlib's existing tests first: `find corpus/bal/library -name '<name>-*.bal'` locates which `subset<N>` it currently lives in — reuse that one by default. **Ask the developer to confirm** rather than assuming, though: if this gap-fill is significant enough to be its own release milestone, they may want it filed under a *new* `subset<N+1>` instead (create `doc/library/subset<N+1>.md` following `subset2.md`'s intro-paragraph pattern in that case). Suffixes per `AGENTS.md`: `*-v.bal` (valid), `*-e.bal` (compile errors), `*-p.bal` (panics). No leading zeros in numeric parts. **`*-v.bal` tests must produce empty stderr** — structure the test (e.g. filtered log levels) so nothing is emitted there.

Cover the new behaviour from `.bal` — a corpus test exercises the full compiler → BIR → interpreter pipeline and is measured by the native-coverage harness (`-coverpkg=./lib/stdlibs/...` over `./corpus/...`). Add a Go unit test only for branches genuinely unreachable from Ballerina (defensive type/arity guards, nil guards, interface-contract paths), kept minimal with a comment explaining why. Don't write wrong-type extern arg guards — the type checker rejects wrong types at compile time. See the **`manage-corpus-tests`** "Test philosophy" section. If you find existing native code that can never execute through Ballerina, remove it rather than testing it.

**Coverage gate, same as `add-stdlib-support`:** `.github/workflows/native-ci.yml` uploads coverage to Codecov (`flags: native`), and `codecov.yml`'s `coverage.status.patch.default.target: 80%` fails the PR if **patch coverage** — coverage of just the lines this change adds or touches — drops below 80%. Unlike a brand-new stdlib, the package's overall coverage % is *not* a reliable stand-in here: the existing, already-tested code dilutes it, so a poorly-tested new function can hide inside a healthy-looking package total. Check the touched lines directly:

```shell
go test -count=1 -coverpkg=./lib/stdlibs/ballerina/<name>/... \
  -coverprofile=/tmp/<name>-coverage.out -covermode=atomic \
  ./corpus/... ./lib/stdlibs/ballerina/<name>/...
go tool cover -func=/tmp/<name>-coverage.out | grep '<newFunctionName>'
```
or open an annotated view of just the new/changed lines with `go tool cover -html=/tmp/<name>-coverage.out`. Add corpus cases until every new branch is exercised — don't rely on the package-total percentage looking fine.

Regenerate goldens via the **`manage-corpus-tests`** skill:
```shell
go test ./corpus -update
```
Review `git diff corpus/` before committing, and revert any unrelated golden drift `-update` introduces (some stages have non-deterministic ordering).

### Documentation

Update the README row via the **`stdlib-readme-format`** skill:

- Promote the affected row's status (`Not Yet Supported` → `Supported`, or `Partially Supported` → `Supported` if the caveats are resolved).
- If the parity check in Step 3 surfaced an unavoidable divergence, add it to **Notable Behavioural Changes**.
- Update the top-level aggregator `lib/stdlibs/ballerina/README.md`: recount this package's row and recompute the **Total** footer; if a behavioural change was added or removed, mirror it into the package's `### <name>` subsection of the consolidated section.
- Re-run the full `stdlib-readme-format` validation checklist against the updated README (catches pre-existing violations too).
- Update `doc/library/subset<N>.md` (the subset the tests were added to above) to document the newly-covered surface — this is separate from, and in addition to, the per-package `README.md`.

### Verify checklist

- [ ] `go build ./...` — no compilation errors.
- [ ] `go vet ./...` — no vet warnings.
- [ ] `go test ./corpus/...` — all corpus tests pass.
- [ ] `go run ./cli/cmd run <test>.bal` for the new corpus test(s) — output matches `@output` markers.
- [ ] New corpus test files live under `corpus/bal/library/subset<N>/` (the subset confirmed with the developer above), not the generic `corpus/bal/subset1..9/` tree.
- [ ] `doc/library/subset<N>.md` documents the newly-covered surface.
- [ ] Every new/touched line in `native/` is exercised (checked via `go tool cover -func=...` or `-html=...`, not the package-total %) — Codecov's patch-coverage check (`codecov.yml`, `native-ci.yml`) targets 80% on just the diff and will fail the PR otherwise.
- [ ] README row status reflects what's now implemented.
- [ ] `lib/stdlibs/ballerina/README.md` aggregator updated (package row recounted, Total footer recomputed, behavioural changes mirrored if any changed).
- [ ] `stdlib-readme-format` validation checklist passes.
- [ ] Any unavoidable divergence is in **Notable Behavioural Changes**.
- [ ] **Run the `validate-stdlib-contract` skill on this package** (at minimum, review its diff output for the touched surface) — the verdict must be PASS, or PASS with notes that you have reviewed.
- [ ] If a new `import ballerina/<dep>` was added to the `.bal` source: `Dependencies.toml` updated with the new entry and `dependencies = [...]` field.
- [ ] PAL fields (if any added) implemented in `palnative/` and wired into `TestPal`.
- [ ] If `docs/spec/spec.md` exists in the jBallerina reference root: every behavioural claim it makes about the touched surface matches the shipped Go implementation exactly — any mismatch found during Step 3 or implementation has been resolved, not left unreconciled.

### Final report

In one short paragraph: which row was promoted, what was added (function names, file paths), which `corpus/bal/library/subset<N>/` the tests landed in (and whether `doc/library/subset<N>.md` was created or extended), any divergences recorded, the `validate-stdlib-contract` verdict, and confirmation that the new/touched lines are covered (per the coverage-gate check above).
