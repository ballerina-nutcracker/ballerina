# Ballerina JWT Library

## Overview

This module provides a framework for issuing and validating JSON Web Tokens (JWT) as specified in [RFC 7519](https://datatracker.ietf.org/doc/html/rfc7519), signed as JSON Web Signatures ([RFC 7515](https://datatracker.ietf.org/doc/html/rfc7515)) with HMAC or RSA. It issues and decodes tokens, validates their signature (shared secret, public certificate, trust store or JWKS endpoint) and claims, and provides a client self-signed JWT auth provider and a listener JWT auth provider for the `http` module and other network modules. The Go Native Interpreter currently supports issuing, decoding and validating tokens (with a shared secret, a certificate file, a `crypto:PublicKey`, a trust store or a JWKS endpoint) and both auth providers; the JWKS client does not yet accept trust or key stores.

## Key Functionalities

- Issue a JWT signed with HMAC (`HS256`, `HS384`, `HS512`) using a shared secret, or with RSA (`RS256`, `RS384`, `RS512`) using a key store, a private key file or a `crypto:PrivateKey`, or issue an unsigned JWT.
- Decode a JWT into its `jwt:Header` and `jwt:Payload` without validating it.
- Validate a JWT's signature against a shared secret, a certificate file, a `crypto:PublicKey` or a trust store, and its `iss`, `sub`, `aud`, `jti`, `kid`, custom claims, `exp` and `nbf` (with clock skew).
- Validate RSA signatures with keys fetched from a JWKS endpoint, optionally preloading the keys into a cache.
- Authenticate inbound tokens with `jwt:ListenerJwtAuthProvider`, optionally caching validated tokens.
- Generate self-signed tokens for outbound requests with `jwt:ClientSelfSignedJwtAuthProvider`.
- Report failures as `jwt:Error`, a `distinct` error type.

## Examples

```ballerina
import ballerina/io;
import ballerina/jwt;

public function main() returns error? {
    string token = check jwt:issue({
        issuer: "wso2",
        username: "alice",
        audience: "ballerina",
        signatureConfig: {algorithm: jwt:HS256, config: "a-shared-secret"}
    });
    [jwt:Header, jwt:Payload] decoded = check jwt:decode(token);
    io:println(decoded[0].alg, " ", decoded[1].sub); // HS256 alice
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
| Module error type | Supported | `jwt:Error` is a `distinct error` |
| Signing algorithms | Supported | `jwt:SigningAlgorithm` and the `RS256`, `RS384`, `RS512`, `HS256`, `HS384`, `HS512` and `NONE` constants |
| Header and payload records | Supported | `jwt:Header`, `jwt:Payload`; claims serialize in insertion order, as jBallerina's `toJsonString` does |
| Issuing with a shared secret | Supported | `jwt:issue` with `HS256`, `HS384` or `HS512` and a `string` secret. An empty secret signs the token, because `crypto` accepts an empty HMAC key (#1677) |
| Issuing with an RSA key | Supported | `jwt:issue` with `RS256`, `RS384` or `RS512` and a `crypto:KeyStore`, a private key file (optionally encrypted) or a `crypto:PrivateKey` |
| Issuing unsigned tokens | Supported | `jwt:NONE`, or no `signatureConfig` |
| Decoding | Supported | `jwt:decode`; as in jBallerina, a token with `alg` `none` is rejected |
| Client self-signed JWT auth provider | Supported | `jwt:ClientSelfSignedJwtAuthProvider`, `jwt:IssuerConfig`, `jwt:IssuerSignatureConfig` |
| Validation | Supported | `jwt:validate`, `jwt:ValidatorConfig`, `jwt:ValidatorSignatureConfig` with `secret`, `certFile` or `trustStoreConfig`. As in jBallerina, custom claims are compared by their string form and an empty `signatureConfig` checks no signature |
| Listener JWT auth provider | Supported | `jwt:ListenerJwtAuthProvider`, with the validated-token cache from `cacheConfig` |
| JWKS-based signature validation | Partially Supported | `jwksConfig` with `url`, `cacheConfig` (preloaded by `jwt:ListenerJwtAuthProvider`) and `jwt:ClientConfiguration`. `jwt:SecureSocket` supports `disable`, a certificate file in `cert` and a `jwt:CertKey` with an unencrypted key; a `crypto:TrustStore`, a `crypto:KeyStore` or an encrypted key returns `jwt:Error` until shared PKCS#12 decoding lands (#1338). `httpVersion` is accepted, but requests use HTTP/1.1 |

### Notable Behavioural Changes

- **Malformed claim types return an error.** jBallerina panics with `TypeCastError` when a token's header or payload is not a JSON object, or when `typ`, `cty`, `kid`, `iss`, `sub`, `jti` or `aud` has an unexpected JSON type; the Go-native version returns a `jwt:Error` naming the field (<https://github.com/ballerina-platform/ballerina-library/issues/9709>).
- **A token without a signature segment validates when no signature is configured.** jBallerina panics with `IndexOutOfRange` when `jwt:validate` gets a two-segment token and the `ValidatorConfig` has no `signatureConfig`; the Go-native version skips the signature check, as it does for a three-segment token (<https://github.com/ballerina-platform/ballerina-library/issues/9710>).
- **A public key without a certificate is accepted.** jBallerina rejects a `crypto:PublicKey` given as `certFile` that carries no certificate with `Public key certificate validity period has passed.`; the Go-native version checks the validity period only when the key has a certificate (<https://github.com/ballerina-platform/ballerina-library/issues/9711>).
- **The JWKS client times out.** jBallerina's JWKS client has no timeout, so an endpoint that never answers blocks validation forever; the Go-native version gives up after 15 seconds connecting or 30 seconds waiting for the response, the defaults `ballerina/oauth2` uses (<https://github.com/ballerina-platform/ballerina-library/issues/9713>).
- **Malformed JWKS input returns an error.** jBallerina panics when the JWKS `keys` member is not an array or the JWKS URL is not a valid URI; the Go-native version returns a `jwt:Error` (`Failed to access 'keys' property as an array ...` or `Failed to create URI for the provided value ...`) (<https://github.com/ballerina-platform/ballerina-library/issues/9714>).
- **Decoding errors carry Go's error text in their cause.** The cause of a Base64 URL or JSON decoding failure holds the Go library's message (e.g. `illegal base64 data at input byte 1`) instead of the JVM's (`Illegal base64 character 24`), and so do JWKS request failures (e.g. `connect: connection refused`); the outer `jwt:Error` messages are unchanged.
