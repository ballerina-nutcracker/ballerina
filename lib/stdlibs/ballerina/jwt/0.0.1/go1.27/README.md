# Ballerina JWT Library

## Overview

This module provides a framework for issuing and validating JSON Web Tokens (JWT) as specified in [RFC 7519](https://datatracker.ietf.org/doc/html/rfc7519), signed as JSON Web Signatures ([RFC 7515](https://datatracker.ietf.org/doc/html/rfc7515)) with HMAC or RSA. It issues and decodes tokens, validates their signature (shared secret, public certificate, trust store or JWKS endpoint) and claims, and provides a client self-signed JWT auth provider and a listener JWT auth provider for the `http` module and other network modules. The Go Native Interpreter currently supports issuing and decoding tokens and the client self-signed JWT auth provider.

## Key Functionalities

- Issue a JWT signed with HMAC (`HS256`, `HS384`, `HS512`) using a shared secret, or with RSA (`RS256`, `RS384`, `RS512`) using a key store, a private key file or a `crypto:PrivateKey`, or issue an unsigned JWT.
- Decode a JWT into its `jwt:Header` and `jwt:Payload` without validating it.
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
| Validation | Not Yet Supported | `jwt:validate`, `jwt:ValidatorConfig` and `jwt:ValidatorSignatureConfig` are not declared yet |
| Listener JWT auth provider | Not Yet Supported | `jwt:ListenerJwtAuthProvider` is not declared yet |
| JWKS-based signature validation | Not Yet Supported | `jwt:ClientConfiguration`, `jwt:SecureSocket`, `jwt:CertKey` and `jwt:HttpVersion` are not declared yet |

### Notable Behavioural Changes

- **Malformed claim types return an error.** jBallerina panics with `TypeCastError` when a token's header or payload is not a JSON object, or when `typ`, `cty`, `kid`, `iss`, `sub`, `jti` or `aud` has an unexpected JSON type; the Go-native version returns a `jwt:Error` naming the field (<https://github.com/ballerina-platform/ballerina-library/issues/9709>).
- **Decoding errors carry Go's error text in their cause.** The cause of a Base64 URL or JSON decoding failure holds the Go library's message (e.g. `illegal base64 data at input byte 1`) instead of the JVM's (`Illegal base64 character 24`); the outer `jwt:Error` messages are unchanged.
