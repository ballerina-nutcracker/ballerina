# Ballerina Auth Library

## Overview

This module provides a framework for authentication and authorization based on the Basic Authentication scheme specified in [RFC 7617](https://datatracker.ietf.org/doc/html/rfc7617). It offers a client Basic Auth provider that generates a token from a username and password, and listener Basic Auth providers that authenticate received credentials against a file user store (users declared in `Config.toml`) or an LDAP user store. The `http` module and other network modules use these providers for declarative and imperative authentication. The Go Native Interpreter currently supports the client Basic Auth provider, credential extraction, the listener Basic Auth provider object type and an unconfigured file user store.

## Key Functionalities

- Generate a Base64-encoded Basic Auth token from a username and password with `auth:ClientBasicAuthProvider`.
- Extract the username and password from a Base64-encoded `username:password` credential.
- Implement custom listener Basic Auth providers against the `auth:ListenerBasicAuthProvider` object type.
- Report authentication failures as `auth:Error`, a `distinct` error type.

## Examples

```ballerina
import ballerina/auth;
import ballerina/io;

public function main() returns error? {
    auth:ClientBasicAuthProvider provider = new ({username: "alice", password: "s3cret"});
    string token = check provider.generateToken();
    io:println(token); // YWxpY2U6czNjcmV0

    [string, string] credentials = check auth:extractUsernameAndPassword(token);
    io:println(credentials[0], " ", credentials[1]); // alice s3cret
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
| Module error type | Supported | `auth:Error` is a `distinct error` |
| Authenticated user details | Supported | `auth:UserDetails` |
| Basic Auth credential extraction | Supported | `extractUsernameAndPassword`; the cause of a Base64 or UTF-8 decoding error carries this interpreter's langlib message |
| Client Basic Auth provider | Supported | `auth:ClientBasicAuthProvider`, `auth:CredentialsConfig` |
| Listener Basic Auth provider object type | Supported | `auth:ListenerBasicAuthProvider` |
| File user store Basic Auth provider | Partially Supported | `auth:ListenerFileUserStoreBasicAuthProvider` and `auth:FileUserStoreConfig` are declared, but users cannot be declared in `Config.toml` because `configurable` variables (#781) and tables (#294) are not supported yet, so `authenticate` always returns the unconfigured-store error for a non-empty credential |
| Use alongside an explicit lang.array or lang.string import | Partially Supported | A program that imports `ballerina/auth` and also imports `ballerina/lang.'array` or `ballerina/lang.'string` explicitly, the lang libraries the auth module imports, fails to compile the auth module with `Unknown import: ballerina/lang.array` (or `lang.string`), because the bundled module can be compiled before that lang library (ballerina-nutcracker/ballerina#1678). Other explicit lang library imports, and the predeclared `array:` and `string:` prefixes used without an import, work |
| LDAP user store Basic Auth provider | Not Yet Supported | `auth:ListenerLdapUserStoreBasicAuthProvider`, `auth:LdapUserStoreConfig` and `auth:SecureSocket` are not declared yet |

### Notable Behavioural Changes

There are **no** notable behavioural changes in the Go-native version compared to the original jBallerina implementation for the currently supported features.
