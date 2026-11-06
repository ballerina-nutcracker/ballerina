---
name: run-jballerina
description: Run Ballerina source with jBallerina to compare behaviour against this interpreter.
---

This repository also builds a CLI named `bal`; make sure the reference run uses jBallerina and record its version for the comparison. If no jBallerina installation is available, report the missing reference rather than treating this interpreter as the baseline.

Run the same source with jBallerina (`<jballerina-bal> run <file.bal>`) and this interpreter (`go run ./cli/cmd run <file.bal>`). Invoke jBallerina by an explicit executable path rather than a bare `bal`, which may resolve to this repository's CLI; confirm with `<jballerina-bal> version` that it reports a jBallerina distribution.

- For compilation errors, compare source locations; diagnostic wording may differ. An unimplemented or internal error does not count as the expected diagnostic.
- A missing or different set of diagnostics is a divergence, including when an earlier-stage error stops compilation before errors jBallerina reports.
- For runtime errors, compare the outer Ballerina error type and message. Underlying platform cause text may differ.
- For panics, also compare source locations in the corresponding Ballerina stack frames.
