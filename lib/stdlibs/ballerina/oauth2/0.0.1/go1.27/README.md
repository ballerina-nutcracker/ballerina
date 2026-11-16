# Ballerina OAuth2 Library

## Overview

This module provides a framework for interacting with OAuth2 authorization servers as specified in [RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749) and [RFC 7662](https://datatracker.ietf.org/doc/html/rfc7662). Its client OAuth2 provider obtains and refreshes access tokens with the client credentials, password, refresh token and JWT bearer grants, and its listener OAuth2 provider validates tokens against an introspection endpoint; the `http` module and other network modules use both for authentication. The Go Native Interpreter currently supports the client OAuth2 provider with all four grant types.

## Key Functionalities

- Obtain access tokens with the client credentials, password, refresh token and JWT bearer grant types through `oauth2:ClientOAuth2Provider`.
- Cache an access token until it expires (less the configured clock skew) and refresh it with the cached or configured refresh token.
- Send client credentials in the `Authorization` header or the request body, with scopes, optional parameters, custom headers and a custom payload.
- Reach token endpoints over HTTPS, trusting a certificate file and presenting a client certificate.

## Examples

```ballerina
import ballerina/io;
import ballerina/oauth2;

public function main() returns error? {
    oauth2:ClientOAuth2Provider provider = new ({
        tokenUrl: "https://localhost:9445/oauth2/token",
        clientId: "client-id",
        clientSecret: "client-secret",
        scopes: ["read", "write"],
        clientConfig: {secureSocket: {cert: "/path/to/public.crt"}}
    });
    string token = check provider.generateToken();
    io:println(token);
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
| Module error type | Supported | `oauth2:Error` is a `distinct error` |
| Global endpoint timeouts | Partially Supported | `globalConnectTimeout` and `globalReqTimeout` are declared as `public final decimal` with the 15 and 30 second defaults, because `configurable` variables are not supported yet (#781); `DEFAULT_CONNECT_TIMEOUT` and `DEFAULT_REQ_TIMEOUT` are supported |
| Endpoint client configuration | Partially Supported | `oauth2:ClientConfiguration` with `customHeaders`, `customPayload`, `connectTimeout`, `reqTimeout` and `oauth2:SecureSocket` (`disable`, a certificate file in `cert`, an `oauth2:CertKey` with an unencrypted key). A `crypto:TrustStore`, a `crypto:KeyStore` or an encrypted key returns `oauth2:Error` until shared PKCS#12 decoding lands (#1338). `httpVersion` is accepted, but requests use HTTP/1.1. The `clientConfig` field defaults call a helper instead of `{}` (#1325), with the same value |
| Client credentials grant | Supported | `oauth2:ClientCredentialsGrantConfig` |
| Password grant | Supported | `oauth2:PasswordGrantConfig`, refreshing with `oauth2:RefreshConfig` or `INFER_REFRESH_CONFIG` |
| Refresh token grant | Supported | `oauth2:RefreshTokenGrantConfig` |
| JWT bearer grant | Supported | `oauth2:JwtBearerGrantConfig` |
| Credential bearer | Supported | `oauth2:CredentialBearer` with `AUTH_HEADER_BEARER` and `POST_BODY_BEARER` |
| Client OAuth2 provider | Supported | `oauth2:ClientOAuth2Provider`; `init` requests the first token and panics if that fails |
| Listener OAuth2 provider | Not Yet Supported | `oauth2:ListenerOAuth2Provider`, `oauth2:IntrospectionConfig` and `oauth2:IntrospectionResponse` are not declared yet |

### Notable Behavioural Changes

- **A restricted custom header returns an error.** jBallerina panics with `IllegalArgumentException` when `customHeaders` holds a header Java's HTTP client restricts (`Connection`, `Content-Length`, `Expect`, `Host`, `Upgrade`); the Go-native version returns an `oauth2:Error` caused by `Failed to send the request to the endpoint. restricted header name: "<name>"` (<https://github.com/ballerina-platform/ballerina-library/issues/9715>).
- **Endpoint and JSON errors carry Go's error text in their cause.** The cause of a failed request (e.g. `connect: connection refused`, or a request timeout) or of an unparsable response holds the Go library's message instead of the JVM's (`null`, `request timed out`, `FromJsonStringError`); the outer `oauth2:Error` messages are unchanged.
