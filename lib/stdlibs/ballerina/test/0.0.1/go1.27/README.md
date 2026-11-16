# Ballerina Test Library

## Overview

This module provides the `ballerina/test` annotations and assertion functions that back `bal test`: declaring, configuring, and grouping test functions, and asserting on values inside them. Test discovery and execution themselves are handled by this port's compiler plugin and `testrunner` package, not by this module's own `.bal` source — this module only supplies the public annotations, config types, and assertion/error API that test code is written against.

## Key Functionalities

- Mark a function as a test with `@test:Config`, controlling enablement, groups, dependencies, a data provider, `before`/`after` hooks, serial execution, and (via `minPassRate`/`runs`) evaluation-style repeated runs.
- Suite-wide and group-wide lifecycle hooks: `@test:BeforeSuite`, `@test:AfterSuite`, `@test:BeforeGroups`, `@test:AfterGroups`, `@test:BeforeEach`, `@test:AfterEach`.
- Assert on values with `assertTrue`, `assertFalse`, `assertEquals`, `assertNotEquals`, `assertExactEquals`, `assertNotExactEquals`, and fail a test directly with `assertFail`.

## Examples

```ballerina
import ballerina/test;

@test:BeforeSuite
function beforeSuite() {
}

@test:Config {
    groups: ["math"]
}
function testAddition() {
    test:assertEquals(2 + 2, 4, msg = "addition failed");
}

@test:Config {
    dataProvider: additionDataProvider
}
function testAdditionWithData(int a, int b, int expected) {
    test:assertEquals(a + b, expected);
}

function additionDataProvider() returns map<[int, int, int]> {
    return {
        "1": [1, 1, 2],
        "2": [2, 3, 5]
    };
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
| The Config annotation | Supported | enable, groups, dependsOn, dataProvider, before, after, serialExecution, minPassRate, runs. |
| The BeforeSuite / AfterSuite annotations | Supported | AfterSuiteConfig.alwaysRun honored. |
| The BeforeGroups / AfterGroups annotations | Supported | The group list value and AfterGroupsConfig.alwaysRun honored. |
| The BeforeEach / AfterEach annotations | Supported | Run once per test, including once per data-driven test (not per row). |
| assertTrue / assertFalse | Supported | |
| assertEquals / assertNotEquals | Supported | Structural equality via an ==-equivalent deep comparison. |
| assertExactEquals / assertNotExactEquals | Supported | Reference/exact-type equality. |
| assertFail | Supported | |
| TestError | Supported | A distinct error, raised by the assert functions above. |
| DataProviderReturnType | Supported | Map and array-of-tuple data-provider shapes. |
| Mocking (function and object mocking, mock stub generation) | Not Yet Supported | No mocking support of any kind is implemented; mock error types/constants are absent too. |
| The ANY wildcard constant | Not Yet Supported | jBallerina's wildcard mock-argument-matcher constant. |
| The mocking Error type | Not Yet Supported | jBallerina's mocking-related error type. |
| The MockConfig record | Not Yet Supported | Configuration record for the mock function. |

### Notable Behavioural Changes

- **No mocking support.** jBallerina's `ballerina/test` includes a function- and object-mocking framework (`test:mock`, `MockFunction`, `MockObject`, stub generation); none of it is implemented in this port. Test code that uses mocking will fail to compile.
