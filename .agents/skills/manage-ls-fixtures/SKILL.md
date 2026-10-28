---
name: manage-ls-fixtures
description: Create/update golden-JSON LSP fixture tests under ls/corpus and run the LS verification gates. Use when LS test scenarios must be added, updated, or re-goldened.
---

The format contract (fixture layout, JSON shape, driver behavior) lives in `ls/AGENTS.md` — read it first; this skill is the procedure, not the spec.

## Adding a scenario

1. Place fixtures per feature: `ls/corpus/<feature>/testdata/<scenario>.json` (transcript with `messages` + golden `expected`), with an optional `<scenario>.bal` named by its `source` field.
2. Prefer fixture tests over Go unit tests — if a scenario can't be driven through a real LSP request against a `.bal` fixture, question whether it can happen in the real world.
3. Keep scenarios narrow: one behavior per transcript, named for the behavior (`overlay-over-disk.json`), not numbered.

## Updating goldens

- Run `go test ./ls/corpus -run 'TestCorpus/<feature>/testdata/<scenario>.json' -update`, then `git diff` the goldens and **revert unrelated drift** — an update run must only change the scenarios you touched.
- Never hand-edit a golden to make a test pass; if the actual response is wrong, the fix is in the server.

## Gates — all three before declaring green

```
go test ./ls/...
go build ./...
go vet ./ls/...
```

## The driver

`TestCorpus` in `ls/corpus/corpus_test.go` is the in-process fake-transport driver: it sends each transcript's `messages` through real framing and dispatch and diffs the output against `expected`. Extend it only when a transcript cannot express the scenario.
