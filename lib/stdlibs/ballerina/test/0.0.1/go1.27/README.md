# Ballerina Test Library

## Overview

The `ballerina/test` library provides the annotation and assertions used by
`bal test`. A function annotated with `@test:Config` is discovered by the CLI
and executed against a live runtime.

## Key Functionalities

- Declare a test function with `@test:Config`, optionally disabling it (`enable`)
  or attaching `before`/`after` hooks.
- Fail the running test with `test:assertFail`.

## Examples

```ballerina
import ballerina/test;

@test:Config {}
function testSomething() {
    if 1 + 1 != 2 {
        test:assertFail("arithmetic is broken");
    }
}
```

## Go Native Interpreter Support Status

This library is currently being migrated to Go to support the Ballerina Native Interpreter. The table below outlines the current support level for various features of this library in the Go implementation.

Support Levels:

- **Supported**: Fully implemented and tested in the Go version.
- **Partially Supported**: Implemented but lacking some edge cases, options, or sub-features. (See comments).
- **Not Yet Supported**: Planned for migration, but not yet implemented.
- **Cannot Support**: Cannot be implemented in the Go version due to technical limitations or architectural differences. (See comments).

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| `@test:Config` with `enable`, `before`, `after` | Supported | `enable` must be a boolean literal. |
| `test:assertFail` | Supported | Marks the running test failed; under `bal run` it is a plain panic. |
| Other assertions (`assertEquals`, `assertTrue`, ...) | Not Yet Supported | |
| `groups`, `dependsOn`, data providers, mocking | Not Yet Supported | |
| Test reports and code coverage | Not Yet Supported | |
