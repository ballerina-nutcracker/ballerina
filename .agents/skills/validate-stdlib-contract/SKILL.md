---
name: validate-stdlib-contract
description: Validate that a ballerina/<name> stdlib's Go public contract does not break the jBallerina public interface. Use when asked to validate, check, or verify a stdlib's public contract against jBallerina, or to review a stdlib PR for public-interface breakage. Produces a summary report at CONTRACT_VALIDATION.md. For porting a new stdlib use `add-stdlib-support`; for filling a gap use `fill-stdlib-gap`.
---

# Validating a Standard Library's Public Contract

This Go-native interpreter re-implements `ballerina/*` standard libraries that originated in jBallerina (Java). Customer Ballerina code is written against the **jBallerina public interface**.

**The golden rule:** the Go implementation must **never break that public interface** — doing so breaks existing customer code at compile time. This skill checks a single `ballerina/<name>` package against that rule and writes a readable, summary-level report.

It validates the **public interface only**, from **Ballerina sources only** (`.bal` files) — it does not audit Go native behaviour line by line. Run it per module; it is repeatable.

Use `add-stdlib-support` to port a new stdlib and `fill-stdlib-gap` to implement a missing function. This skill **only validates** — it does not change the implementation.

## The rule, precisely

The README support matrix is the **contract of record**. Its status column decides how each interface is treated:

- **Supported / Partially Supported** — interface must be present in the Go `.bal` with a signature matching jBallerina exactly. A limitation surfaces as a runtime error/warning, never a removed or altered declaration.
- **Cannot Support** — a *permanent* limitation, but the interface must **still be present** and degrade to a runtime error/warning. Removing it hands the customer a compile error — the exact break we forbid. It is **in scope**, not a coverage gap.
- **Not Yet Supported** — the only legitimately-absent bucket (roadmap/deferred). Counted as coverage, never failed.

Three facets, enforced over `Supported` + `Partially Supported` + `Cannot Support`:

1. **No removed interface** — every public symbol behind one of those rows exists in the Go `.bal`.
2. **No changed signature** — its Go signature matches jBallerina exactly (two-way compatible).
3. **No new interface** — no `public` symbol exists in the Go `.bal` that jBallerina lacks. (Backward-compatible additions may be reconsidered in future; for now, forbidden.)

## 1. Acquire inputs

Ask the user for the path to the corresponding jBallerina **library implementation root**, e.g. `~/github/ballerina-platform/module-ballerina-<name>/`. Do not proceed without it.

Then locate:

- **jBallerina public API** — `.bal` files under `<root>/ballerina/` (exclude `tests/` and `build/`).
- **Go implementation** — `lib/stdlibs/ballerina/<name>/0.0.1/go1.2/*.bal`.
- **Support matrix** — `lib/stdlibs/ballerina/<name>/0.0.1/go1.2/README.md`, the "Go Native Interpreter Support Status" table.

Stop and ask if any of these is missing. If the Go module does not exist at all, this is a porting task — redirect to `add-stdlib-support`.

## 2. Extract both public surfaces (tool-driven)

Do **not** enumerate public symbols by reading `.bal` files by eye — that is slow and misses symbols on large surfaces. Use the extraction tool shipped with this skill, which parses each `.bal` tree with this repo's own parser and emits a sorted, stable dump of every `public` declaration with its caller-observable signature:

```shell
go run ./.agents/skills/validate-stdlib-contract/cmd/extract-surface <jball-root>/ballerina > /tmp/<name>-jball.surface
go run ./.agents/skills/validate-stdlib-contract/cmd/extract-surface lib/stdlibs/ballerina/<name>/0.0.1/go1.2 > /tmp/<name>-go.surface
```

(Run from this repo's root. The tool skips `tests/` and `build/` subdirectories automatically and reports files it failed to parse on stderr — a parse failure means that file must be reviewed by hand; do not silently drop it.)

The dump covers, per symbol kind:

- **Functions** — name, parameter names + types + defaults, rest parameter, return type (including the error union).
- **Types / records** — field names, types, optionality, `readonly`, defaults, and any inclusions.
- **Classes / objects** — public methods (full signatures) and public fields, plus `client`/`service`/`readonly`/`isolated` qualifiers.
- **Enums / consts** — members and values. Also annotations, listeners, and public module-level variables.

Private declarations and `= external` plumbing are excluded — they are not part of the contract.

Then diff the two dumps:

```shell
diff -u /tmp/<name>-jball.surface /tmp/<name>-go.surface
```

An empty diff over the in-scope symbols is a strong PASS signal; every diff hunk must be classified in Step 5. Spot-check a couple of symbols against the raw `.bal` source to confirm the tool's output is faithful before relying on it.

## 3. Determine validation scope from the support matrix

Read the README support table. Map each row whose status is **Supported**, **Partially Supported**, or **Cannot Support** (the Feature/API column is prose, per `stdlib-readme-format` — interpret it to the concrete public symbols it covers) into the **in-scope contract**. Rows marked **Not Yet Supported** are out of scope (deferred coverage).

If a matrix row's prose can't be mapped to concrete symbols with confidence, note it as a 🟡 documentation gap rather than guessing.

## 4. Compare and classify

Walk the Step 2 diff, bounded by the in-scope set (Step 3). Classify every symbol:

- ✅ **Compatible** — in-scope (`Supported`/`Partially`), present in Go, signature matches jBallerina.
- 🔵 **Gracefully degraded** — present in Go with a matching signature, but the implementation degrades to a documented **runtime error/warning**. Expected for `Cannot Support` rows and limited `Partially Supported` cases. **Acceptable** (does not fail) *provided* the degradation is documented in the README (matrix Comments / Notable Behavioural Changes) and is a runtime error/warning — not a compile error, not silent wrong behaviour.
- 🔴 **Breaking — signature diverges** — in-scope, present, but the signature differs (param name/type/order/default, rest param, return type). Record the jBallerina and Go signatures side by side.
- 🔴 **Breaking — missing** — matrix claims `Supported` / `Partially` / `Cannot Support` but the symbol is absent from the Go `.bal` (customer hits a compile error). For `Cannot Support`, the fix is to re-add the declaration backed by a runtime error/warning — not to leave it out.
- 🔴 **Disallowed — new interface** — `public` in Go, absent in jBallerina.
- 🟡 **Matrix inaccurate / undocumented degradation** — present in Go but not reflected (or mis-stated) in the matrix; marked `Supported` while clearly partial; or a `Cannot Support`/limited symbol whose runtime-error degradation is undocumented. A documentation fix, not a hard fail unless it hides one of the 🔴 cases above.
- ⚪ **Deferred (coverage gap)** — a jBallerina public symbol with no Go counterpart, documented `Not Yet Supported`. Informational only.

### Comparison rules

- **Parameter names matter** — Ballerina supports named arguments, so renaming a public parameter **is breaking**.
- Optional / defaulted parameters and rest parameters are part of the signature.
- The return type, including the error union, must match.
- For records: field name, type, optionality, `readonly`, defaults, and inclusions all count.
- For classes/objects: public methods + fields, and `client`/`service`/`readonly` qualifiers that affect the caller.
- Qualifier differences a caller cannot observe (e.g. an `isolated` that doesn't change the call site) are **notes**, not breaks.
- The tool dumps signatures as written in source; a purely textual difference that is semantically identical (e.g. whitespace inside a union, `int[] ` vs `int[]`) is **not** a break — confirm against the raw source before recording a 🔴.

## 5. Cross-check accepted divergences

Read `AGENTS.md` for known interpreter limitations (`distinct` error subtypes aliased to `error`, `readonly &` intersections, `stream`, XML, full `typedesc` handling). A signature divergence wholly attributable to one of these is **🔵 Accepted (interpreter limitation)** — surfaced, not failed — *provided* it is recorded in the README **Notable Behavioural Changes**. If it is not documented there, downgrade it to 🟡 (documentation gap).

So 🔵 covers both interpreter-limitation divergences and `Cannot Support` runtime-error degradations: both are acceptable only when documented.

## 6. Write the report

Write `lib/stdlibs/ballerina/<name>/0.0.1/go1.2/CONTRACT_VALIDATION.md`, summary-first, problems up top, not technically deep.

**Lifecycle:** this report is an ephemeral review artifact — regenerated in full on every run, never committed (`CONTRACT_VALIDATION.md` is gitignored). Overwrite any existing copy without preserving its content; the git-tracked contract of record remains the README support matrix.

```markdown
# Public Contract Validation — ballerina/<name>

## Verdict
**PASS / FAIL** — one line (e.g. "FAIL — 1 signature break, 1 new interface").

## Scope
- jBallerina reference: <path> (commit/version if known)
- Go implementation: lib/stdlibs/ballerina/<name>/0.0.1/go1.2/
- Compared: public functions, types, constants, enums, classes, annotations, listeners
- Enforcement scope: Supported / Partially Supported / Cannot Support rows (presence + signature). Not Yet Supported is out of scope.

## Summary
| Category                                      | Count |
|-----------------------------------------------|-------|
| In-scope public symbols (validated)           |       |
| ✅ Compatible                                  |       |
| 🔵 Gracefully degraded (runtime error/warning) |       |
| 🔴 Breaking — signature diverges               |       |
| 🔴 Breaking — missing (claimed)                |       |
| 🔴 Disallowed — new interface in Go            |       |
| 🟡 Matrix inaccurate / undocumented            |       |
| ⚪ Deferred (Not Yet Supported)                 |       |

## Coverage
- jBallerina public surface: N symbols total
- In scope — enforced (Supported + Partially + Cannot Support): X  (XX%)
- Deferred (Not Yet Supported): Y — list briefly

## 🔴 Violations of the golden rule
### Removed / missing public interface
### Changed signature (jBallerina → Go, side by side)
### New public interface in Go (not in jBallerina)

## 🔵 Accepted divergences & graceful degradations
- symbol — what differs or how it degrades (runtime error/warning) — where it's documented

## 🟡 Documentation gaps
- matrix rows that over- or under-state the real surface

## Recommendations
- per violation: the concrete fix — re-add the declaration backed by a runtime error/warning;
  revert the signature to the jBallerina shape; remove or gate the new public symbol;
  correct the matrix row.
```

Keep each section short. Omit a 🔴/🟡/🔵 section entirely if it has no entries (but always keep Verdict, Summary, and Coverage).

## 7. Verdict

- **FAIL** if any 🔴 exists.
- **PASS (with notes)** if only 🟡 / 🔵 / ⚪ exist.
- **PASS** if everything is ✅.

State the verdict on the first line of the report and repeat it in the final chat summary, with the headline counts (e.g. "FAIL — 1 missing Cannot-Support interface, 1 signature break; 3 documentation gaps").
