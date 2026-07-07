---
name: fill-stdlib-gap
description: Fill a gap in an existing ballerina/<name> stdlib — implement a function marked Not Yet Supported, promote a Partially Supported row, or fix a behavioural divergence. Use when the target stdlib already exists under `lib/stdlibs/ballerina/`. For brand-new stdlibs, use `add-stdlib-support`.
---

# Filling a Gap in an Existing Standard Library

Lightweight 5-step workflow for adding a missing function, promoting a `Not Yet Supported` row to `Supported`, or fixing a divergence. Unlike `add-stdlib-support`, this skill has **no plan-approval gate** and **no library-evaluation gate** — the stdlib already exists, its file layout and wire-up are already in place, and the surface change is small.

If the user wants to port a brand-new stdlib (`lib/stdlibs/ballerina/<name>/` does not exist), use `add-stdlib-support` instead.

Coding rules and the PAL constraint live in `AGENTS.md` at the repo root — read it before editing.

## 1. Identify the gap

Open the target stdlib's README, e.g. `lib/stdlibs/ballerina/io/0.0.1/go1.26/README.md`, and confirm the row to be promoted.

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

Look for the same hot-spots as in `add-stdlib-support` Step 5, scoped to just this surface:

- Decimal precision, UTF-8 vs UTF-16 string ops, NaN/overflow, error message wording on the outer error.
- Domain-specific risks for the module (see exemplars in `add-stdlib-support` Step 5).

Rules:
- **Avoidable** divergences — fix during Step 4.
- **Unavoidable** divergences — record in the README under **Notable Behavioural Changes** during Step 5.

If every row is "No risk identified", say so and move on.

## 4. Implement

You are editing existing files, **not** creating new ones. In particular:

- **Do not** create new manifest files (`Ballerina.toml`, `Bala.toml`, `Dependencies.toml` already exist).
- **Do not** modify `lib/rt/libs.go` — the blank import is already there.
- **Do not** modify `test_util/testphases/phases.go` — the `builtinStdlibs` entry is already there.

What you *do* edit:

- **`lib/stdlibs/ballerina/<name>/0.0.1/go1.26/<name>.bal`** (or sibling `.bal` files like `file.bal`, `types.bal`) — add the public function, type declaration, or extern signature. Preserve the existing license header and doc-comment style. Function names match jBallerina exactly.
- **`lib/stdlibs/ballerina/<name>/0.0.1/go1.26/native/<name>.go`** (or sibling `.go` files like `file_io.go`) — add the Go implementation. Register it in the existing `init<Name>Module` function:
  ```go
  func init<Name>Module(rt *runtime.Runtime) {
      // existing registrations...
      runtime.RegisterExternFunction(rt, orgName, moduleName, "externNewFn", externNewFnExtern(rt))
  }
  ```
  If the new logic is large enough to warrant a new file, create `native/<feature>.go` alongside the existing ones — keep `package native` and reuse the `orgName` / `moduleName` constants already defined.

### PAL hookup (only if needed)

If the new function performs a platform op (io, fs, time, http, env) not already covered by the PAL, add it across three files — same as `add-stdlib-support`:

1. `platform/pal/platform.go` — add the field.
2. `platform/palnative/<category>.go` — implement for CLI.
3. `test_util/test_util.go` → `TestPal` — wire in (start from `palnative.NewNative<Category>PAL()` and override only test-specific fields).

Failing to update `TestPal` causes nil-pointer dereferences in corpus tests even when the CLI run succeeds.

### Coding rules (full list in `AGENTS.md`)

- License header on every new file (existing files retain theirs).
- No per-line comments — extract named functions instead.
- No new public symbols unless required by the public API.

## 5. Test, document, and verify

### Tests

Add corpus tests under the existing `corpus/bal/subset8/<NN>-<name>/` directory for this stdlib. If no directory exists for this stdlib yet, pick the next free `<NN>` prefix. Suffixes per `AGENTS.md`: `*-v.bal` (valid), `*-e.bal` (compile errors), `*-p.bal` (panics). No leading zeros in numeric parts.

Cover the new behaviour from `.bal` — a corpus test exercises the full compiler → BIR → interpreter pipeline and is measured by the native-coverage harness (`-coverpkg=./lib/stdlibs/...` over `./corpus/...`). Add a Go unit test only for branches genuinely unreachable from Ballerina (defensive type/arity guards, nil guards, interface-contract paths), kept minimal with a comment explaining why. Don't write wrong-type extern arg guards — the type checker rejects wrong types at compile time. See the **`manage-corpus-tests`** "Test philosophy" section. If you find existing native code that can never execute through Ballerina, remove it rather than testing it.

Regenerate goldens via the **`update-corpus-tests`** skill:
```shell
go test ./corpus --update
```
Review `git diff corpus/` before committing, and revert any unrelated golden drift `--update` introduces (some stages have non-deterministic ordering).

### Documentation

Update the README row via the **`stdlib-readme-format`** skill:

- Promote the affected row's status (`Not Yet Supported` → `Supported`, or `Partially Supported` → `Supported` if the caveats are resolved).
- If the parity check in Step 3 surfaced an unavoidable divergence, add it to **Notable Behavioural Changes**.
- Update the top-level aggregator `lib/stdlibs/ballerina/README.md`: recount this package's row and recompute the **Total** footer; if a behavioural change was added or removed, mirror it into the package's `### <name>` subsection of the consolidated section.
- Re-run the full `stdlib-readme-format` validation checklist against the updated README (catches pre-existing violations too).

### Verify checklist

- [ ] `go build ./...` — no compilation errors.
- [ ] `go vet ./...` — no vet warnings.
- [ ] `go test ./corpus/...` — all corpus tests pass.
- [ ] `go run ./cli/cmd run <test>.bal` for the new corpus test(s) — output matches `@output` markers.
- [ ] README row status reflects what's now implemented.
- [ ] `lib/stdlibs/ballerina/README.md` aggregator updated (package row recounted, Total footer recomputed, behavioural changes mirrored if any changed).
- [ ] `stdlib-readme-format` validation checklist passes.
- [ ] Any unavoidable divergence is in **Notable Behavioural Changes**.
- [ ] PAL fields (if any added) implemented in `palnative/` and wired into `TestPal`.

### Final report

In one short paragraph: which row was promoted, what was added (function names, file paths), any divergences recorded.
