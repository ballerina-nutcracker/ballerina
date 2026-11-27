# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/auth` package, providing Basic Auth credential handling for clients and listeners.

## [auth](https://github.com/ballerina-platform/module-ballerina-auth/blob/master/docs/spec/spec.md)

Types: `auth:ClientBasicAuthProvider` (class), `auth:ListenerFileUserStoreBasicAuthProvider` (class), `auth:ListenerBasicAuthProvider` (object type), `auth:CredentialsConfig`, `auth:FileUserStoreConfig`, `auth:UserDetails`, `auth:Error`.

| Method | Notes |
|---|---|
| `new auth:ClientBasicAuthProvider(credentialsConfig)` | Copies the username and password, so later changes to the caller's record have no effect |
| `ClientBasicAuthProvider.generateToken()` | Returns the Base64 encoding of `username:password`; an empty username or password returns `auth:Error` |
| `auth:extractUsernameAndPassword(credential)` | Decodes a Base64 `username:password` credential into a `[string, string]`; only the first colon separates the two, and a missing colon, empty username or empty password returns `auth:Error`. A credential that is not valid Base64 or not valid UTF-8 returns `auth:Error` with the decoding error as its cause |
| `ListenerBasicAuthProvider.authenticate(credential)` | Object type for custom listener providers; returns `auth:UserDetails\|auth:Error` |
| `new auth:ListenerFileUserStoreBasicAuthProvider(fileUserStoreConfig?)` | Accepts the empty `auth:FileUserStoreConfig` |
| `ListenerFileUserStoreBasicAuthProvider.authenticate(credential)` | An empty credential returns `auth:Error` (`Credential cannot be empty.`); any other credential returns the unconfigured-store `auth:Error` |

Not covered in this subset: users declared in `Config.toml` for the file user store, which needs `configurable` variables and tables, and the LDAP user store provider.
