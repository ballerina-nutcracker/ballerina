# Ballerina FTP Library

## Overview

The `ballerina/ftp` library reads and writes files on a remote FTP, FTPS (FTP over TLS) or SFTP (file transfer over SSH) server with an `ftp:Client`, watches a directory on such a server with an `ftp:Listener` that polls it and dispatches each file to a typed handler of an `ftp:Service`, and gives handlers an `ftp:Caller` to act on the server. It also binds file content to JSON, XML, CSV and record types, retries failed reads, and guards the client with a circuit breaker. The Go Native Interpreter supports the client over FTP, FTPS and SFTP with text, byte, JSON and XML content, streaming reads and writes, retry and the circuit breaker, and the polling listener with format-specific handlers, `onFileChange`, deletion handlers, `onError` and the caller; CSV reading, XML and CSV record binding, post-processing actions, proxies and distributed coordination are not yet supported.

## Key Functionalities

- Connect over FTP, explicit or implicit FTPS (PEM or PKCS12 trust material, PKCS12 client key store) and SFTP (password, keyboard-interactive and private-key authentication, `known_hosts` verification)
- Read files as bytes, text, JSON bound to `json` or a record type (`laxDataBinding` supported), XML, or a byte stream (`getBytes`, `getText`, `getJson`, `getXml`, `getBytesAsStream`)
- Write bytes, text, JSON, XML and CSV, from values or streams, overwriting or appending (`putBytes`, `putText`, `putJson`, `putXml`, `putCsv`, `putBytesAsStream`, `putCsvAsStream`)
- Manage files and directories (`mkdir`, `rmdir`, `delete`, `rename`, `move`, `copy`, `exists`, `size`, `list`, `isDirectory`)
- The deprecated untyped `get`, `put` (with ZIP compression) and `append`
- Retry failed whole-file reads with exponential backoff (`RetryConfig`) and stop calling a failing server with a circuit breaker (`CircuitBreakerConfig`)
- Poll directories with an `ftp:Listener`, per service through `@ftp:ServiceConfig` or through the deprecated listener-level fields, with file name patterns, age filters and dependency conditions
- Dispatch files to `onFile`, `onFileText`, `onFileJson` and `onFileXml` (with `ftp:FileInfo` and `ftp:Caller` parameters), deletions to `onFileDelete` or `onFileDeleted`, whole poll results to `onFileChange`, and binding failures to `onError`
- The typed error hierarchy rooted at `ftp:Error`

## Examples

```ballerina
import ballerina/ftp;
import ballerina/io;

type Order record {|
    int id;
    string item;
|};

public function main() returns error? {
    ftp:Client ftpClient = check new ({
        host: "127.0.0.1",
        port: 21,
        auth: {credentials: {username: "alice", password: "secret"}}
    });
    Order 'order = {id: 1, item: "pen"};
    check ftpClient->putJson("/orders/1.json", 'order);
    Order stored = check ftpClient->getJson("/orders/1.json");
    io:println(stored.item); // pen

    byte[]|ftp:Error missing = ftpClient->getBytes("/orders/2.json");
    io:println(missing is ftp:FileNotFoundError); // true
    check ftpClient->close();
}
```

## Go Native Interpreter Support Status

This library is currently being migrated to Go to support the Ballerina Native Interpreter. The table below outlines the current support level for various features of this library in the Go implementation.

Support Levels:

- **Supported**: Fully implemented and tested in the Go version.
- **Partially Supported**: Implemented but lacking some edge cases, options, or sub-features. (See comments).
- **Not Yet Supported**: Planned for migration, but not yet implemented.
- **Cannot Support**: Cannot be implemented in the Go version due to technical limitations or architectural differences. (See comments).

### Client

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Client initialization and configuration validation | Supported | A `port` outside 1–65535 and an invalid `circuitBreaker` fail with an `ftp:InvalidConfigError` before connecting, the intended behaviour confirmed in ballerina-platform/ballerina-library#9385; jBallerina 2.22.0 falls back to the default port and reports breaker errors as `ftp:Error` after connecting |
| FTP connections | Supported | Directories are listed with `LIST`, and a symbolic link takes its target's type, as commons-vfs does |
| FTPS connections | Partially Supported | Explicit and implicit modes, PEM `cert` paths and PKCS12 `crypto:TrustStore`/`crypto:KeyStore` files, and `verifyHostName` are supported. `dataChannelProtection` and JKS stores are not; see Notable Behavioural Changes |
| SFTP connections | Partially Supported | Password, keyboard-interactive and private-key (optionally encrypted) authentication, `preferredMethods` and `sftpSshKnownHosts` are supported; a `known_hosts` file that cannot be read fails the connection. `GSSAPI_WITH_MIC`, `sftpCompression` and `sftpSessionTimeout` have no effect; see Notable Behavioural Changes |
| Anonymous login and server root | Supported | `userDirIsRoot` maps `/` to the login directory |
| Connection and socket timeouts | Supported | `connectTimeout`, `ftpDataTimeout` and `ftpSocketTimeout` |
| Proxies | Not Yet Supported | A `proxy` configuration is validated, then rejected with `Proxy configuration is not yet supported` |
| Read whole files as bytes or text | Supported | `getBytes`, `getText` |
| Read JSON with data binding | Partially Supported | `getJson` binds to `json` and to record and array types with `cloneWithType` semantics, and honours `laxDataBinding`. Parse and binding error messages differ from `data.jsondata`'s until that library is ported |
| Read XML with data binding | Partially Supported | `getXml` binds to `xml`. Binding to a record type needs `data.xmldata` and returns an `ftp:Error` |
| Read CSV | Not Yet Supported | `getCsv` needs `data.csv` and returns an `ftp:Error`; `csvFailSafe` has no effect |
| Read CSV as a stream | Not Yet Supported | `getCsvAsStream` is not declared: its dependently-typed `stream<targetType, error?>` return is blocked by ballerina-nutcracker/ballerina#1389 |
| Read files as byte streams | Supported | `getBytesAsStream`, in chunks of up to 8192 bytes |
| Write bytes, text and JSON | Supported | `putBytes`, `putText`, `putJson`, with `OVERWRITE` or `APPEND` |
| Write XML | Partially Supported | `putXml` writes `xml` values. Converting a record needs `data.xmldata` and returns an `ftp:Error` |
| Write CSV | Supported | `putCsv` with `string[][]` or `record {}[]`; the header and quoting rules match jBallerina |
| Write from streams | Supported | `putBytesAsStream`, `putCsvAsStream`. A record stream written with `APPEND` gets no header row, as `putCsv` (ballerina-platform/ballerina-library#9385); jBallerina 2.22.0 writes one |
| File and directory management | Supported | `mkdir`, `rmdir`, `delete`, `rename`, `move`, `copy`, `exists`, `size`, `list`, `isDirectory`. `list` order is unspecified. `size` reports sizes of 2 GiB and more in full. `copy` of a folder creates an empty folder, as jBallerina does. `rmdir` and `delete` fail on a non-empty folder; see Notable Behavioural Changes |
| Deprecated untyped operations | Supported | `get`, `put` (including ZIP compression) and `append`. The ZIP archive name differs from jBallerina's; see Notable Behavioural Changes |
| Retry | Supported | `retryConfig` retries only transient failures of a whole-file read: 4xx server replies and broken or unreachable connections. Other errors, such as `FileNotFoundError` and `ContentBindingError`, return at once, as intended per ballerina-platform/ballerina-library#9385; jBallerina 2.22.0 retries every error |
| Circuit breaker | Supported | `circuitBreaker` with rolling windows, failure categories and the `HALF_OPEN` trial. A failure is counted also when its request spans buckets; see Notable Behavioural Changes |
| Close | Supported | `close` |

### Listener and Caller

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Listener initialization and lifecycle | Supported | `start`, `attach`, `detach`, `gracefulStop`, `immediateStop`, `poll`. Polling runs on a `task:Listener`. `detach` removes only that service and `attach` accepts a `string[]` name, as intended per ballerina-platform/ballerina-library#9385; jBallerina 2.22.0 stops polling for every service on `detach` and ignores `attach` with a `string[]` name |
| Service configuration | Supported | `@ftp:ServiceConfig` per service, or the deprecated listener-level `path`, `fileNamePattern`, `fileAgeFilter` and `fileDependencyConditions`. Configuration errors surface at `attach`, as in jBallerina (ballerina-platform/ballerina-library#9385) |
| File filtering | Supported | File name patterns, age filters and dependency conditions; patterns use Go regular expressions (see Notable Behavioural Changes) |
| Change detection | Supported | Every file that passes the filters is reported as added on every poll, as in jBallerina and as intended per ballerina-platform/ballerina-library#9385. A deletion is reported once, also when the directory has become empty. A symbolic link back into a folder being scanned is not followed. Reported paths and URIs carry no credentials; see Notable Behavioural Changes |
| Format-specific content handlers | Partially Supported | `onFile` (`byte[]` or a byte stream), `onFileText`, `onFileJson` and `onFileXml` with `xml` content are supported. `onFileCsv` and `onFileXml` with a record type need `data.csv`/`data.xmldata` and are rejected at `attach` |
| Per-handler routing and post-processing actions | Not Yet Supported | `@ftp:FunctionConfig` (`fileNamePattern`, `afterProcess`, `afterError`) needs method annotations at runtime, blocked by ballerina-nutcracker/ballerina#1360 and #1361 |
| Deletion handlers | Supported | `onFileDelete`, `onFileDeleted` |
| Whole-event handler | Supported | `onFileChange` with `ftp:WatchEvent` or `ftp:WatchEvent & readonly` |
| Binding error handler | Supported | `onError` receives the `ftp:ContentBindingError`. Reading its detail record from Ballerina is blocked by ballerina-nutcracker/ballerina#1443 |
| Service signature validation | Partially Supported | jBallerina rejects invalid service shapes at compile time through a compiler plugin; here the same checks fail `attach` with the plugin's messages, and parameter types are only checked for unsupported content types |
| Caller | Supported | All client operations except `close` |
| Distributed coordination | Not Yet Supported | `coordination` depends on `task:WarmBackupConfig`, which `ballerina/task` does not yet support |
| Observability | Not Yet Supported | Metrics and tracing need `ballerina/observe` |

### Errors

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Module-level error types | Supported | `ftp:Error` and its distinct subtypes. `ContentBindingError` is declared through a private named distinct type (ballerina-nutcracker/ballerina#617); its subtyping is unchanged |
| Reply-code error classification | Supported | Uses the code of the server reply; see Notable Behavioural Changes |

### Notable Behavioural Changes

- **FTPS data channel protection.** jBallerina negotiates the `dataChannelProtection` level (`CLEAR`, `PRIVATE`, `SAFE`, `CONFIDENTIAL`); the Go-native version always negotiates `PROT P` (`PRIVATE`) — the Go FTP client offers no other level.
- **FTPS key and trust stores.** jBallerina reads JKS and PKCS12 stores; the Go-native version reads PKCS12 stores and PEM certificate files only — Go has no JKS reader.
- **SFTP compression and GSSAPI.** jBallerina negotiates `sftpCompression` and can authenticate with `GSSAPI_WITH_MIC`; the Go-native version always transfers uncompressed and skips `GSSAPI_WITH_MIC` — the Go SSH library supports neither.
- **SFTP session timeout.** jBallerina applies `sftpSessionTimeout` as the SSH socket timeout; the Go-native version applies only `connectTimeout` to SFTP connections.
- **Regular expressions.** jBallerina compiles file name and dependency patterns as Java regular expressions; the Go-native version uses Go regular expressions, which reject look-around and back-references, and reports invalid patterns with Go's error text.
- **File permissions in FileInfo.** jBallerina reports SFTP readability, writability and executability for the connected user; the Go-native version reports the owner permission bits.
- **Handler failure output.** jBallerina prints a stack trace when a listener handler fails or a file cannot be read; the Go-native version prints `error: <message>` to stderr.
- **Deleting a non-empty folder.** jBallerina's `delete` returns `()` and leaves a non-empty folder in place; the Go-native version returns an `ftp:Error` — spec section 3.4 requires an error when the operation fails.
- **Credentials in file URIs.** jBallerina puts the plaintext password in `FileInfo.path`, `uri` and `rootURI` (listener handlers and `list`), in the deleted-file paths of `onFileDelete`, `onFileDeleted` and `WatchEvent.deletedFiles`, and in a listener `ContentBindingError`'s `filePath`; the Go-native version leaves the user and password out of these URIs (`ftp://host:port/path`), while `friendlyURI` and `publicURIString` keep `user:***@` (ballerina-platform/ballerina-library#9381, #9286).
- **Deletion detection.** jBallerina reports a deleted file once for each poll that had reported it, and reports no deletion while the watched directory is empty; the Go-native version reports each deletion once, including the last file of a directory (ballerina-platform/ballerina-library#9288, #9380).
- **Reply-code classification.** jBallerina takes the first 3-digit word of an error message as the FTP reply code, so `127` of `127.0.0.1` hides a `421` reply; the Go-native version uses the code of the server reply, and otherwise the last 3-digit word of the message (ballerina-platform/ballerina-library#9383).
- **ZIP archive name.** jBallerina's `put` with `ftp:ZIP` cuts the whole path at its last dot, so `/v1.2/report` uploads to `/v1.zip` and a path without a dot panics; the Go-native version replaces the extension of the file name, or appends `.zip` when it has none (ballerina-platform/ballerina-library#9382).
- **Removing a non-empty folder.** jBallerina's `rmdir` deletes a non-empty folder with everything inside it; the Go-native version returns an `ftp:Error` and leaves the folder in place — spec section 3.4 of 2.22.0 says `rmdir` fails on a non-empty folder (ballerina-platform/ballerina-library#9410).
- **Unreadable known_hosts file.** jBallerina logs a warning and connects without host key verification when the `sftpSshKnownHosts` file does not exist; the Go-native version fails the SFTP connection with an `ftp:Error` (ballerina-platform/ballerina-library#9421).
- **File size.** jBallerina's `size` narrows the size to 32 bits, so a 3 GiB file reports `-1073741824` while `list` reports `3221225472`; the Go-native version returns the full size from both (ballerina-platform/ballerina-library#9420).
- **Symbolic link loops in listener scans.** jBallerina follows a symbolic link back into the folder being scanned again at every level, so a watched folder holding such links reports no files; the Go-native version does not follow a linked folder that lists the same entries as a folder being scanned (ballerina-platform/ballerina-library#9422).
- **Circuit breaker failures across buckets.** jBallerina counts a failure in the bucket where the request ends but leaves the bucket where it started as the last one used, so the next window update clears a failure that ended in a later bucket and the circuit may never open; the Go-native version counts every failure (ballerina-platform/ballerina-library#9423).
- **Transport error causes.** jBallerina reports connection and transfer failures with commons-vfs and JSch messages; the Go-native version keeps the same outer messages and error types but the underlying cause text comes from the Go network, FTP and SSH libraries.
