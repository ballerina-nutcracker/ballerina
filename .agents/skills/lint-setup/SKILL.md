---
name: lint-setup
description: Use when you need to understand and modify the current lint setup
---

# Lint setup

Use this skill for the Go lint setup in `ballerina-lang-go`. Paths and commands below are relative to the repository root. Before describing or changing the setup, read the current configs, resolve module `.golangci.yml` symlinks, and inspect the checker. Do not infer a module's tier from its name or directory alone.

> [!IMPORTANT]
> If you find any discrepancy between what is described here and the actual lint setup, always tell the user the difference and ask whether to update the skill.

- When the user asks you to add a lint to a module, ask whether to add it to every module in the same tier or to add a new tier. Never change the lint hierarchy unless explicitly told to.
- Don't use this document to describe each lint unless the user explicitly asks. Lint errors should explain what is wrong and how to fix it.

## Tier policy and module assignments

Each tier adds enabled linters to the one it extends:

```text
Basic → Strict ─┬→ Data
                └→ Frontend → Semantics
```

- **Data** adds the private-base visibility check and allows panics.
- **Frontend** prohibits panics.
- **Semantics** adds scoped globals checks, custom guards and import rules. It does not include Data's base-visibility check.
- **Strict** allows panics, except `decimal`, which keeps its existing panic ban through its own config.

A tier is a minimum: the policy checker allows extra enabled linters. When changing tiers:

- Do not disable an enabled lint to align tiers or silence findings.
- Keep a module's existing checks when promoting it.
- Do not demote a module or add suppressions to bypass its tier.

| Tier | Baseline and additional checks | Modules assigned directly to this tier | Config |
|---|---|---|---|
| Basic | `errcheck`, `govet`, `ineffassign`, `staticcheck`, `depguard`, `usetesting` | root (`.`), `cli`, `lib`, `platform`, `projects`, `tools`, `compiler-tools/benchmark`, `compiler-tools/benchmark-http`, `compiler-tools/tree-gen`, `compiler-tools/walkpurity` | `.golangci.yml` |
| Strict | `modernize`, `dupl`, `exhaustive`, `gochecksumtype`, `iface`, `iotamixing`, `reassign`, `unparam`, `wastedassign`, plus the workspace-wide `funcusage` check and the `shallow` analyzer; `decimal` also enables `forbidigo`, configured to prohibit `panic` | `semtypes`, `common`, `decimal`, `prettyprint`, `runtime`, `values` | `.golangci/strict.yml` plus `make lint-func-usage` and `make lint-shallow`; `decimal/.golangci.yml` for `decimal` |
| Data | Strict baseline plus `nodebase`: prohibit exported type names ending in `Base`; panics are allowed | `ast`, `st`, `bir`, `model` | `.golangci/data.yml` plus `make lint-node-bases` |
| Frontend | Strict baseline plus `forbidigo`, configured to prohibit `panic` | `birgen`, `desugar`, `nodebuilder`, `parser`, `context` | `.golangci/frontend.yml` |
| Semantics | Frontend baseline plus scoped `gochecknoglobals`, custom `forbidigo` rules, import allowlists, and the separate walk-purity check | `semantics` | `semantics/.golangci.yml` |

Sources: `.golangci.yml:3`, `.golangci/strict.yml:3`, `decimal/.golangci.yml:3`, `.golangci/data.yml:3`, `.golangci/frontend.yml:3`, `semantics/.golangci.yml:3`; restricted-tier assignments are encoded in `.github/scripts/check_lint_configs.py:34`. Basic modules are the remaining workspace modules using root-config fallback; verify against `go.work:5` and their actual symlinks.

The YAML files are separate complete configs, not YAML inheritance. Module-local `.golangci.yml` symlinks select shared configs; semantics and decimal have their own regular files. Data modules use `.golangci/data.yml`, retaining Strict's enabled lints and adding the private-AST-base diagnostic exclusion. They additionally run the base-visibility checker through Make and allow panics. The Makefile uses a module-local config when present and otherwise explicitly selects the root config (the `lint-module/%` target in `Makefile`).

## Shared settings and existing exceptions

- All configs use `default: standard` and explicitly enable `depguard` and `usetesting`. `depguard` prevents production imports of `test_util` and `test_util/testphases`, with exceptions for test files and files inside `test_util`; `usetesting.os-temp-dir` is enabled (`.golangci.yml:3`).
- `unused` is already disabled in all tiers. Staticcheck uses `all` except `ST1000`, `ST1003`, `ST1016`, `ST1020`, `ST1021`, and `ST1022`. Preserve these existing settings unless the user requests a change; they do not authorize disabling another lint (`.golangci.yml:23`).
- Strict and above use `reassign.patterns: [".*"]`, `exhaustive.default-signifies-exhaustive: true`, and `gochecksumtype.default-signifies-exhaustive: false` (`.golangci/strict.yml:20`).
- Data, frontend, and semantics exclude only `gochecksumtype` diagnostics for `BLangActionOrExpression` whose complete missing-types list consists of the six private AST bases: `bLangAccessExpressionBase`, `bLangActionBase`, `bLangValueExpressionBase`, `bLangVariableReferenceBase`, `bLangWorkerSendExprBase`, and `bLangWorkerSendReceiveExprBase`. The anchored diagnostic regex does not match a report containing any missing concrete implementation, an unlisted base, or another sum type. Preserve inherited markers and explicit concrete cases; do not broaden this into a switch/file suppression or a wildcard for every `Base` suffix. The effective-config regression tests exercise all three configs (`.golangci/data.yml:57`, `.golangci/frontend.yml:62`, `semantics/.golangci.yml:140`, `.github/scripts/test_check_lint_configs.py:77`).
- `gochecknoglobals` is enabled only in semantics, with its existing scoped exceptions (`semantics/.golangci.yml:18`). The shared Strict, Data, and Frontend configs do not enable it.

## Data-tier rules

`DATA` in `.github/scripts/check_lint_configs.py:37` assigns `ast`, `st`, `bir`, and `model` to this tier. Data requires the Strict baseline and allows panics; a Data config may still add a panic ban. `NO_PANIC` (frontend modules, semantics and `decimal`) requires the no-panic rule. Make obtains the Data module list with `check_lint_configs.py --data-modules`; update this assignment rather than maintaining a second list in Make (`DATA_MODULES` in `Makefile`).

`make lint-node-bases` runs `compiler-tools/nodebase` over those modules. It rejects every exported package-level type name ending in `Base`, including aliases and grouped declarations. It parses generated files, test files, and files for other build tags; the rule is independent of the current build platform (`compiler-tools/nodebase/main.go:57`, `compiler-tools/nodebase/main.go:90`).

The diagnostic is: `do not expose Base type <name>; keep base types private and expose concrete implementation types instead` (`compiler-tools/nodebase/main.go:35`). Make shared node bases private and expose concrete implementation types. Update generation templates when a generated reference changes. Preserve inherited expression markers on the shared bases and explicitly list concrete implementations in exhaustive switches; keep `ast.SetType` exhaustive. Do not replace inherited markers with per-implementation marker methods or export replacement base aliases to satisfy an external switch.

`make lint` and every Data module's `lint-module` target include this check. CI tests the checker with `go test ./compiler-tools/nodebase` before running `make lint` (the `lint` target in `Makefile`, `.github/workflows/golangci-lint.yml:24`).

## Semantics-specific rules

Read `semantics/.golangci.yml` before changing its policy. It additionally enables `gochecknoglobals`, with `forbidigo.analyze-types: true` and additional settings:

- Prohibit `panic` and `CompilerContext.SyntaxError` throughout the module (`semantics/.golangci.yml:29`).
- Require explicit traversal in non-test `internal/symbols` code by prohibiting `ast.Walk` and `ast.Visitor` there (`semantics/.golangci.yml:33`, `semantics/.golangci.yml:140`).
- Restrict analysis AST walking/pretty-print traversal, `runtime.Goexit`, selected symbol/type mutations, type-environment freezing, and diagnostic-state queries (`semantics/.golangci.yml:35`).
- Apply strict import allowlists to analysis production code, scope production code, and analysis tests (`semantics/.golangci.yml:68`).

Semantics preserves scoped exceptions: explicit path exclusions exempt its existing non-analysis packages and analysis tests from the analysis-specific rules and global-variable check. `NoFacts` and `AllFacts` have explicit global-variable exemptions. Do not describe its global-variable prohibition as module-wide (`semantics/.golangci.yml:136`).

`make lint-walks` separately runs `go run ./compiler-tools/walkpurity/cmd/walkpurity -test=false ./semantics/internal/types`; it is not a golangci-lint plugin (the `lint-walks` target in `Makefile`).

## Enforcement

1. **Config-policy gate:** `make lint-configs` runs `.github/scripts/check_lint_configs.py`. It checks strict, data, frontend, and semantics modules. It resolves configs, rejects broken symlinks, and uses `golangci-lint linters --json` to compare effective enabled sets with required minimums. This gate checks golangci-lint configuration; node-base visibility is checked separately against source. It does not impose the stronger baseline on Basic-tier modules (the `lint-configs` target in `Makefile`, `.github/scripts/check_lint_configs.py:109`).
2. **Effective-rule probes:** the checker copies each relevant config into a temporary Go module, sets `GOWORK=off`, and runs isolated restriction checks. Only modules in `NO_PANIC` are probed: frontend, semantics and `decimal` must report panics in production and tests. Semantics must also report production globals, preserving its test-global exception. Strict and Data are minimums, so an extra panic ban there is accepted. These probes catch ineffective or blanket-excluded rules; they do not exhaustively validate every semantics-specific rule (`.github/scripts/check_lint_configs.py:57`).
3. **Data base-visibility gate:** `make lint-node-bases` rejects exported base types in the modules assigned to `DATA`. It runs after config validation and before golangci-lint for those modules (the `lint-node-bases` target in `Makefile`).
4. **Function usage gate:** `make lint-func-usage` runs the function usage checker over every Strict-and-above module after config validation (`Makefile`, `compiler-tools/funcusage/main.go`).
5. **Shallow declaration gate:** `make lint-shallow` runs the `shallow` analyzer over every Strict-and-above module after config validation, once in the default build configuration and once with `-tags=debug,native_interp` added to `GOFLAGS` (`Makefile`, `compiler-tools/shallow/analyzer.go`).
6. **Module linting:** `make lint` discovers all workspace modules using `go list -m`, runs the config gate before module linting and the custom checks, and runs golangci-lint separately inside each module with `--allow-parallel-runners --concurrency 1`. `make -j4 lint` parallelizes the module jobs (`LIST_WORKSPACE_MODULES` and the `lint` target in `Makefile`).
7. **CI:** `.github/workflows/golangci-lint.yml` runs on pull requests and pushes to `main`, installs Go `1.27.1` and golangci-lint `v2.15.0`, runs the policy-checker, node-base-checker, function-usage-checker and shallow-analyzer tests, then `make -j4 lint` (`.github/workflows/golangci-lint.yml:1`). Verify the workflow before installing a version; these pins can change.
8. **Local commit hook:** when installed with `make install-hooks`, `.githooks/commit-msg` checks Go formatting and runs `make -j4 build lint` (the `install-hooks` target in `Makefile`, `.githooks/commit-msg:68`).

A successful config-policy check proves the required checks are active, not that the module's Go source passes them. Report config validation and source lint results separately. Do not record transient finding counts in this skill.

## Changing or validating the setup

Keep shared config contents, module symlinks, checker tier memberships/minimums, checker tests, custom-check targets, and this skill consistent when changing a tier or assignment. The checker allows extra enabled linters, so retain them when reconciling configs (`.github/scripts/check_lint_configs.py:127`).

Run from the repository root as appropriate to the change:

```sh
python3 .github/scripts/test_check_lint_configs.py
go test ./compiler-tools/nodebase
go test ./compiler-tools/funcusage
go test ./compiler-tools/shallow/...
make lint-configs
make lint-node-bases
make lint-func-usage
make lint-shallow
golangci-lint config verify --config .golangci/strict.yml
make lint-module/runtime
make -j4 lint
```

Use the actual changed config and module in the targeted commands. The checker tests cover required linter omissions, extra enabled linters, fallback and broken symlinks, and effective global/panic restrictions (`.github/scripts/test_check_lint_configs.py:11`). Fix valid source findings without weakening the tier or changing intended compiler/runtime behaviour.
