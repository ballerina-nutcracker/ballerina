# Ballerina OS Library

## Overview

This module provides operating-system interaction for Ballerina programs. It covers environment variable management, current-user queries, and subprocess execution. The Go Native Interpreter supports the full surface of the jBallerina `os` module.

## Key Functionalities

- Read, set, unset, and list environment variables with `getEnv`, `setEnv`, `unsetEnv`, and `listEnv`.
- Query the current user's name and home directory with `getUsername` and `getUserHome`.
- Spawn a subprocess with `exec` and interact with it through the `Process` object: wait for exit (`waitForExit`), capture stdout/stderr (`output`), and terminate the process (`exit`).

## Examples

```ballerina
import ballerina/io;
import ballerina/os;

public function main() returns error? {
    // Environment variables
    check os:setEnv("MY_KEY", "hello");
    string val = os:getEnv("MY_KEY");
    io:println(val);              // hello
    check os:unsetEnv("MY_KEY");

    map<string> env = os:listEnv();
    io:println(env.length() > 0); // true

    // User info
    io:println(os:getUsername() != ""); // true
    io:println(os:getUserHome() != ""); // true

    // Execute a subprocess
    os:Process p = check os:exec({value: "echo", arguments: ["world"]});
    int code = check p.waitForExit();
    io:println(code);             // 0

    byte[] out = check p.output();
    io:println(out.length() > 0); // true
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
| Read an environment variable | Supported | `getEnv(name)`. Returns empty string when variable is unset. |
| Set an environment variable | Supported | `setEnv(key, value)`. Validates that key is not empty or `"=="`. |
| Unset an environment variable | Supported | `unsetEnv(key)`. Validates that key is not empty. |
| List all environment variables | Supported | `listEnv()`. Returns a `map<string>` snapshot at call time. |
| Query the current user's name | Supported | `getUsername()`. Returns empty string if the OS query fails. |
| Query the current user's home directory | Supported | `getUserHome()`. Returns empty string if the OS query fails. |
| Execute a subprocess | Supported | `exec(command, *envProperties)`. Merges parent environment with any overrides passed via `envProperties`. |
| Wait for subprocess exit | Supported | `Process.waitForExit()`. Returns the exit code; non-zero for failure. |
| Read subprocess output | Supported | `Process.output(fileOutputStream)`. Reads stdout (default) or stderr after the process exits. |
| Terminate a subprocess | Supported | `Process.exit()`. Sends SIGKILL to the subprocess immediately. |
| Module-level error types | Partially Supported | `os:Error` and `os:ProcessExecError` declared as plain `error` aliases; `distinct` error subtypes not yet supported. |
| Exclusion guard on environment property keys | Supported | `never command?;` field in `EnvProperties` prevents `command` from being used as an environment variable key. |

### Notable Behavioural Changes

- **Environment mutations are process-wide.** jBallerina uses per-strand env maps for isolation; the Go-native version calls `os.Setenv` / `os.Unsetenv` directly, mutating the process-wide environment. This is safe for single-threaded Ballerina programs but not for concurrent strand execution.
