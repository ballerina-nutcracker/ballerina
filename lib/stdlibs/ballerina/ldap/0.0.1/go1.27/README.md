# Ballerina LDAP Library

## Overview

LDAP (Lightweight Directory Access Protocol) is a vendor-neutral protocol for accessing and maintaining distributed directory information services. The Ballerina LDAP module provides the capability to connect, authenticate, and interact with directory servers — searching for entries, adding, modifying, renaming, comparing, and deleting them. The Go Native Interpreter currently supports the full `ldap:Client` operation surface (bind, add, delete, modify, rename, compare, search, searchWithType, close, isConnected) over plaintext or TLS, with certificate-file-based TLS trust; PKCS12 trust-store-based TLS trust is not yet supported.

## Key Functionalities

- Connect to a directory server with a simple bind (`hostName`, `port`, `domainName`, `password`), optionally over TLS with a PEM certificate file, host-name verification, and TLS version selection.
- Create, delete, and rename directory entries.
- Update entries by replacing attribute values.
- Compare an entry's attribute value against an assertion value.
- Retrieve a single entry, or search a subtree, with the result converted into a caller-specified record type or the generic `ldap:Entry` type.
- Special-case decoding of Active Directory `objectGUID`/`objectSid` binary attributes into their canonical string forms, and RFC 2849 base64 encoding of other binary/non-ASCII attribute values.

## Examples

```ballerina
import ballerina/ldap;
import ballerina/io;

public function main() returns error? {
    ldap:Client ldapClient = check new ({
        hostName: "localhost",
        port: 389,
        domainName: "cn=admin,dc=example,dc=com",
        password: "adminpassword"
    });

    string userDN = "cn=John Doe,dc=example,dc=com";
    ldap:LdapResponse addResult = check ldapClient->add(userDN, {
        "objectClass": ["top", "person"],
        "sn": "Doe",
        "cn": "John Doe"
    });
    io:println(addResult.resultCode);

    ldap:SearchResult result = check ldapClient->search("dc=example,dc=com", "(cn=John Doe)", ldap:SUB);
    io:println(result.entries);

    ldapClient->close();
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
| Client initialization with a plaintext simple bind | Supported | |
| Secure connection using a certificate file path | Supported | As in jBallerina, a present `clientSecureSocket` always selects TLS; its `enable` field has no effect. |
| Secure connection using a trust store | Cannot Support | Extracting every certificate from a PKCS12 trust store isn't implemented for this module; supplying a trust store value for `clientSecureSocket.cert` returns an `Error` at runtime. Use a PEM certificate file path instead. |
| TLS host-name verification and TLS version selection | Supported | `verifyHostName: false` skips only the host-name check; the certificate chain is still verified. `tlsVersions` accepts `TLSv1` (or `TLSv1.0`), `TLSv1.1`, `TLSv1.2` and `TLSv1.3`; an unrecognised entry returns an `Error`. |
| Adding an entry | Supported | |
| Deleting an entry | Supported | |
| Updating an entry | Supported | Every field is sent as an attribute replace, matching jBallerina; there is no attribute add/delete modification mode. |
| Renaming an entry | Supported | |
| Comparing an attribute value | Supported | |
| Retrieving a single entry | Supported | Attributes to fetch are inferred from the target record's field names when not given explicitly. A required field with no matching attribute, or an attribute value that can't be converted to its field type, returns an `Error`. |
| Searching a subtree | Supported | |
| Searching with a caller-specified result type | Supported | |
| Closing the connection and checking connection status | Supported | |
| Search scope | Supported | |
| Operation status codes | Supported | |
| Binary attribute decoding | Supported | `objectGUID`/`objectSid` are decoded to their canonical string forms; other values containing ASCII control characters or non-ASCII bytes are base64-encoded. |
| Search result references and controls | Partially Supported | The underlying Go LDAP client library merges every search-result-reference's referral URIs into one flat list without preserving per-reference message IDs or controls; this module synthesizes one `SearchReference` per referral URI with `messageId` set to `0` and `controls` empty. |
| Error type | Partially Supported | `distinct` error types are not yet supported, so `Error` is currently an alias for `error`. `error:detail()` is not yet implemented in this interpreter's `lang.error`, so `ErrorDetails.resultCode` is not retrievable structurally — the LDAP result code is still visible in `error:message()` text. |

### Notable Behavioural Changes

- **Corrected result-code strings for two operation statuses.** jBallerina derives `resultCode` from the underlying LDAP SDK's free-text display name, which for two codes doesn't actually match the declared `Status` enum literal (`StrongAuthRequired` renders as `"STRONG AUTH REQUIRED"` instead of `Status`'s `"STRONGER AUTH REQUIRED"`, and `NotAllowedOnNonLeaf` renders with a hyphen instead of `Status`'s space-separated `"NOT ALLOWED ON NON LEAF"`); the Go-native version emits the string that matches the declared `Status` enum literal for both codes.
- **Client-side connection failures map to `OTHER`.** LDAP result codes that only ever originate client-side (e.g. connection timeouts, connect failures) have no dedicated `Status` member in either implementation; the Go-native version reports these as `OTHER` rather than an arbitrary unmatched string.
- **TLS version constraints are connection-scoped.** jBallerina configures `tlsVersions` as a JVM-wide static setting shared by every LDAP connection in the process; the Go-native version scopes it to the individual connection, which is more correct but means concurrent connections with different `tlsVersions` no longer interfere with each other.
- **A control with no value renders as an empty string instead of failing.** jBallerina's control-to-record conversion unconditionally dereferences the control's value and throws if one is absent; the Go-native version treats a valueless control as an empty string.
- **`Client.init` takes a plain configuration record.** jBallerina declares this as `*ConnectionConfig config`, an included-record parameter that additionally allows named-argument-style construction at the call site (e.g. `check new (hostName = "x", port = 389, ...)`). This port uses a plain parameter instead (`ConnectionConfig config`, no included-record spread); call sites pass a record literal (`check new ({hostName: "x", port: 389, ...})`), matching `ballerina/tcp` and `ballerina/udp`'s existing workaround for the same underlying gap.
- **Server error messages include the LDAP result code.** jBallerina's error message is only the server's diagnostic text (e.g. `entry already exists`). The Go-native version reports the full LDAP result, e.g. `LDAP Result Code 68 "Entry Already Exists": entry already exists (ENTRY ALREADY EXISTS)`, because `error:detail()` isn't implemented yet and the message is the only place the result code is visible.
- **Typed binding converts attribute strings to scalar field types.** jBallerina never parses attribute values, so a target record with an `int`, `boolean`, `float` or `decimal` field panics with a `ConversionError`. The Go-native version parses the string into the field's type (e.g. `"7"` into `int`), and returns an `Error` when the value can't be parsed.

