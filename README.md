<div align="center">

<img alt="Ballerina Nutcracker" src="https://raw.githubusercontent.com/ballerina-nutcracker/ballerina/main/doc/img/logo-light.svg#gh-light-mode-only" width="300">
<img alt="Ballerina Nutcracker" src="https://raw.githubusercontent.com/ballerina-nutcracker/ballerina/main/doc/img/logo-dark.svg#gh-dark-mode-only" width="300">

**A native interpreter for the Ballerina programming language.**

[![Release](https://img.shields.io/github/v/release/ballerina-nutcracker/ballerina)](https://github.com/ballerina-nutcracker/ballerina/releases)
[![Native CI](https://github.com/ballerina-nutcracker/ballerina/actions/workflows/native-ci.yml/badge.svg)](https://github.com/ballerina-nutcracker/ballerina/actions/workflows/native-ci.yml)
[![codecov](https://codecov.io/gh/ballerina-nutcracker/ballerina/graph/badge.svg)](https://codecov.io/gh/ballerina-nutcracker/ballerina)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Discord](https://img.shields.io/badge/Discord-Ballerina-52C3C2?logo=discord&logoColor=white)](https://discord.gg/ballerinalang)

[Website](https://ballerina.io) &nbsp;·&nbsp;
[Playground](https://play.ballerina.io/) &nbsp;·&nbsp;
[Language spec](https://github.com/ballerina-platform/ballerina-spec) &nbsp;·&nbsp;
[Developing](doc/guides/DEVELOPING.md) &nbsp;·&nbsp;
[Architecture](doc/guides/ARCHITECTURE.md) &nbsp;·&nbsp;
[Roadmap](https://github.com/ballerina-nutcracker/ballerina/milestones)

</div>

---

[Ballerina](https://ballerina.io) is an open-source, cloud-native programming language optimized for integration, with built-in support for JSON and XML, first-class constructs for services and concurrency, and structural typing. It is developed and supported by [WSO2](https://wso2.com) and the wider Ballerina community. Try the language in your browser on the [Ballerina Playground](https://play.ballerina.io/).

**Ballerina Nutcracker** compiles Ballerina source to **Ballerina Intermediate Representation (BIR)** and interprets the BIR directly. Written in Go, it ships as one `bal` binary — no separate runtime, nothing to warm up — which keeps startup fast and the footprint small for short-lived cloud-native workloads such as CLIs, functions, and sidecars.

> [!IMPORTANT]
> Nutcracker is under active development and does not yet support the whole language. For production use today, reach for [Ballerina Swan Lake](https://ballerina.io/downloads/) — the official distribution, which supports the full language.

## Architecture

![Ballerina Nutcracker architecture: the bal CLI (new, run, pack, build, push, version) is the entry point. parser/ produces st/; nodebuilder/ produces ast/. semantics/ resolves types; desugar/ and birgen/ lower to BIR. The runtime interprets BIR. Native stdlib uses extern calls; pure-Ballerina modules run as BIR. PAL is platform/pal; palnative is on the host OS and pal_wasm.go on the browser. The central cache is the on-disk default for dependency resolution; bal push writes the local repository.](doc/img/architecture.png)

Almost everything that ships in the `bal` binary is a Go package. The central cache, the local repository, the host OS, and the browser sit outside it. See [ARCHITECTURE.md](doc/guides/ARCHITECTURE.md) for how the diagram maps onto source directories.

## Getting started

Download a binary from the [latest release](https://github.com/ballerina-nutcracker/ballerina/releases), or build from source with [Go 1.27 or later](https://go.dev/dl/):

```bash
git clone https://github.com/ballerina-nutcracker/ballerina.git
cd ballerina
go build -o bal ./cli/cmd
```

Create and run your first program:

```bash
./bal new hello
./bal run hello
```

```ballerina
import ballerina/io;

public function main() {
    io:println("Hello, World!");
}
```

| Command | Description |
| --- | --- |
| `bal new <path>` | Create a new package (`-t <template>`, `--workspace`) |
| `bal run <file.bal> \| <package-dir> \| .` | Compile and execute a source file or package |
| `bal pack [<package-dir>]` | Build the `.bala` distribution archive of a package |
| `bal build [<package-dir>]` | Build a standalone executable that bundles the Ballerina runtime |
| `bal push [<bala-path>] --repository=local` | Push a `.bala` of the current package (or a given archive) to the local repository |
| `bal version` | Print the version |

`bal build` needs a `balrt` stripped-down runtime alongside `bal` (`go build -o balrt ./cli/internal/balrt`), or pointed at via the `main.RuntimeStubPath` link-time override.

For debugging flags, profiling, testing, and linting, see [DEVELOPING.md](doc/guides/DEVELOPING.md).

## Scope & roadmap

Development is organized by **subsets** of the Ballerina language; each milestone adds support for a defined subset.

- **Progress:** [GitHub Milestones](https://github.com/ballerina-nutcracker/ballerina/milestones)
- **Supported language features:** [`doc/lang`](doc/lang)
- **Supported library features:** [`doc/library`](doc/library)

## Standard library coverage

**73%** of tracked standard library features are supported across 18 packages: 280 supported, 31 partially supported, 71 not yet supported. See [Standard library support](doc/library/README.md) for per-package coverage and behavioural changes from jBallerina.

**Supported packages:** [avro](lib/stdlibs/ballerina/avro/0.0.1/go1.27/README.md), [crypto](lib/stdlibs/ballerina/crypto/0.0.1/go1.27/README.md), [file](lib/stdlibs/ballerina/file/0.0.1/go1.27/README.md), [http](lib/stdlibs/ballerina/http/0.0.1/go1.27/README.md), [io](lib/stdlibs/ballerina/io/0.0.1/go1.27/README.md), [ldap](lib/stdlibs/ballerina/ldap/0.0.1/go1.27/README.md), [log](lib/stdlibs/ballerina/log/0.0.1/go1.27/README.md), [math.vector](lib/stdlibs/ballerina/math.vector/0.0.1/go1.27/README.md), [mime](lib/stdlibs/ballerina/mime/0.0.1/go1.27/README.md), [mqtt](lib/stdlibs/ballerina/mqtt/0.0.1/go1.27/README.md), [os](lib/stdlibs/ballerina/os/0.0.1/go1.27/README.md), [protobuf](lib/stdlibs/ballerina/protobuf/0.0.1/go1.27/README.md), [random](lib/stdlibs/ballerina/random/0.0.1/go1.27/README.md), [task](lib/stdlibs/ballerina/task/0.0.1/go1.27/README.md), [time](lib/stdlibs/ballerina/time/0.0.1/go1.27/README.md), [udp](lib/stdlibs/ballerina/udp/0.0.1/go1.27/README.md), [url](lib/stdlibs/ballerina/url/0.0.1/go1.27/README.md), [uuid](lib/stdlibs/ballerina/uuid/0.0.1/go1.27/README.md)

## Lang library coverage

**47%** of jBallerina lang library functions are supported across 19 modules: 107 supported, 0 partially supported, 123 not yet supported. Functions handled directly by the compiler count as supported. See [Lang library support](doc/langlib/README.md) for per-module coverage.

**Supported modules:** [lang.array](lib/langlibs/ballerina/lang.array/0.0.1/any/README.md), [lang.boolean](lib/langlibs/ballerina/lang.boolean/0.0.1/any/README.md), [lang.decimal](lib/langlibs/ballerina/lang.decimal/0.0.1/any/README.md), [lang.error](lib/langlibs/ballerina/lang.error/0.0.1/any/README.md), [lang.float](lib/langlibs/ballerina/lang.float/0.0.1/any/README.md), [lang.int](lib/langlibs/ballerina/lang.int/0.0.1/any/README.md), [lang.map](lib/langlibs/ballerina/lang.map/0.0.1/any/README.md), [lang.runtime](lib/langlibs/ballerina/lang.runtime/0.0.1/any/README.md), [lang.string](lib/langlibs/ballerina/lang.string/0.0.1/any/README.md), [lang.value](lib/langlibs/ballerina/lang.value/0.0.1/any/README.md), [lang.xml](lib/langlibs/ballerina/lang.xml/0.0.1/any/README.md)

## Contributing

Contributions are welcome — read the [contribution guidelines](CONTRIBUTING.md) to get started, and the [code of conduct](CODE_OF_CONDUCT.md) before you take part.

- **Found a bug or want a feature?** Search [existing issues](https://github.com/ballerina-nutcracker/ballerina/issues), then open a new one.
- **New here?** Start with issues labelled [`good first issue`](https://github.com/ballerina-nutcracker/ballerina/labels/good%20first%20issue).
- **Found a security vulnerability?** Do not open an issue. Email [security@ballerina.io](mailto:security@ballerina.io) — see the [security policy](SECURITY.md).

## Community

Questions and ideas belong in [GitHub Discussions](https://github.com/ballerina-nutcracker/ballerina/discussions) or on [Discord](https://discord.gg/ballerinalang) — Discord is where the team is most active. See [ballerina.io/community](https://ballerina.io/community/) for everything else.

## License

Distributed under the [Apache License 2.0](LICENSE).
