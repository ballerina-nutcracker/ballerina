# Ballerina File Library

## Overview
The `ballerina/file` module provides utilities for working with files, directories, and file paths. It covers file system operations (create, remove, rename, copy, metadata), directory listing, temporary file/directory creation, file property testing, and a comprehensive set of cross-platform path manipulation utilities.

## Key Functionalities

- Creating, removing, renaming, and copying files and directories
- Retrieving file metadata (size, modification time, permissions, type)
- Reading directory contents
- Creating temporary files and directories
- Testing file properties (existence, type, readability, writability)
- Path operations: absolute path resolution, basename, parent directory, join, split, normalize, and relative path computation

## Examples

```ballerina
import ballerina/file;
import ballerina/io;

public function main() returns error? {
    check file:create("/tmp/hello.txt");
    file:MetaData meta = check file:getMetaData("/tmp/hello.txt");
    io:println(meta.readable);   // true

    string joined = check file:joinPath("/tmp", "subdir", "file.txt");
    io:println(joined);           // /tmp/subdir/file.txt

    string rel = check file:relativePath("/tmp/a/b", "/tmp/a/b/c/d");
    io:println(rel);              // c/d
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
| Get current working directory | Supported | |
| Create directory (non-recursive and recursive) | Supported | |
| Remove file or directory (non-recursive and recursive) | Supported | |
| Rename or move a file or directory | Supported | |
| Create a file at a given path | Supported | |
| Retrieve file metadata (size, modification time, permissions, type) | Supported | |
| Read directory contents | Supported | |
| Copy file or directory with options | Supported | `REPLACE_EXISTING`, `COPY_ATTRIBUTES`, `NO_FOLLOW_LINKS` options supported |
| Create a temporary file | Supported | |
| Create a temporary directory | Supported | |
| Test file or directory properties | Supported | `EXISTS`, `IS_DIR`, `IS_SYMLINK`, `READABLE`, `WRITABLE` options supported |
| Retrieve absolute path | Supported | |
| Check whether a path is absolute | Supported | |
| Extract the base name of a path | Supported | |
| Extract the parent directory of a path | Supported | |
| Normalize a path | Supported | `CLEAN`, `SYMLINK`, and `NORMCASE` options supported |
| Split a path into components | Supported | |
| Join path components | Supported | |
| Compute relative path between two paths | Supported | |
| File event types and constants | Supported | `FileEvent` record, `DirOption`, `CopyOption`, `TestOption`, `NormOption` constants all defined |
| Directory change listener and file watcher service | Cannot Support | Requires `distinct service object` types and a start/stop service lifecycle, neither of which is supported in this interpreter |

### Notable Behavioural Changes

- **`distinct` error types flattened.** jBallerina declares each error type (e.g. `FileNotFoundError`, `PermissionError`) as a `distinct` subtype of `file:Error`, allowing precise `is`-checks. The Go-native version declares them as plain type aliases of `Error` — they are structurally identical at runtime. Code that uses `error is file:FileNotFoundError` to distinguish error kinds will not work as expected.
- **Path separator detection on Windows.** `isWindows` is determined at startup by checking whether the `OS` environment variable is set. On non-standard Windows environments where this variable is absent the path functions will behave as on POSIX.
