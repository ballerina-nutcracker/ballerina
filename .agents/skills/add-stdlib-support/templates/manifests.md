# Manifest templates for a `ballerina/<name>` stdlib package

All three manifests live at `lib/stdlibs/ballerina/<name>/0.0.1/go1.2/`.

## `Ballerina.toml`

```toml
[package]
org     = "ballerina"
name    = "<name>"
version = "0.0.1"
```

## `Bala.toml`

```toml
[bala]
schema_version = "4"

[build]
ballerina_version      = ""
implementation_vendor  = "WSO2"
language_spec_version  = "2024R1"
platform               = "go1.2"

[[modules]]
name   = "<name>"
export = true
```

## `Dependencies.toml` — no cross-stdlib imports

```toml
[ballerina]
dependencies-toml-version = "2"

[[package]]
org     = "ballerina"
name    = "<name>"
version = "0.0.1"
```

## `Dependencies.toml` — when the `.bal` source imports other stdlibs

When the package imports one or more other stdlibs (e.g. `import ballerina/time;`), add one `[[package]]` entry per dependency, then list the deps inline on the package that needs them:

```toml
[ballerina]
dependencies-toml-version = "2"

[[package]]
org     = "ballerina"
name    = "<dep1>"
version = "0.0.1"

[[package]]
org     = "ballerina"
name    = "<dep2>"
version = "0.0.1"

[[package]]
org     = "ballerina"
name    = "<name>"
version = "0.0.1"
dependencies = [
    {org = "ballerina", name = "<dep1>"},
    {org = "ballerina", name = "<dep2>"}
]
```

## Why `Dependencies.toml` matters (read this — silent-failure trap)

`projects/package_resolution.go` runs a BFS over `pkg.Manifest().Dependencies()` to build the transitive dependency graph before compiling anything. If a stdlib dependency is not listed here, the project resolver never compiles it before this package, and every run of a `.bal` file that imports this package fails with `Unknown import: ballerina/<dep>`.

The `builtinStdlibs` ordering in `test_util/testphases/phases.go` handles the corpus test loader **separately** — a dependency must appear *before* its dependents in that list. Both mechanisms must be correct and kept in sync: `Dependencies.toml` for the full project resolver, `builtinStdlibs` ordering for corpus tests.
