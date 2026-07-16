<div align="center">

<h1 style="font-size: 3.5em;">Ballerina Nutcracker</h1>

**A native Ballerina interpreter written in Go, compiling to and executing Ballerina Intermediate Representation (BIR).**

</div>

[![Website](https://img.shields.io/badge/Website-ballerina.io-52C3C2)](https://ballerina.io/)
[![Release](https://img.shields.io/github/v/release/ballerina-nutcracker/ballerina)](https://github.com/ballerina-nutcracker/ballerina/releases)
[![Native CI](https://github.com/ballerina-nutcracker/ballerina/actions/workflows/native-ci.yml/badge.svg)](https://github.com/ballerina-nutcracker/ballerina/actions/workflows/native-ci.yml)
[![golangci-lint](https://github.com/ballerina-nutcracker/ballerina/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/ballerina-nutcracker/ballerina/actions/workflows/golangci-lint.yml)
[![codecov](https://codecov.io/gh/ballerina-nutcracker/ballerina/graph/badge.svg)](https://codecov.io/gh/ballerina-nutcracker/ballerina)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Code of Conduct](https://img.shields.io/badge/Code%20of%20Conduct-CNCF-4baaaa.svg)](CODE_OF_CONDUCT.md)
[![Discord](https://img.shields.io/badge/Discord-Ballerina-52C3C2?logo=discord&logoColor=white)](https://discord.gg/ballerinalang)
[![Stack Overflow](https://img.shields.io/stackexchange/stackoverflow/t/ballerina?logo=stackoverflow&label=Stack%20Overflow)](https://stackoverflow.com/questions/tagged/ballerina)
![X](https://img.shields.io/twitter/follow/ballerinalang?style=social&label=Follow%20Us)

[![Try it on Ballerina Playground](https://img.shields.io/badge/Try%20it-Ballerina%20Playground-52C3C2)](https://play.ballerina.io/)
Run and share Ballerina snippets in your browser — no installation needed.

## Table of contents

- [Ballerina Language](#ballerina-language)
- [Ballerina Nutcracker](#ballerina-nutcracker)
- [Scope & Roadmap](#scope--roadmap)
- [Architecture](#architecture)
- [Getting started](#getting-started)
- [Report issues](#report-issues)
- [Contribute](#contribute-to-ballerina-nutcracker)
- [License](#license)
- [Code of conduct](#code-of-conduct)
- [Join the community](#join-the-community)

## Ballerina Language

[Ballerina](https://ballerina.io) is an open-source, cloud-native programming language optimized for integration. It has built-in support for JSON and XML, first-class constructs for services and concurrency, and structural typing. It is developed and supported by [WSO2](https://wso2.com) and the wider Ballerina community.

## Ballerina Nutcracker

**Ballerina Nutcracker** is a native Ballerina interpreter written in Go. It compiles Ballerina source to **Ballerina Intermediate Representation (BIR)** and interprets the BIR directly, with a focus on speed, low memory use, and fast startup — properties suited to short-lived, cloud-native workloads (CLIs, functions, sidecars) where JVM warm-up cost is undesirable.

## Scope & Roadmap

Development is organized by **subsets** of the Ballerina language; each milestone adds support for a defined subset.

- **Progress:** [GitHub Milestones](https://github.com/ballerina-nutcracker/ballerina/milestones)
- **Subset docs:** [doc/](doc/) (language and standard library features and restrictions per subset)
- **Language spec:** [ballerina-platform/ballerina-spec](https://github.com/ballerina-platform/ballerina-spec)

## Architecture

A `.bal` program passes through a compilation pipeline (source → BIR) and is then executed by the BIR interpreter; both stages draw on the language and standard library:

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="doc/img/architecture-dark.png">
  <img alt="Ballerina Nutcracker architecture: the CLI and project/package management feed the compilation pipeline (parser, ast, semantics, semtypes, desugar, bir), which feeds the runtime (BIR interpreter); both the pipeline and the runtime draw on the Library (language library lang.* and the standard library); the runtime reaches the host only through the Platform Adaptation Layer; diagnostics, profiling, and corpus testing observe the whole system." src="doc/img/architecture-light.png">
</picture>

The compilation pipeline maps to source directories in stage order: [`parser/`](parser/) (lexing/parsing) → [`ast/`](ast/) → [`semantics/`](semantics/) (symbol/type resolution, semantic analysis, CFG) → [`desugar/`](desugar/) → [`bir/`](bir/) (BIR model, generation, codec).

The runtime takes over once the pipeline produces BIR. [`runtime/`](runtime/) and [`values/`](values/) hold the BIR interpreter and its runtime value representations.

[`semtypes/`](semtypes/), the structural type system, cuts across both of the above rather than sitting at one stage — it's used inside `semantics/` for type resolution, and again by `desugar/`, `bir/`, and `runtime/`+`values/`.

The **Library** is split in two. [`lib/langlibs/`](lib/langlibs/) is the language library (`lang.array`, `lang.map`, `lang.string`, …), required by every program. [`lib/stdlibs/`](lib/stdlibs/) is the standard library (`http`, `io`, `os`, `crypto`, …), made of optional capability modules. Both are declared in Ballerina and backed by native Go implementations, wired together by [`lib/rt`](lib/rt).

A few packages cut across every stage: [`projects/`](projects/) (manifest/package resolution), [`model/`](model/) (symbols, package/flag metadata), [`context/`](context/) (compiler context/environment shared across stages), and [`platform/pal/`](platform/pal/) (the Platform Adaptation Layer). All I/O, filesystem, and network access is routed through the PAL rather than calling the OS/Go stdlib directly.

Every component in the diagram is a Go package compiled into the single `bal` binary. The only real process/network boundaries are the Ballerina Central registry (package downloads) and the host OS, reached exclusively through the Platform Adaptation Layer.

See [AGENTS.md](AGENTS.md) for the exact stage list and the concurrency/error-handling rules between stages.

## Getting started

### Prerequisites

The project is built using the [Go programming language](https://go.dev/). The following dependencies are required:

- [Go 1.26 or later](https://go.dev/dl/)

### Build the CLI

```bash
go build -o bal ./cli/cmd
```

### Using the CLI

```bash
./bal --help
./bal run --help
```

#### Running a `.bal` source

Currently, the following are supported:

- Single `.bal` file
- Ballerina package with only the default module

E.g.

```bash
./bal run --dump-bir corpus/bal/subset1/01-boolean/equal1-v.bal
./bal run projects/testdata/myproject
```

For debugging flags, profiling, and running tests, see [doc/guides/DEVELOPING.md](doc/guides/DEVELOPING.md).

## Report issues

> **Tip:** If you are unsure whether you have found a bug, search the [existing issues](https://github.com/ballerina-nutcracker/ballerina/issues) in the GitHub repo and open an issue if needed.

### Open an issue

- [Open an issue](https://github.com/ballerina-nutcracker/ballerina/issues) for bug reports or feature requests related to Ballerina Nutcracker.

### Report security issues

- Send an email to [security@ballerina.io](mailto:security@ballerina.io). For details, see the [security policy](SECURITY.md).

## Contribute to Ballerina Nutcracker

As an open-source project, this repository welcomes contributions from the community. To start contributing, read the [contribution guidelines](CONTRIBUTING.md).

We look forward to receiving your contributions.

## License

This project is distributed under [Apache License 2.0](./LICENSE).

## Code of conduct

This project adheres to the [CNCF Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Join the community

- Get help on [Stack Overflow](https://stackoverflow.com/questions/tagged/ballerina)
- Discuss features, issues, and ideas in [GitHub Discussions](https://github.com/ballerina-nutcracker/ballerina/discussions)
- Join the conversations in the [Discord community](https://discord.gg/ballerinalang)
- For more details on how to engage with the community, see [Community](https://ballerina.io/community/)
