---
name: stdlib-readme-format
description: Authoritative format contract for `lib/stdlibs/ballerina/<name>/0.0.1/go1.27/README.md` files. Use when creating or updating any stdlib README, or when auditing an existing one for consistency.
---

# stdlib README Format

This skill defines the exact structure and rules for every `lib/stdlibs/ballerina/<name>/0.0.1/go1.27/README.md`. It can be invoked standalone to audit or fix an existing README, or embedded in another workflow (e.g. `add-stdlib-support`, `fill-stdlib-gap`) when writing a new one.

## Template

Use this skeleton exactly. Do not add, remove, or reorder sections. One permitted extension: a large module may group its support tables under `###` subsections inside **Go Native Interpreter Support Status** (e.g. http's `### Client` / `### Request` / `### Response`), each subsection holding its own three-column table.

````markdown
# Ballerina <Name> Library

## Overview

<Brief description of the full jBallerina module scope, ending with one sentence stating which subset the Go Native Interpreter currently supports.>

## Key Functionalities

<Bullet list of what the Go-native version currently supports — not the full jBallerina feature set.>

## Examples

```ballerina
<Short working example using only currently supported APIs.>
```

## Go Native Interpreter Support Status

This library is currently being migrated to Go to support the Ballerina Native Interpreter. The table below outlines the current support level for various features of this library in the Go implementation.

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
For a jBallerina bug fix, end the bullet with the upstream issue link: (<https://github.com/ballerina-platform/ballerina-library/issues/N>).

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
- **Content**: permanent divergences of two kinds:
  - Architectural Go-level constraints that cannot be resolved in the `native/` layer.
  - Fixes of jBallerina bugs, where the port deliberately implements the correct behaviour (see `add-stdlib-support` Step 5). New or edited entries link their `ballerina-platform/ballerina-library` issue.
- **Do not include**:
  - Temporary language gaps that will be fixed when the interpreter gains the feature (e.g. `distinct` error subtypes, `readonly &` intersections, `stream` type, XML, full `typedesc` parameter handling). These belong in the support table as `Not Yet Supported` or `Partially Supported`.
  - Entries that say "identical" or "matching" — if the behaviour is identical, it is not a change.
  - Future potential divergences for features that are `Not Yet Supported` — document those in the Comments column of the relevant table row instead.
- If there are no permanent changes, write the "no changes" sentence from the template rather than omitting the section.

## Validation

### Mechanical checks — run the script

From the repo root:

```shell
python3 .agents/skills/stdlib-readme-format/scripts/check_readmes.py --write
```

This command **modifies files**: it rewrites the generated regions of `doc/library/README.md` and the repo-root `README.md` from the per-package READMEs (see [Top-level summary aggregator](#top-level-summary-aggregator)), then validates. Without `--write` (or with `--check`) it only validates and changes nothing. It refuses to rewrite anything while a per-package README has a violation, because the counts would be wrong. It validates every per-package README: required sections and their order, status values, `|---|---|---|` separators, no backticks in Feature/API cells, no "Fully implemented and tested" filler in Supported rows, non-empty Comments on every gap row, bullet format, and bullets-vs-"no changes"-sentence consistency. It also checks that every generated region matches what the per-package READMEs produce. **It must exit 0 before you finish the task** — fix every `FAIL` line it prints, then run it again.

### Judgment checks — verify by hand

The script cannot check these; confirm each is YES:

- [ ] Every `Supported` row's Comments cell is either empty or a *meaningful* caveat
- [ ] If the module declares a module-level error type (e.g. `io:Error`), the table has a row tracking it with an accurate status — `Partially Supported` with a `distinct` comment when the type ships as a plain alias, `Not Yet Supported` when it isn't declared at all
- [ ] No bullet describes behaviour identical to jBallerina
- [ ] Every jBallerina bug-fix bullet this change adds or edits links its upstream issue (existing unlinked bullets are not retroactively failed)
- [ ] No Comments cell justifies a behaviour only with "as in jBallerina" when that behaviour is a known jBallerina bug
- [ ] No bullet describes a temporary language gap (`distinct`, `readonly &`, `stream`, etc.) — those belong in the support table
- [ ] No bullet describes a future feature's potential divergence
- [ ] Key Functionalities reflects only currently supported features
- [ ] Examples use only currently supported APIs
- [ ] No `Not Yet Supported` row that was just implemented in this session

## Canonical exemplars

These existing READMEs already conform — useful for cross-reference when in doubt:

- `lib/stdlibs/ballerina/io/0.0.1/go1.27/README.md` — multi-section coverage (print, file I/O, channels), one **Notable Behavioural Change** (`fileWriteJson` key ordering).
- `lib/stdlibs/ballerina/time/0.0.1/go1.27/README.md` — parity-heavy library with multiple documented divergences.
- `lib/stdlibs/ballerina/url/0.0.1/go1.27/README.md` — minimal stdlib README (good template for small surface).

## Top-level summary aggregator

The repo ships an aggregator at `doc/library/README.md` — a summary table of support percentages across stdlibs plus a consolidated **Notable Behavioural Changes** section grouped by package — and a **Standard library coverage** summary in the repo-root `README.md`. Both are derived from the per-package READMEs. **After every per-package README change, regenerate them** as part of the same task; do not leave them stale and do not edit the generated parts by hand:

```shell
python3 .agents/skills/stdlib-readme-format/scripts/check_readmes.py --write
```

It rewrites only the text between each `<!-- BEGIN <region> … -->` line and its matching `<!-- END <region> -->` line:

- `stdlib-coverage-table` — the aggregator's coverage table: one row per package, sorted alphabetically, and the **Total** footer. Support % is `round(Supported / Total * 100)` where `Total = Supported + Partially Supported + Not Yet Supported + Cannot Support`. **Note the asymmetry:** the table has no `Cannot Support` column, but `Cannot Support` rows still count in the % denominator — that's why a package with zero visible gaps can show less than 100%.
- `stdlib-behavioural-changes` — the aggregator's `### <package>` subsections, each copying that package's **Notable Behavioural Changes** bullets verbatim (wrapped lines and indented sub-bullets included), and the closing sentence listing the packages that have none.
- `stdlib-coverage-summary` — the repo-root summary sentence (`**X%** of tracked standard library features are supported across N packages: …`) and the `**Supported packages:**` list.

Everything outside the markers (the introductions above the table and above the subsections, and any later sections) is hand-written and never touched. Keep the marker lines; the script reports a region whose markers are missing or duplicated. A brand-new package needs no aggregator edit: once its README exists, `--write` adds its row, its subsection or closing-sentence entry, and its link.

Stale totals are the usual failure after a rebase onto a `main` that changed another package: the merge has no textual conflict, but the totals no longer add up. The repository's pre-push hook (`make install-hooks`) runs the check on pushed commits that touch these files; when it fails, run `--write` and amend.
