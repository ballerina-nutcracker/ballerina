# Supported ballerina library features

Subset 3 extends the released [subset 2](subset2.md) with the `file` module.

## [file](https://github.com/ballerina-platform/module-ballerina-file/blob/master/docs/spec/spec.md)

File, directory, and path manipulation utilities.

| Feature | Notes |
|---|---|
| `create` / `remove` / `rename` / `copy` | Create, remove (non-recursive and recursive), rename/move, and copy (with `REPLACE_EXISTING`, `COPY_ATTRIBUTES`, `NO_FOLLOW_LINKS` options) files and directories |
| `getMetaData` | File size, modification time, permissions, and type |
| `readDir` | Read directory contents |
| `createTemp` / `createTempDir` | Create a temporary file / directory |
| `test` | Test file/directory properties: `EXISTS`, `IS_DIR`, `IS_SYMLINK`, `READABLE`, `WRITABLE` |
| `getCurrentDir` | Get the current working directory |
| `getAbsolutePath` / `isAbsolutePath` / `basename` / `parentPath` / `normalizePath` / `splitPath` / `joinPath` / `relativePath` | Cross-platform path manipulation |
| `Listener` / `Service` | Directory change listener: attaches a service whose `onCreate`/`onModify`/`onDelete` remote methods are dispatched on filesystem changes under the configured `path` (optionally `recursive`) |

`file:Error`'s `distinct` subtypes (`FileNotFoundError`, `PermissionError`, etc.)
are declared as plain type aliases of `Error` instead — they are structurally
identical at runtime, so `error is file:FileNotFoundError`-style checks don't
narrow. `file:Service` is likewise declared as a plain (non-`distinct`)
`service object {}` marker instead of jBallerina's `distinct service object
{}`. Unlike jBallerina, `gracefulStop()` closes the underlying OS watch
immediately (same as `immediateStop()`) rather than leaving it running until
process exit, and `attach()` returns its "at least one resource required"
validation error through its `error?` return type instead of throwing it.
