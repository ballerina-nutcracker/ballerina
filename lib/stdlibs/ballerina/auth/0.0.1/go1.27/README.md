# Ballerina Auth Library

## Overview

This module provides a framework for authentication and authorization based on the Basic Authentication scheme specified in [RFC 7617](https://datatracker.ietf.org/doc/html/rfc7617). It offers a client Basic Auth provider that generates a token from a username and password, and listener Basic Auth providers that authenticate received credentials against a file user store (users declared in `Config.toml`) or an LDAP user store. The `http` module and other network modules use these providers for declarative and imperative authentication. The Go Native Interpreter currently supports the client Basic Auth provider, credential extraction, the listener Basic Auth provider object type, the LDAP user store provider (with TLS trust from a certificate file) and an unconfigured file user store.

## Key Functionalities

- Generate a Base64-encoded Basic Auth token from a username and password with `auth:ClientBasicAuthProvider`.
- Extract the username and password from a Base64-encoded `username:password` credential.
- Implement custom listener Basic Auth providers against the `auth:ListenerBasicAuthProvider` object type.
- Authenticate credentials against an LDAP directory with `auth:ListenerLdapUserStoreBasicAuthProvider`, returning the user's LDAP groups as scopes, over `ldap://` or `ldaps://`.
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
| LDAP user store Basic Auth provider | Supported | `auth:ListenerLdapUserStoreBasicAuthProvider`, `auth:LdapUserStoreConfig`. As in jBallerina, `domainName`, `userEntryObjectClass`, `userNameAttribute`, `userNameListFilter`, `groupEntryObjectClass`, `groupNameSearchFilter` and `userRolesCacheEnabled` are accepted but have no effect. `connectionPoolingEnabled` is ignored: each `authenticate` binds the user on a new connection, where jBallerina pools these connections |
| LDAP over TLS | Partially Supported | `auth:SecureSocket` with a PEM or DER certificate file in `cert` is supported; a `crypto:TrustStore` makes `init` panic with a not-yet-supported error until shared PKCS#12 decoding lands (#1338) |

### Notable Behavioural Changes

- **LDAPS uses the configured certificate.** jBallerina loads `secureSocket.cert` but verifies an `ldaps://` server against the JVM default trust store, so a server signed by the configured CA fails the handshake; the Go-native version trusts the configured certificate (<https://github.com/ballerina-platform/ballerina-library/issues/9706>).
- **LDAP timeouts keep their fractional part.** jBallerina truncates `connectionTimeout` and `readTimeout` to whole seconds, so a value below one second disables the timeout; the Go-native version applies the configured value exactly (<https://github.com/ballerina-platform/ballerina-library/issues/9707>).
- **A user DN ending in a backslash is supported.** jBallerina panics with `StringIndexOutOfBoundsException` while building the group filter for such a DN; the Go-native version escapes the trailing backslash and returns the user's groups (<https://github.com/ballerina-platform/ballerina-library/issues/9708>).
- **LDAP connection and file errors carry Go's error text.** A network or certificate-file failure during `init` panics with the Go platform message (e.g. `dial tcp 127.0.0.1:389: connect: connection refused`) instead of the JVM's (`Connection refused (connect failed)`), and a network failure or timeout in `authenticate` has go-ldap's message as its cause (e.g. `LDAP Result Code 200 "Network Error": ldap: connection timed out`); LDAP result errors keep jBallerina's `[LDAP: error code N - message]` form.
- **Membership values are escaped in the group filter.** jBallerina inserts the user's DN into the group filter with any `*` left unescaped, and a `memberUid` value without escaping, so a user whose name is `*` matches every group with a `memberUid` and receives its scopes; the Go-native version escapes both as RFC 4515 filter values (<https://github.com/ballerina-platform/ballerina-library/issues/9717>).
- **`membershipAttribute` matches `memberUid` case-insensitively.** jBallerina uses the user's RDN value only when `membershipAttribute` is exactly `memberUid`, so `memberuid` searches for the user's DN and finds none of the user's posix groups; the Go-native version compares the attribute name case-insensitively, as LDAP does (<https://github.com/ballerina-platform/ballerina-library/issues/9718>).
- **A backslash before `*` in the user's DN is kept in the group filter.** jBallerina escapes each RDN of the user's DN with the rule it uses for usernames, where `\*` stands for a literal `*`, so for a DN such as `uid=foo\\*,ou=Users,dc=example,dc=com` it drops the backslash, searches for a different DN and gives the user none of their groups; the Go-native version escapes the DN text as an RFC 4515 filter value as it is (<https://github.com/ballerina-platform/ballerina-library/issues/9726>).
