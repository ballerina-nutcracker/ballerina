---
name: manage-ls-fixtures
description: Create/update golden-JSON LSP fixture tests under ls/corpus and run the LS verification gates. Use during the verify stage of an ls-iteration, or standalone when the user wants LS test scenarios added, updated, or re-goldened.
---

The format contract (fixture layout, JSON shape, driver behavior) lives in `ls/AGENTS.md` — read it first; this skill is the procedure, not the spec.

## Adding a scenario

1. Place fixtures per feature: `ls/corpus/<feature>/testdata/<scenario>.bal` + `<scenario>.<method>.json` (request params + golden expected response).
2. Prefer fixture tests over Go unit tests — if a scenario can't be driven through a real LSP request against a `.bal` fixture, question whether it can happen in the real world.
3. Keep scenarios narrow: one behavior per fixture pair, named for the behavior (`record_field.hover.json`), not numbered.

## Updating goldens

- Run `go test ./ls/corpus --update`, then `git diff` the goldens and **revert unrelated drift** — an update run must only change the scenarios the iteration touched.
- Never hand-edit a golden to make a test pass; if the actual response is wrong, the fix is in the server.

## Gates — all three before declaring green

```
go test ./ls/...
go build ./...
go vet ./ls/...
```

## The driver

The in-process fake-transport driver (start the LS, feed the fixture as a real JSON-RPC request — real framing, real dispatch — diff against `expected`) **may not exist yet**. If it doesn't, stop: it must be built as an explicit deliverable of an iteration's approved design (see the `ls-iteration` skill, stage 4), not improvised here.
