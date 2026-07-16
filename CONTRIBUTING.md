# Contributing to Ballerina Nutcracker

ballerina-lang-go is a native Ballerina compiler frontend written in Go. It is licensed under the [Apache License](https://www.apache.org/licenses/LICENSE-2.0) and is part of the Ballerina ecosystem.

We appreciate your help!

- [Get started](#get-started)
- [Build the source code](#build-the-source-code)
- [Submit your contribution](#submit-your-contribution)
- [Propose changes](#propose-changes)

## Get started

- Read the [Ballerina Code of Conduct](CODE_OF_CONDUCT.md).
- Join the [Ballerina community](https://ballerina.io/community/).
- Submitting a bug is just as important as contributing code. [Report an issue](https://github.com/ballerina-platform/ballerina-lang-go/issues) in this repo.
- Start with GitHub issues labeled `good first issue`. Use comments on the issue to indicate that you will be working on it and get guidance.

## Build the source code

- Ensure you have [Go 1.24 or later](https://go.dev/dl/).
- Production build: `go build -o bal ./cli/cmd`
- Debug build (enables profiling): `go build -tags debug -o bal-debug ./cli/cmd`

See the [README](README.md) for more details on building, running corpus tests, and profiling.

## Submit your contribution

1. Make your changes in the source code.
2. Add or update tests as needed. Prefer [corpus tests](AGENTS.md#corpus-tests) for interpreter stages; use `-update` to refresh expected output when appropriate.
3. Commit and push to your fork, then open a Pull Request (PR).

   **Commit message guidelines:**
   - Separate subject from body with a blank line
   - Limit the subject line to 50 characters
   - Capitalize the subject line
   - Do not end the subject line with a period
   - Use the imperative mood (e.g., "Add X" not "Added X")
   - Wrap the body at 72 characters
   - Use the body to explain what and why vs. how

   See [How to Write a Git Commit Message](https://chris.beams.io/posts/git-commit/) for more.

4. If prompted, accept the Contributor License Agreement (CLA) when submitting your first PR.

## Propose changes

Start the discussion on the [Ballerina Discord](https://discord.com/invite/wAJYFbMrG2). For substantial changes, you may be asked to open a GitHub issue (e.g., labeled as a proposal) to continue the discussion.
