---
name: stdlib-readme-format
description: Authoritative format contract for `lib/stdlibs/ballerina/<name>/0.0.1/go1.26/README.md` files. Use when creating or updating any stdlib README, or when auditing an existing one for consistency.
---

# stdlib README Format

This skill defines the exact structure and rules for every `lib/stdlibs/ballerina/<name>/0.0.1/go1.26/README.md`. It can be invoked standalone to audit or fix an existing README, or embedded in another workflow (e.g. `add-stdlib-support`, `fill-stdlib-gap`) when writing a new one.

## Template

Use this skeleton exactly. Do not add, remove, or reorder sections. One permitted extension: a large module may group its support tables under `###` subsections inside **Ballerina Nutcracker Support Status** (e.g. http's `### Client` / `### Request` / `### Response`), each subsection holding its own three-column table.

````markdown
# Ballerina <Name> Library

## Overview

<Brief description of the full jBallerina module scope, ending with one sentence stating which subset Ballerina Nutcracker currently supports.>

## Key Functionalities

<Bullet list of what the Go-native version currently supports — not the full jBallerina feature set.>

## Examples

```ballerina
<Short working example using only currently supported APIs.>
```

## Ballerina Nutcracker Support Status

This library is currently being migrated to Go to support Ballerina Nutcracker. The table below outlines the current support level for various features of this library in the Go implementation.

Support Levels:

- **Supported**: Fully implemented and tested in the Go version.
- **Partially Supported**: Implemented but lacking some edge cases, options, or sub-features. (See comments).
- **Not Yet Supported**: Planned for migration, but not yet implemented.
- **Cannot Support**: Cannot be implemented in the Go version due to technical limitations or architectural differences. (See comments).

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| ... | ... | ... |

### Notable Behavioural Changes

<Use bullet points with bold headers, one bullet per divergence. Format each as:
- **<Short title>.** <jBallerina behaviour>; the Go-native version <Go-native behaviour> — <reason if helpful>.

If there are no notable behavioural changes, write:
There are **no** notable behavioural changes in the Go-native version compared to the original jBallerina implementation for the currently supported features.>
````

## Column rules

### Feature/API column

- **Prose only.** No backtick function names, type names, or object names anywhere in this column — not even in parentheses. Wrong: `"File read — string (\`fileReadString\`)"`. Right: `"File read — string"`.
- Function and type names belong in the **Comments / Limitations** column only.

### Support Status column

Exactly one of: `Supported`, `Partially Supported`, `Not Yet Supported`, `Cannot Support`.

### Comments / Limitations column

- **Supported rows with no caveat** — leave this cell empty. Do not write "Fully implemented and tested in the Go version." — that is implied by the status.
- **Supported rows with a caveat** — write only the caveat. Function names, type names, and signatures are allowed here.
- **Partially Supported / Not Yet Supported / Cannot Support** — explain the gap. Include relevant function or type names here.

### Table separator

Always `|---|---|---|`. Never wide-padded column separators.

## Notable Behavioural Changes rules

- **Format**: bullet list with a bold header followed by a period, then a sentence. Example: `- **Title.** jBallerina does X; the Go-native version does Y — reason.`
- **Content**: only permanent, architectural Go-level constraints that cannot be resolved in the `native/` layer.
- **Do not include**:
  - Temporary language gaps that will be fixed when the interpreter gains the feature (e.g. `distinct` error subtypes, `readonly &` intersections, `stream` type, XML, full `typedesc` parameter handling). These belong in the support table as `Not Yet Supported` or `Partially Supported`.
  - Entries that say "identical" or "matching" — if the behaviour is identical, it is not a change.
  - Future potential divergences for features that are `Not Yet Supported` — document those in the Comments column of the relevant table row instead.
- If there are no permanent changes, write the "no changes" sentence from the template rather than omitting the section.

## Validation

### Mechanical checks — run the script

From the repo root:

```shell
python3 .agents/skills/stdlib-readme-format/scripts/check_readmes.py
```

It validates every per-package README **and** the aggregator in one pass: required sections and their order, status values, `|---|---|---|` separators, no backticks in Feature/API cells, no "Fully implemented and tested" filler in Supported rows, non-empty Comments on every gap row, bullet format, bullets-vs-"no changes"-sentence consistency, aggregator counts/percentages/Total footer recomputed from the per-package tables, verbatim bullet mirroring, and the closing "no changes" sentence membership. **It must exit 0 before you save** — fix every `FAIL` line it prints.

### Judgment checks — verify by hand

The script cannot check these; confirm each is YES:

- [ ] Every `Supported` row's Comments cell is either empty or a *meaningful* caveat
- [ ] If the module declares a module-level error type (e.g. `io:Error`), the table has a row tracking it with an accurate status — `Partially Supported` with a `distinct` comment when the type ships as a plain alias, `Not Yet Supported` when it isn't declared at all
- [ ] No bullet describes behaviour identical to jBallerina
- [ ] No bullet describes a temporary language gap (`distinct`, `readonly &`, `stream`, etc.) — those belong in the support table
- [ ] No bullet describes a future feature's potential divergence
- [ ] Key Functionalities reflects only currently supported features
- [ ] Examples use only currently supported APIs
- [ ] No `Not Yet Supported` row that was just implemented in this session

## Canonical exemplars

These existing READMEs already conform — useful for cross-reference when in doubt:

- `lib/stdlibs/ballerina/io/0.0.1/go1.26/README.md` — multi-section coverage (print, file I/O, channels), one **Notable Behavioural Change** (`fileWriteJson` key ordering).
- `lib/stdlibs/ballerina/time/0.0.1/go1.26/README.md` — parity-heavy library with multiple documented divergences.
- `lib/stdlibs/ballerina/url/0.0.1/go1.26/README.md` — minimal stdlib README (good template for small surface).

## Top-level summary aggregator

The repo ships a top-level aggregator at `lib/stdlibs/ballerina/README.md` — a summary table of support percentages across stdlibs plus a consolidated **Notable Behavioural Changes** section grouped by package. This skill is responsible for keeping it in sync. **After every per-package README change, update the aggregator** as part of the same task; do not leave it stale.

Maintenance rules — after every per-package README change:

- Recount `Supported`, `Partially Supported`, and `Not Yet Supported` rows from the updated per-package `README.md`, and update that package's row in the aggregator table.
- Recompute support %: `round(Supported / Total * 100)` where `Total = Supported + Partially Supported + Not Yet Supported + Cannot Support`. **Note the asymmetry:** the aggregator table has no `Cannot Support` column, but `Cannot Support` rows still count in the % denominator — that's why a package with zero visible gaps can show less than 100% (e.g. file at 95%). Don't "fix" such a percentage without recounting the per-package table.
- Keep rows sorted alphabetically (no explicit dependency-level system exists in this repo); row format is `| [<name>](<name>/0.0.1/go1.26/README.md) | S | P | N | X% |`.
- Recompute the **Total** footer row (sum of each column; the % cell is `round(TotalSupported / TotalTotal * 100)`, where `TotalTotal` again includes the invisible `Cannot Support` rows).
- Mirror the package's **Notable Behavioural Changes** bullets verbatim into the matching `### <package>` subsection of the aggregator — copy the exact bullet text, don't paraphrase; the checker compares them word for word. Add a `### <package>` subsection when a package gains its first behavioural change; remove it (and add the package to the closing "no notable behavioural changes" sentence) when it has none.

When **adding a brand-new package**, add a new table row (alphabetical), recompute the Total footer, and add a `### <package>` subsection only if that package has notable behavioural changes.

The `check_readmes.py` script (see Validation above) verifies all of this arithmetic and mirroring — run it after every aggregator edit instead of trusting manual recounts.
