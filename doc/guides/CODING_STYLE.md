# Coding style

## Visibility and APIs

- Keep Go identifiers unexported unless callers outside the package need them or an exported API is explicitly requested.
- Remove unused helpers and superseded forwarding methods or interfaces. If backward compatibility requires retaining an old API, deprecate it and document its removal plan.
- Avoid interfaces with only one implementation unless they serve a real boundary.

## Initialization and state

- Constructors should initialize every field unless its zero value is intentional. Initialize map fields to empty maps.
- Establish invariants up front rather than relying on lazy initialization or repeated defensive checks. Do not add nil or zero checks that silently hide a broken invariant.
- Use a minimal enum for mutually exclusive states. For independent flags, prefer a compact bit set with explicit set and check methods.
- Prefer a field's zero value over a separate “is set” boolean when a safe representation is available, such as storing an index plus one.
- Pass required values directly or in a focused argument struct. Make ordered public operations return values consumed by the next operation instead of communicating through hidden shared state.
- Avoid storing values derived from other fields unless there is a clear performance reason. Document that reason, keep source fields private where possible, and update source and derived fields together.

## Shared behavior and helpers

- When structs share fields and methods, consider embedding a private base struct and implementing the shared methods on it.
- When functions differ only in how they access data, use an existing or minimal interface. When preparation differs, separate that preparation from the shared computation.
- Extract functions for meaningful operations. Inline shallow private helpers with one caller unless they perform data preparation or serve as interface accessors, parent proxies, or public inner-package proxies.
- When adding a shared helper, update existing callers that duplicate its logic unless there is a specific, documented reason to keep them separate.
- When related functions or types repeat a common name prefix, consider a focused package for them.

## Control flow and results

- Check whether optional input is present before calling a helper that requires it. Helpers should not silently succeed when their work does not apply unless handling absence is part of their contract.
- Return an `error` or an `ok` boolean alongside a value only when failure is possible. Remove failure signals that always indicate success and the dead checks at their call sites.
- Keep each switch case's handling inside that case or a helper it calls, rather than handling some cases after the switch.
- Trigger internal errors or panics at the failing call site rather than hiding them in helpers whose sole purpose is to fail.

## Comments

- Avoid comments that narrate code or restate names and types. Prefer clearer names and meaningful operations.
- Function documentation should describe behavior and the contract callers rely on: inputs, results, and preconditions. Keep implementation details, cost notes, and caller-specific design justifications out of function documentation.
- Field comments should explain non-obvious meaning, special behavior, restrictions, or invariants. Keep comments proportional to what they explain.
- Every lint suppression must include a comment explaining why it is necessary.

## Formatting and license headers

- Follow the [formatting and linting workflow](DEVELOPING.md#linting-and-formatting).
- Every Go and Ballerina source file must have the repository's Apache 2.0 license header; see [an existing source header](../../platform/pal/platform.go).
