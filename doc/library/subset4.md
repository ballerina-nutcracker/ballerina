# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/auth` package, providing Basic Auth credential handling for clients and listeners and authentication against an LDAP user store.

## [auth](https://github.com/ballerina-platform/module-ballerina-auth/blob/master/docs/spec/spec.md)

Types: `auth:ClientBasicAuthProvider` (class), `auth:ListenerFileUserStoreBasicAuthProvider` (class), `auth:ListenerLdapUserStoreBasicAuthProvider` (class), `auth:ListenerBasicAuthProvider` (object type), `auth:CredentialsConfig`, `auth:FileUserStoreConfig`, `auth:LdapUserStoreConfig`, `auth:SecureSocket`, `auth:UserDetails`, `auth:Error`.

| Method | Notes |
|---|---|
| `new auth:ClientBasicAuthProvider(credentialsConfig)` | Copies the username and password, so later changes to the caller's record have no effect |
| `ClientBasicAuthProvider.generateToken()` | Returns the Base64 encoding of `username:password`; an empty username or password returns `auth:Error` |
| `auth:extractUsernameAndPassword(credential)` | Decodes a Base64 `username:password` credential into a `[string, string]`; only the first colon separates the two, and a missing colon, empty username or empty password returns `auth:Error`. A credential that is not valid Base64 or not valid UTF-8 returns `auth:Error` with the decoding error as its cause |
| `ListenerBasicAuthProvider.authenticate(credential)` | Object type for custom listener providers; returns `auth:UserDetails\|auth:Error` |
| `new auth:ListenerFileUserStoreBasicAuthProvider(fileUserStoreConfig?)` | Accepts the empty `auth:FileUserStoreConfig` |
| `ListenerFileUserStoreBasicAuthProvider.authenticate(credential)` | An empty credential returns `auth:Error` (`Credential cannot be empty.`); any other credential returns the unconfigured-store `auth:Error` |
| `new auth:ListenerLdapUserStoreBasicAuthProvider(ldapUserStoreConfig)` | Connects to `connectionUrl` (`ldap://` or `ldaps://`, scheme case-insensitive, default ports 389 and 636; a space-separated list is tried in order) and binds as `connectionName`; panics with an `auth:Error` on a connection, TLS or bind failure, or a URL that is not an LDAP URL. `connectionTimeout` bounds the connect and `readTimeout` each LDAP request, both in exact fractional seconds |
| `ListenerLdapUserStoreBasicAuthProvider.authenticate(credential)` | Replaces every `?` in `userNameSearchFilter` with the escaped, trimmed username and searches each `#`-separated `userSearchBase` in turn, then binds as the first match. On success, returns the values of `groupNameAttribute` from the groups under each `groupSearchBase` whose `membershipAttribute` holds the user's DN (or the user's RDN value for `memberUid`, matched case-insensitively), escaped as a filter value, as `scopes`; a user with no groups has no `scopes`. An unknown user, a wrong password or a failed group lookup returns `auth:Error` with the LDAP failure as its cause |

TLS to an `ldaps://` server trusts the PEM or DER certificate file given in `secureSocket.cert`.

Not covered in this subset: users declared in `Config.toml` for the file user store, which needs `configurable` variables and tables, and a `crypto:TrustStore` as `secureSocket.cert`.
