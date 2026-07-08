# Package Resolution Corpus

End-to-end CLI integration tests for package-resolution scenarios. Each
subdirectory is a self-contained scenario that exercises one resolution
behaviour through the `bal run` binary.

## Scenario pattern

```text
<scenario-name>/
  README.md          # one-liner describing the scenario
  project/           # Ballerina project passed to `bal run`
    Ballerina.toml
    main.bal
  bal_env/           # synthetic BAL_ENV root
    repositories/
      local/bala/...                    # local repository (may be absent)
      central.ballerina.io/bala/...     # central cache
  expected.txtar     # expected stdout and stderr fragments
```

`expected.txtar` contains `stdout` and `stderr` sections. The runner asserts
that the captured output **contains** each non-empty line from the respective
section (substring match after trimming whitespace). This allows for incidental
surrounding diagnostics without failing the test.

## Seeded scenarios

| Directory | What it tests |
|-----------|---------------|
| `local-repo-hit` | Pin to `repository = "local"` resolves from the local cache; central copy is ignored |
| `local-repo-miss-warns` | Local cache absent; resolver warns and falls back to central |

## Future scenarios (not yet seeded)

- `workspace-member-resolution` — workspace member packages resolved within the workspace
- `transitive-from-local` — transitive dependency of a local-pinned package
- `multi-version-resolution` — highest compatible version wins when multiple versions present
