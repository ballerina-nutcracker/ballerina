# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/ftp` package, providing an FTP, FTPS and SFTP client and a polling listener that dispatches files to typed service handlers.

## [ftp](https://github.com/ballerina-platform/module-ballerina-ftp/blob/master/docs/spec/spec.md)

A client for files on an FTP, FTPS or SFTP server, and a listener that polls server directories and dispatches each file to a typed service handler.

Types: `ftp:Client` (client class), `ftp:Listener` (class), `ftp:Caller` (client class), `ftp:Service`, `ftp:ClientConfiguration`, `ftp:ListenerConfiguration`, `ftp:AuthConfiguration`, `ftp:SecureSocket`, `ftp:RetryConfig`, `ftp:CircuitBreakerConfig`, `ftp:ServiceConfiguration`, `ftp:FileInfo`, `ftp:WatchEvent`, `ftp:Error` and its subtypes (`ConnectionError`, `FileNotFoundError`, `FileAlreadyExistsError`, `InvalidConfigError`, `ServiceUnavailableError`, `ContentBindingError`, `AllRetryAttemptsFailedError`, `CircuitBreakerOpenError`).

| Method | Notes |
|---|---|
| `new ftp:Client(config)` | FTP; explicit or implicit FTPS with PEM or PKCS12 trust material; SFTP with password, keyboard-interactive or private-key authentication and `known_hosts` verification. Proxies are not yet supported |
| `getBytes` / `getText` / `getBytesAsStream` | Whole-file and streamed reads |
| `getJson(path, targetType?)` / `getXml(path, targetType?)` | JSON bound to `json` or record types (with `laxDataBinding`); XML bound to `xml`. CSV reads and XML record binding are not yet supported |
| `putBytes` / `putText` / `putJson` / `putXml` / `putCsv` / `putBytesAsStream` / `putCsvAsStream` | `OVERWRITE` or `APPEND`; `putXml` takes `xml` values |
| `mkdir` / `rmdir` / `delete` / `rename` / `move` / `copy` / `exists` / `size` / `list` / `isDirectory` / `close` | Parent directories are created for writes, renames and copies; `rmdir` and `delete` fail on a non-empty folder, and `rmdir` on a file |
| `get` / `put` / `append` | Deprecated untyped operations, including ZIP compression for `put` |
| `retryConfig` / `circuitBreaker` | Exponential-backoff retry of whole-file reads; a rolling-window circuit breaker over all client operations except the deprecated `get`, `put` and `append` |
| `new ftp:Listener(config)`, `attach` / `detach`, `start` / `gracefulStop` / `immediateStop`, `poll` | Polls each `@ftp:ServiceConfig` path (or the deprecated listener-level path) with file name patterns, age filters and dependency conditions |
| `onFile` / `onFileText` / `onFileJson` / `onFileXml`, `onFileDelete` / `onFileDeleted`, `onFileChange`, `onError` | Content handlers take the content, an optional `ftp:FileInfo` and an optional `ftp:Caller`. `onFileCsv` and `@ftp:FunctionConfig` post-processing actions are not yet supported |
