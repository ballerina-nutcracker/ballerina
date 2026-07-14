## Native Nutcracker

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
![X](https://img.shields.io/twitter/follow/ballerinalang?style=social&label=Follow%20Us)
[![stackoverflow](https://img.shields.io/badge/Get%20Support%20on%20Stack%20Overflow-ballerina-52C3C2)](https://stackoverflow.com/questions/tagged/ballerina)
[![Join the community on Discord](https://img.shields.io/badge/Join%20us%20on%20Discord-Ballerina-52C3C2)](https://discord.gg/ballerinalang)
[![codecov](https://codecov.io/gh/ballerina-platform/ballerina-lang-go/graph/badge.svg)](https://codecov.io/gh/ballerina-platform/ballerina-lang-go)

[Ballerina](https://ballerina.io) is an open-source, cloud-native programming language optimized for integration. It has built-in support for JSON and XML, first-class constructs for services and concurrency, and structural typing. It is developed and supported by WSO2.

**Try Ballerina in your browser:** [Ballerina Playground](https://play.ballerina.io/) — run and share snippets without installing anything.

## Goals

Ballerina Nutcracker is a **native Ballerina interpreter in Go**: compile Ballerina source to **Ballerina Intermediate Representation (BIR)** and interpret the BIR, with a focus on speed, low memory use, and fast startup. Development is organized by **subsets** of the language; each milestone adds support for a defined subset.

- **Progress:** [GitHub Milestones](https://github.com/ballerina-platform/ballerina-lang-go/milestones)
- **Subset docs:** [doc/](doc/) (language features and restrictions per subset)

## Usage

### Dependencies

The project is built using the [Go programming language](https://go.dev/). The following dependencies are required:

- [Go 1.24 or later](https://go.dev/dl/)

### Build the CLI

#### Production Build (default)

```bash
go build -o bal ./cli/cmd
```

#### Debug Build

```bash
go build -tags debug -o bal-debug ./cli/cmd
```

### Using Profiling

Profiling is only available in debug builds (compiled with `-tags debug`).

#### Enable Profiling

```bash
# Default profiling port (:6060)
./bal-debug run --prof corpus/bal/subset1/01-boolean/equal1-v.bal

# Custom port
./bal-debug run --prof --prof-addr=:8080 corpus/bal/subset1/01-boolean/equal1-v.bal
```

#### Access Profiling Data

- Web UI: http://localhost:6060/debug/pprof/
- CPU Profile: http://localhost:6060/debug/pprof/profile?seconds=30
- Heap Profile: http://localhost:6060/debug/pprof/heap
- Goroutines: http://localhost:6060/debug/pprof/goroutine

#### Analyze with pprof Tool

```bash
# CPU profiling (30 second sample)
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Heap profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# Interactive web UI
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile?seconds=30
```

### Using the CLI

#### CLI Help

```bash
./bal --help
```

```bash
./bal run --help
```

#### Running a bal source

Currently, the following are supported:

- Single .bal file
- Ballerina package with only the default module

E.g.

```bash
./bal run --dump-bir corpus/bal/subset1/01-boolean/equal1-v.bal
./bal run project-api-test/testdata/myproject
```

### Testing

To run the tests, use the following command:

```bash
go test ./...
```

## Report issues

> **Tip:** If you are unsure whether you have found a bug, search the [existing issues](https://github.com/ballerina-nutcracker/ballerina/issues) in the GitHub repo and open an issue if needed.

### Open an issue

- [Open an issue](https://github.com/ballerina-nutcracker/ballerina/issues) for bug reports or feature requests related to the native interpreter.

### Report security issues

- Send an email to [security@ballerina.io](mailto:security@ballerina.io). For details, see the [security policy](SECURITY.md).

## Contribute to Ballerina Nutcracker

As an open-source project, Ballerina Nutcracker welcomes contributions from the community. To start contributing, read the [contribution guidelines](CONTRIBUTING.md).

## License

Ballerina code is distributed under [Apache License 2.0](./LICENSE).

## Join the community

- Get help on [Stack Overflow](https://stackoverflow.com/questions/tagged/ballerina)
- Join the conversations in [Discord community](https://discord.gg/ballerinalang).
- For more details on how to engage with the community, see [Community](https://ballerina.io/community/).
