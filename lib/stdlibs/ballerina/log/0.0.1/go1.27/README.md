# Ballerina Log Library

## Overview

This module provides structured logging for Ballerina programs. The full jBallerina `log` module covers configurable log levels and formats (LOGFMT and JSON), multiple output destinations (stderr, stdout, rotating files), per-module level overrides, key-value pair annotations, sensitive data masking, a named `Logger` object API with child-logger support, and observability integration. The Go Native Interpreter supports the core module-level print functions with basic level filtering.

## Key Functionalities

- Print structured log messages at four severity levels using `printDebug`, `printInfo`, `printWarn`, and `printError`.
- Attach an optional `error` value to any log call via the `'error` named parameter.
- Attach arbitrary key-value pair annotations to any log call using rest-record syntax (e.g. `id = 845315, path = "/api"`).
- Level filtering at the default `INFO` level: `DEBUG` messages are silently suppressed; `INFO`, `WARN`, and `ERROR` messages are emitted.
- Log output is written to stderr in LOGFMT format: `time=<RFC3339> level=<LEVEL> module="" message="<msg>" [error=<err>] [key=value ...]`.

## Examples

```ballerina
import ballerina/log;

public function main() {
    log:printInfo("server started", port = 8080, host = "localhost");
    log:printWarn("connection slow", latency = 1500);

    error e = error("disk full");
    log:printError("write failed", 'error = e, path = "/var/log");

    // DEBUG is silently dropped at the default INFO level
    log:printDebug("this will not appear", id = 42);
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
| Print at DEBUG level | Supported | |
| Print at INFO level | Supported | |
| Print at WARN level | Supported | |
| Print at ERROR level | Supported | |
| Optional error parameter | Supported | Named parameter `'error` is supported. |
| Default level filtering | Supported | Hardcoded to `INFO`; `DEBUG` suppressed, `INFO`/`WARN`/`ERROR` emitted. |
| LOGFMT output format | Supported | Written to stderr. Format: `time=<RFC3339> level=<LEVEL> module="" message="<msg>"`. |
| JSON output format | Not Yet Supported | `JSON_FORMAT` enum constant declared; switching format requires `configurable` variable support. |
| Configurable log level | Not Yet Supported | Level is hardcoded to `INFO`; `configurable Level level` requires configurable variable support. |
| Configurable log format | Not Yet Supported | Format is hardcoded to LOGFMT; `configurable LogFormat format` requires configurable variable support. |
| Key-value pair annotations | Partially Supported | KV values are restricted to `anydata`; `Valuer` function values and `PrintableRawTemplate` values are not supported. |
| Per-module level overrides | Not Yet Supported | Requires `configurable table<Module>` support; tables not yet supported. |
| Multiple output destinations | Not Yet Supported | `configurable OutputDestination[] destinations` not supported; output is always written to stderr. |
| File output destination | Not Yet Supported | Requires `lock` statements and file I/O integration not yet implemented. |
| Log rotation | Not Yet Supported | Depends on file output support. |
| Logger object interface | Not Yet Supported | Requires `isolated` object support. Affects `root()`, `fromConfig`, `withContext`, and `LoggerRegistry`. |
| Child loggers | Not Yet Supported | Depends on `Logger` object support; `withContext` not implemented. |
| Logger registry | Not Yet Supported | Depends on `Logger` object support; `getLoggerRegistry` not implemented. |
| Sensitive data masking | Not Yet Supported | `@Sensitive` annotation and `toMaskedString` not implemented. |
| Template message support | Not Yet Supported | `PrintableRawTemplate` type requires template expression support not yet available. |
| Stack trace parameter | Not Yet Supported | Omitted from function signatures; stack trace access differs from JVM. |
| Observability integration | Not Yet Supported | `ballerina/observe` module not yet available; tracing context fields omitted from log records. |
| Deprecated output file function | Not Yet Supported | `setOutputFile` is deprecated in jBallerina and not implemented. |
| Module-level error type | Partially Supported | `log:Error` declared as a plain `error` alias; `distinct` error subtypes not yet supported. |

### Notable Behavioural Changes

- **Module name always empty.** jBallerina uses JVM `StackWalker` to detect the calling module name at runtime; the Go-native version has no equivalent mechanism, so `module=""` in all log records.
- **Error field format.** jBallerina serialises a full `FullErrorDetails` record (message, stack trace, cause chain) for the `error` field; the Go-native version formats the error as `error("message")` using the Ballerina `toBalString` representation of the error value.
